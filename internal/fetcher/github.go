package fetcher

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v57/github"
	"github.com/masumomo/voice-brief/internal/config"
	"github.com/masumomo/voice-brief/internal/model"
)

// GitHubFetcher はGitHubからイベントを取得します
type GitHubFetcher struct {
	client *github.Client
	config *config.GitHubConfig
}

// NewGitHubFetcher は新しいGitHubFetcherを作成します
func NewGitHubFetcher(cfg *config.GitHubConfig) *GitHubFetcher {
	client := github.NewClient(nil)
	if cfg.Token != "" {
		client = client.WithAuthToken(cfg.Token)
	}

	return &GitHubFetcher{
		client: client,
		config: cfg,
	}
}

// TestConnection はGitHub APIへの接続をテストします
func (f *GitHubFetcher) TestConnection(ctx context.Context) error {
	_, _, err := f.client.Users.Get(ctx, "")
	if err != nil {
		return fmt.Errorf("GitHub API接続テストに失敗: %w", err)
	}
	return nil
}

// Fetch は指定期間のGitHub活動を取得します
func (f *GitHubFetcher) Fetch(ctx context.Context, since time.Time) (model.Events, error) {
	if !f.config.Enabled {
		return model.Events{}, nil
	}

	allEvents := make(model.Events, 0)

	for _, repo := range f.config.Repositories {
		// リポジトリのコミット取得
		commits, err := f.fetchCommits(ctx, repo, since)
		if err != nil {
			fmt.Printf("⚠️  警告: リポジトリ %s のコミット取得に失敗: %v\n", repo, err)
		} else {
			allEvents = append(allEvents, commits...)
		}

		// Issue/PR更新取得
		issues, err := f.fetchIssuesAndPRs(ctx, repo, since)
		if err != nil {
			fmt.Printf("⚠️  警告: リポジトリ %s のIssue/PR取得に失敗: %v\n", repo, err)
		} else {
			allEvents = append(allEvents, issues...)
		}
	}

	return allEvents, nil
}

// fetchCommits はリポジトリのコミットを取得します
func (f *GitHubFetcher) fetchCommits(ctx context.Context, repoPath string, since time.Time) (model.Events, error) {
	events := make(model.Events, 0)

	// リポジトリパスをパース（owner/repo形式）
	parts := strings.Split(repoPath, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("無効なリポジトリパス: %s (owner/repo形式で指定してください)", repoPath)
	}
	owner, repo := parts[0], parts[1]

	// コミット一覧を取得
	opts := &github.CommitsListOptions{
		Since: since,
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	// ユーザー名フィルタがある場合
	if f.config.Username != "" {
		opts.Author = f.config.Username
	}

	commits, _, err := f.client.Repositories.ListCommits(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("コミット一覧の取得に失敗: %w", err)
	}

	for _, commit := range commits {
		event := f.commitToEvent(commit, owner, repo)
		events = append(events, event)
	}

	return events, nil
}

// fetchIssuesAndPRs はIssueとPull Requestを取得します
func (f *GitHubFetcher) fetchIssuesAndPRs(ctx context.Context, repoPath string, since time.Time) (model.Events, error) {
	events := make(model.Events, 0)

	// リポジトリパスをパース
	parts := strings.Split(repoPath, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("無効なリポジトリパス: %s", repoPath)
	}
	owner, repo := parts[0], parts[1]

	// Issue一覧を取得（PRも含む）
	opts := &github.IssueListByRepoOptions{
		Since: since,
		State: "all", // open, closed, all
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	// ユーザー名フィルタ
	if f.config.Username != "" {
		opts.Creator = f.config.Username
	}

	issues, _, err := f.client.Issues.ListByRepo(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("Issue一覧の取得に失敗: %w", err)
	}

	for _, issue := range issues {
		// since以降に更新されたもののみ
		if issue.UpdatedAt.Before(since) {
			continue
		}

		event := f.issueToEvent(issue, owner, repo)
		events = append(events, event)
	}

	return events, nil
}

// commitToEvent はコミットをEventに変換します
func (f *GitHubFetcher) commitToEvent(commit *github.RepositoryCommit, owner, repo string) *model.Event {
	event := model.NewEvent(model.EventSourceGitHub)

	event.ID = commit.GetSHA()
	event.Timestamp = commit.GetCommit().GetAuthor().GetDate().Time
	event.Location = fmt.Sprintf("%s/%s", owner, repo)
	event.URL = commit.GetHTMLURL()

	// タイトルはコミットメッセージの1行目
	message := commit.GetCommit().GetMessage()
	lines := strings.Split(message, "\n")
	event.Title = truncate(lines[0], 100)

	// 本文はコミットメッセージ全体
	event.Body = message

	// 作成者
	if commit.GetAuthor() != nil {
		event.Author = commit.GetAuthor().GetLogin()
	} else if commit.GetCommit().GetAuthor() != nil {
		event.Author = commit.GetCommit().GetAuthor().GetName()
	}

	// Refs に追加情報
	event.Refs["repository"] = fmt.Sprintf("%s/%s", owner, repo)
	event.Refs["sha"] = commit.GetSHA()
	if commit.GetStats() != nil {
		event.Refs["additions"] = fmt.Sprintf("%d", commit.GetStats().GetAdditions())
		event.Refs["deletions"] = fmt.Sprintf("%d", commit.GetStats().GetDeletions())
	}

	// カテゴリはMultiFetcherのCategorizerで判定される

	return event
}

// issueToEvent はIssue/PRをEventに変換します
func (f *GitHubFetcher) issueToEvent(issue *github.Issue, owner, repo string) *model.Event {
	event := model.NewEvent(model.EventSourceGitHub)

	event.ID = fmt.Sprintf("%d", issue.GetNumber())
	event.Timestamp = issue.GetUpdatedAt().Time
	event.Location = fmt.Sprintf("%s/%s", owner, repo)
	event.URL = issue.GetHTMLURL()

	// PRかIssueか判定
	isPR := issue.IsPullRequest()
	prefix := "Issue"
	if isPR {
		prefix = "PR"
	}

	event.Title = fmt.Sprintf("%s #%d: %s", prefix, issue.GetNumber(), issue.GetTitle())
	event.Body = issue.GetBody()

	// 作成者
	if issue.GetUser() != nil {
		event.Author = issue.GetUser().GetLogin()
	}

	// Refs に追加情報
	event.Refs["repository"] = fmt.Sprintf("%s/%s", owner, repo)
	event.Refs["number"] = fmt.Sprintf("%d", issue.GetNumber())
	event.Refs["state"] = issue.GetState()
	event.Refs["comments"] = fmt.Sprintf("%d", issue.GetComments())

	if isPR {
		event.Refs["type"] = "pull_request"
	} else {
		event.Refs["type"] = "issue"
	}

	// ラベル情報
	labels := make([]string, 0)
	for _, label := range issue.Labels {
		labels = append(labels, label.GetName())
	}
	if len(labels) > 0 {
		event.Refs["labels"] = strings.Join(labels, ",")
	}

	// カテゴリはMultiFetcherのCategorizerで判定される

	return event
}
