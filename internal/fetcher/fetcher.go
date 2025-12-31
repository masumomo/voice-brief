package fetcher

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/masumomo/voice-brief/internal/categorizer"
	"github.com/masumomo/voice-brief/internal/config"
	"github.com/masumomo/voice-brief/internal/model"
)

// Fetcher は複数ソースからイベントを並列取得するインターフェース
type Fetcher interface {
	Fetch(ctx context.Context, since time.Time) (model.Events, error)
}

// MultiFetcher は複数のFetcherを並列実行します
type MultiFetcher struct {
	config        *config.Config
	slackFetcher  *SlackFetcher
	notionFetcher *NotionFetcher
	githubFetcher *GitHubFetcher
	categorizer   categorizer.Categorizer
}

// NewMultiFetcher は新しいMultiFetcherを作成します
func NewMultiFetcher(cfg *config.Config) *MultiFetcher {
	mf := &MultiFetcher{
		config:        cfg,
		slackFetcher:  NewSlackFetcher(&cfg.Slack),
		notionFetcher: NewNotionFetcher(&cfg.Notion),
		categorizer:   categorizer.NewRuleCategorizer(),
	}

	// GitHub Fetcherは有効な場合のみ初期化
	if cfg.GitHub.Enabled {
		mf.githubFetcher = NewGitHubFetcher(&cfg.GitHub)
	}

	return mf
}

// SetCategorizer はCategorizerを設定します（将来のLLM対応用）
func (f *MultiFetcher) SetCategorizer(c categorizer.Categorizer) {
	f.categorizer = c
}

// Fetch はすべてのソースから並列でイベントを取得します
func (f *MultiFetcher) Fetch(ctx context.Context, since time.Time) (model.Events, error) {
	// 並列取得用のチャンネル
	type fetchResult struct {
		source string
		events model.Events
		err    error
	}

	// チャンネルサイズを動的に設定
	numSources := 2 // Slack + Notion
	if f.config.GitHub.Enabled {
		numSources++
	}
	results := make(chan fetchResult, numSources)
	eg, ctx := errgroup.WithContext(ctx)

	// Slack取得（並列）
	eg.Go(func() error {
		events, err := f.slackFetcher.Fetch(ctx, since)
		results <- fetchResult{
			source: "Slack",
			events: events,
			err:    err,
		}
		// Best Effort: エラーが発生しても nil を返して他の取得を継続
		return nil
	})

	// Notion取得（並列）
	eg.Go(func() error {
		events, err := f.notionFetcher.Fetch(ctx, since)
		results <- fetchResult{
			source: "Notion",
			events: events,
			err:    err,
		}
		// Best Effort: エラーが発生しても nil を返して他の取得を継続
		return nil
	})

	// GitHub取得（並列）
	if f.config.GitHub.Enabled && f.githubFetcher != nil {
		eg.Go(func() error {
			events, err := f.githubFetcher.Fetch(ctx, since)
			results <- fetchResult{
				source: "GitHub",
				events: events,
				err:    err,
			}
			// Best Effort: エラーが発生しても nil を返して他の取得を継続
			return nil
		})
	}

	// 並列取得の完了を待つ
	go func() {
		eg.Wait()
		close(results)
	}()

	// 結果を集約
	allEvents := make(model.Events, 0)
	hasError := false

	for result := range results {
		if result.err != nil {
			fmt.Printf("⚠️  警告: %s からの取得に失敗: %v\n", result.source, result.err)
			hasError = true
			continue
		}
		allEvents = append(allEvents, result.events...)
		fmt.Printf("✓ %s から %d 件のイベントを取得\n", result.source, len(result.events))
	}

	// 少なくとも1つのソースから取得できていればOK
	if len(allEvents) == 0 && hasError {
		return nil, fmt.Errorf("すべてのソースからの取得に失敗しました")
	}

	// カテゴリ判定
	if f.categorizer != nil {
		f.categorizer.CategorizeAll(allEvents)
	}

	fmt.Printf("✓ 合計 %d 件のイベントを取得しました\n", len(allEvents))
	return allEvents, nil
}

// FetchSlackOnly はSlackのみから取得します（デバッグ用）
func (f *MultiFetcher) FetchSlackOnly(ctx context.Context, since time.Time) (model.Events, error) {
	return f.slackFetcher.Fetch(ctx, since)
}

// FetchNotionOnly はNotionのみから取得します（デバッグ用）
func (f *MultiFetcher) FetchNotionOnly(ctx context.Context, since time.Time) (model.Events, error) {
	return f.notionFetcher.Fetch(ctx, since)
}

// TestAllConnections はすべてのAPI接続をテストします
func (f *MultiFetcher) TestAllConnections(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)

	// Slack接続テスト
	eg.Go(func() error {
		if err := f.slackFetcher.TestConnection(ctx); err != nil {
			return fmt.Errorf("Slack: %w", err)
		}
		fmt.Println("✓ Slack API接続成功")
		return nil
	})

	// Notion接続テスト
	eg.Go(func() error {
		if err := f.notionFetcher.TestConnection(ctx); err != nil {
			return fmt.Errorf("Notion: %w", err)
		}
		fmt.Println("✓ Notion API接続成功")
		return nil
	})

	// GitHub接続テスト（有効な場合のみ）
	if f.config.GitHub.Enabled && f.githubFetcher != nil {
		eg.Go(func() error {
			if err := f.githubFetcher.TestConnection(ctx); err != nil {
				return fmt.Errorf("GitHub: %w", err)
			}
			fmt.Println("✓ GitHub API接続成功")
			return nil
		})
	}

	// すべての接続テストが成功するまで待つ
	if err := eg.Wait(); err != nil {
		return fmt.Errorf("API接続テストに失敗: %w", err)
	}

	return nil
}
