package fetcher

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/masumomo/voice-brief/internal/config"
	"github.com/masumomo/voice-brief/internal/importance"
	"github.com/masumomo/voice-brief/internal/model"
	"github.com/slack-go/slack"
)

// SlackFetcher はSlackからイベントを取得します
type SlackFetcher struct {
	client    *slack.Client
	config    *config.SlackConfig
	userCache map[string]*slack.User // ユーザーIDをキーにしたキャッシュ
	cacheMu   sync.RWMutex           // キャッシュの排他制御
}

// NewSlackFetcher は新しいSlackFetcherを作成します
func NewSlackFetcher(cfg *config.SlackConfig) *SlackFetcher {
	return &SlackFetcher{
		client:    slack.New(cfg.Token),
		config:    cfg,
		userCache: make(map[string]*slack.User),
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

	authorName := f.getUserName(msg.User)
	event.Author = authorName

	// メンション先のユーザー名を抽出
	mentionedUsers := f.ExtractMentionedUsers(msg.Text)

	// メンションをDisplay nameに置換
	bodyText := f.ReplaceMentions(msg.Text)

	// メンション先がある場合は、誰から誰へかを明示
	if len(mentionedUsers) > 0 {
		event.Refs["mentioned_to"] = strings.Join(mentionedUsers, ", ")
		// 本文の先頭に「発信者→宛先」を付加（LLMが文脈を理解しやすいように）
		// 例: 「[発信者: Aさん → 宛先: Bさん] レビューお願いします」
		mentionPrefix := fmt.Sprintf("[発信者: %s → 宛先: %s] ", authorName, strings.Join(mentionedUsers, ", "))
		bodyText = mentionPrefix + bodyText
	}

	// タイトルは本文の先頭50文字（プレフィックスなし）
	event.Title = truncate(f.ReplaceMentions(msg.Text), 50)
	event.Body = bodyText

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

	// カテゴリはMultiFetcherのCategorizerで判定される

	return event
}

// getUserName はユーザーIDからユーザー名を取得します
func (f *SlackFetcher) getUserName(userID string) string {
	if userID == "" {
		return "Unknown"
	}

	user := f.getUser(userID)
	if user == nil {
		return userID
	}

	// Display name を優先、なければ Real name、それもなければ Name
	if user.Profile.DisplayName != "" {
		return user.Profile.DisplayName
	}
	if user.Profile.RealName != "" {
		return user.Profile.RealName
	}
	if user.Name != "" {
		return user.Name
	}
	return userID
}

// getUser はユーザーIDからユーザー情報を取得します（キャッシュ付き）
func (f *SlackFetcher) getUser(userID string) *slack.User {
	// まずキャッシュをチェック
	f.cacheMu.RLock()
	if user, ok := f.userCache[userID]; ok {
		f.cacheMu.RUnlock()
		return user
	}
	f.cacheMu.RUnlock()

	// APIから取得
	user, err := f.client.GetUserInfo(userID)
	if err != nil {
		// エラーの場合はnilを返す（キャッシュしない）
		return nil
	}

	// キャッシュに保存
	f.cacheMu.Lock()
	f.userCache[userID] = user
	f.cacheMu.Unlock()

	return user
}

// GetUserDisplayName はユーザーIDからDisplay nameを取得します（外部公開用）
func (f *SlackFetcher) GetUserDisplayName(userID string) string {
	return f.getUserName(userID)
}

// mentionRegex はSlackメンション形式にマッチする正規表現
var mentionRegex = regexp.MustCompile(`<@(U[A-Z0-9]+)>`)

// groupMentionRegex はSlackグループメンション形式にマッチする正規表現
// <!here>, <!channel>, <!everyone>, <!subteam^SXXXXX|@groupname> など
var groupMentionRegex = regexp.MustCompile(`<!([a-z]+)(?:\^[A-Z0-9]+)?(?:\|@?([^>]+))?>`)

// groupMentionNames はグループメンションの置換名
var groupMentionNames = map[string]string{
	"here":     "チャンネルメンバー(オンライン)",
	"channel":  "チャンネルメンバー全員",
	"everyone": "ワークスペース全員",
	"subteam":  "", // subteamはラベル部分から取得
}

// ExtractMentionedUsers はテキスト内のSlackメンションからユーザー名のリストを抽出します
func (f *SlackFetcher) ExtractMentionedUsers(text string) []string {
	matches := mentionRegex.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}

	// 重複を除去しつつ順序を保持
	seen := make(map[string]bool)
	var users []string
	for _, match := range matches {
		userID := match[1]
		if !seen[userID] {
			seen[userID] = true
			displayName := f.getUserName(userID)
			users = append(users, displayName)
		}
	}
	return users
}

// ReplaceMentions はテキスト内のSlackメンション（<@UXXXXX>）をDisplay nameに置換します
// グループメンション（<!here>, <!channel>, <!everyone>, <!subteam^...>）も置換します
func (f *SlackFetcher) ReplaceMentions(text string) string {
	// 個人メンションを置換
	result := mentionRegex.ReplaceAllStringFunc(text, func(match string) string {
		// <@U078XKCNW5V> から U078XKCNW5V を抽出
		userID := mentionRegex.FindStringSubmatch(match)[1]
		displayName := f.getUserName(userID)
		return displayName
	})

	// グループメンションを置換
	result = groupMentionRegex.ReplaceAllStringFunc(result, func(match string) string {
		submatches := groupMentionRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}

		mentionType := submatches[1] // here, channel, everyone, subteam など

		// subteamの場合はラベル部分を使用
		if mentionType == "subteam" && len(submatches) >= 3 && submatches[2] != "" {
			return submatches[2] // @groupname の部分
		}

		// 定義済みのグループメンションを置換
		if replacement, ok := groupMentionNames[mentionType]; ok && replacement != "" {
			return replacement
		}

		// 不明なグループメンションはラベル部分があればそれを使用
		if len(submatches) >= 3 && submatches[2] != "" {
			return submatches[2]
		}

		// ラベルがない場合はメンションタイプをそのまま返す
		return mentionType
	})

	return result
}

// applyFilters はフィルタを適用します
func (f *SlackFetcher) applyFilters(events model.Events) model.Events {
	// キーワード除外
	if len(f.config.Filters.ExcludeKeywords) > 0 {
		events = importance.FilterByKeywords(events, f.config.Filters.ExcludeKeywords)
	}

	// 短文除外
	if f.config.Filters.ExcludeShortMessages && f.config.Filters.MinLength > 0 {
		events = importance.FilterShortMessages(events, f.config.Filters.MinLength)
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

// fetchThreadReplies はスレッドの返信を取得して、親イベントのCommentsに追加します
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

	commentCount := 0

	// スレッドの返信をCommentsに追加
	for i, reply := range replies {
		if i == 0 {
			continue // 親メッセージをスキップ
		}

		// Bot投稿を除外
		if f.config.Filters.ExcludeBots && reply.BotID != "" {
			continue
		}

		parentEvent.AddComment(
			f.getUserName(reply.User),
			f.ReplaceMentions(reply.Text),
			parseSlackTimestamp(reply.Timestamp),
		)
		commentCount++
	}

	return commentCount, nil
}
