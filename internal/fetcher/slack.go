package fetcher

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/masumomo/voice-brief/internal/config"
	"github.com/masumomo/voice-brief/internal/filter"
	"github.com/masumomo/voice-brief/internal/model"
	"github.com/slack-go/slack"
)

// SlackFetcher はSlackからイベントを取得します
type SlackFetcher struct {
	client     *slack.Client
	config     *config.SlackConfig
	calculator filter.ImportanceCalculator
}

// NewSlackFetcher は新しいSlackFetcherを作成します
func NewSlackFetcher(cfg *config.SlackConfig) *SlackFetcher {
	return &SlackFetcher{
		client:     slack.New(cfg.Token),
		config:     cfg,
		calculator: filter.NewRuleBasedCalculator(),
	}
}

// TestConnection はSlack APIへの接続をテストします
func (f *SlackFetcher) TestConnection(ctx context.Context) error {
	_, err := f.client.AuthTestContext(ctx)
	if err != nil {
		return fmt.Errorf("Slack API接続テストに失敗: %w", err)
	}
	return nil
}

// Fetch は指定期間のメッセージを取得します
func (f *SlackFetcher) Fetch(ctx context.Context, since time.Time) (model.Events, error) {
	allEvents := make(model.Events, 0)

	for _, channel := range f.config.Channels {
		events, err := f.fetchChannel(ctx, channel, since)
		if err != nil {
			// エラーログを出力するが、他のチャンネルの取得は継続（Best Effort）
			fmt.Printf("⚠️  警告: チャンネル %s (%s) の取得に失敗: %v\n", channel.Name, channel.ID, err)
			continue
		}
		allEvents = append(allEvents, events...)
	}

	// フィルタリング
	allEvents = f.applyFilters(allEvents)

	// 重要度計算
	filter.CalculateAll(allEvents, f.calculator)

	return allEvents, nil
}

// fetchChannel は単一チャンネルからメッセージを取得します
func (f *SlackFetcher) fetchChannel(ctx context.Context, channel config.ChannelConfig, since time.Time) (model.Events, error) {
	events := make(model.Events, 0)

	// oldest/latestはUnix timestampの文字列
	oldest := strconv.FormatInt(since.Unix(), 10)
	latest := strconv.FormatInt(time.Now().Unix(), 10)

	params := &slack.GetConversationHistoryParameters{
		ChannelID: channel.ID,
		Oldest:    oldest,
		Latest:    latest,
		Limit:     1000, // 最大取得数
	}

	history, err := f.client.GetConversationHistoryContext(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("チャンネル履歴の取得に失敗: %w", err)
	}

	for _, msg := range history.Messages {
		// Bot投稿の除外（設定による）
		if f.config.Filters.ExcludeBots && msg.BotID != "" {
			continue
		}

		// Join/Leaveメッセージの除外
		if msg.SubType == "channel_join" || msg.SubType == "channel_leave" {
			continue
		}

		// システムメッセージの除外
		if msg.SubType != "" && msg.SubType != "thread_broadcast" {
			continue
		}

		event := f.messageToEvent(&msg, channel)
		events = append(events, event)
	}

	return events, nil
}

// messageToEvent はSlackメッセージをEventに変換します
func (f *SlackFetcher) messageToEvent(msg *slack.Message, channel config.ChannelConfig) *model.Event {
	event := model.NewEvent(model.EventSourceSlack)

	event.ID = msg.Timestamp
	event.Timestamp = parseSlackTimestamp(msg.Timestamp)
	event.Location = channel.Name
	event.Author = f.getUserName(msg.User)

	// タイトルは本文の先頭50文字
	event.Title = truncate(msg.Text, 50)
	event.Body = msg.Text

	// URL生成（workspace ID は実際には取得する必要がある。簡易版として省略）
	event.URL = fmt.Sprintf("https://slack.com/archives/%s/p%s", channel.ID, strings.ReplaceAll(msg.Timestamp, ".", ""))

	// Refs に追加情報を格納
	event.Refs["channel_id"] = channel.ID
	event.Refs["channel_name"] = channel.Name
	if msg.ThreadTimestamp != "" {
		event.Refs["thread_ts"] = msg.ThreadTimestamp
		event.Refs["in_thread"] = "true"
	}
	if msg.ReplyCount > 0 {
		event.Refs["thread_count"] = strconv.Itoa(msg.ReplyCount)
	}

	// カテゴリの自動判定（簡易版）
	event.Category = f.detectCategory(msg.Text)

	return event
}

// getUserName はユーザーIDからユーザー名を取得します（簡易版）
func (f *SlackFetcher) getUserName(userID string) string {
	if userID == "" {
		return "Unknown"
	}

	// 本来はキャッシュして users.info APIを呼ぶべきだが、
	// v1.0では簡易的にユーザーIDをそのまま返す
	// TODO: Phase 1.1+ でユーザー名取得を実装
	return userID
}

// detectCategory はメッセージ内容からカテゴリを判定します
func (f *SlackFetcher) detectCategory(text string) string {
	lower := strings.ToLower(text)

	// Incident関連
	if containsAny(lower, []string{"障害", "エラー", "停止", "down", "error", "incident", "緊急"}) {
		return model.EventCategoryIncident
	}

	// Dev関連
	if containsAny(lower, []string{"pr", "pull request", "レビュー", "review", "deploy", "デプロイ", "リリース", "release"}) {
		return model.EventCategoryDev
	}

	// Biz関連
	if containsAny(lower, []string{"会議", "meeting", "ミーティング", "打ち合わせ", "定例"}) {
		return model.EventCategoryBiz
	}

	return model.EventCategoryOther
}

// applyFilters はフィルタを適用します
func (f *SlackFetcher) applyFilters(events model.Events) model.Events {
	// キーワード除外
	if len(f.config.Filters.ExcludeKeywords) > 0 {
		events = filter.FilterByKeywords(events, f.config.Filters.ExcludeKeywords)
	}

	// 短文除外
	if f.config.Filters.ExcludeShortMessages && f.config.Filters.MinLength > 0 {
		events = filter.FilterShortMessages(events, f.config.Filters.MinLength)
	}

	return events
}

// parseSlackTimestamp はSlackのタイムスタンプ（例: "1234567890.123456"）をtime.Timeに変換します
func parseSlackTimestamp(ts string) time.Time {
	parts := strings.Split(ts, ".")
	if len(parts) == 0 {
		return time.Now()
	}

	sec, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Now()
	}

	return time.Unix(sec, 0)
}

// truncate は文字列を指定長で切り詰めます
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// containsAny はいずれかの文字列が含まれているかチェックします
func containsAny(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}
