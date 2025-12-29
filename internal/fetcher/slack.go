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

		// スレッドがある場合は返信を取得
		if msg.ThreadTimestamp != "" && msg.ThreadTimestamp != msg.Timestamp {
			// このメッセージはスレッドの返信なので、親メッセージとして扱う
			replyCount, err := f.fetchThreadReplies(ctx, channel.ID, msg.ThreadTimestamp, event)
			if err != nil {
				// エラーでもスレッド取得失敗として継続
				fmt.Printf("⚠️  警告: スレッド返信の取得に失敗: %v\n", err)
			} else if replyCount > 0 {
				// スレッド数をRefsに記録
				event.Refs["thread_reply_count"] = strconv.Itoa(replyCount)
			}
		}

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

// fetchThreadReplies はスレッドの返信を取得して、親イベントに情報を追加します
func (f *SlackFetcher) fetchThreadReplies(ctx context.Context, channelID, threadTS string, parentEvent *model.Event) (int, error) {
	params := &slack.GetConversationRepliesParameters{
		ChannelID: channelID,
		Timestamp: threadTS,
		Limit:     100, // スレッド返信の最大取得数
	}

	replies, _, _, err := f.client.GetConversationRepliesContext(ctx, params)
	if err != nil {
		return 0, fmt.Errorf("スレッド返信の取得に失敗: %w", err)
	}

	// 最初のメッセージは親メッセージ自身なので除外
	if len(replies) <= 1 {
		return 0, nil
	}

	replyCount := len(replies) - 1

	// スレッドの返信をBodyに追加（簡易版：最初の3件のみ）
	threadSummary := make([]string, 0, 3)
	for i, reply := range replies {
		if i == 0 {
			continue // 親メッセージをスキップ
		}
		if i > 3 {
			break // 最初の3件のみ
		}

		// Bot投稿を除外
		if f.config.Filters.ExcludeBots && reply.BotID != "" {
			continue
		}

		threadSummary = append(threadSummary, fmt.Sprintf("  💬 %s", truncate(reply.Text, 100)))
	}

	if len(threadSummary) > 0 {
		parentEvent.Body = parentEvent.Body + "\n\nスレッド返信:\n" + strings.Join(threadSummary, "\n")
		if replyCount > 3 {
			parentEvent.Body += fmt.Sprintf("\n  ... 他%d件の返信", replyCount-3)
		}
	}

	return replyCount, nil
}
