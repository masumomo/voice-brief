package model

import (
	"time"
)

// Comment はイベントに紐づくコメント（スレッド返信、Issue/PRコメント等）を表す構造体
type Comment struct {
	Author    string    // 投稿者
	Text      string    // 本文
	Timestamp time.Time // 投稿時刻
}

// Event は正規化されたイベントモデル
type Event struct {
	ID         string            // Unique ID
	Source     string            // "slack" | "notion" | "github"
	Category   string            // "dev" | "biz" | "incident" | "ops" | "other"
	Timestamp  time.Time         // イベント発生時刻
	Title      string            // 短い見出し（50文字程度）
	Body       string            // 本文（全文）
	Summary    string            // 本文の要約（Summarizer等で生成）
	URL        string            // Direct Link
	Location   string            // Channel Name or DB Name
	Author     string            // User Name
	Importance int               // 0-100（フィルタ・並び替えに利用）
	Refs       map[string]string // 追加情報（channel_id, tags等）
	Comments   []Comment         // 関連コメント（Slackスレッド返信、GitHubコメント等）
}

// EventSource はイベントソースの定数
const (
	EventSourceSlack  = "slack"
	EventSourceNotion = "notion"
	EventSourceGitHub = "github"
)

// EventCategory はイベントカテゴリの定数
const (
	EventCategoryDev      = "dev"
	EventCategoryBiz      = "biz"
	EventCategoryIncident = "incident"
	EventCategoryOps      = "ops"
	EventCategoryOther    = "other"
)

// NewEvent はEventの新しいインスタンスを作成します
func NewEvent(source string) *Event {
	return &Event{
		Source:     source,
		Category:   EventCategoryOther,
		Refs:       make(map[string]string),
		Comments:   make([]Comment, 0),
		Importance: 50, // デフォルト値
	}
}

// AddComment はイベントにコメントを追加します
func (e *Event) AddComment(author, text string, timestamp time.Time) {
	e.Comments = append(e.Comments, Comment{
		Author:    author,
		Text:      text,
		Timestamp: timestamp,
	})
}

// GetCommentCount はコメント数を返します
func (e *Event) GetCommentCount() int {
	return len(e.Comments)
}

// GetDisplayText は表示用テキストを返します（Summaryがあればそれを、なければBodyを切り詰めて返す）
func (e *Event) GetDisplayText(maxLen int) string {
	if e.Summary != "" {
		return e.Summary
	}
	if len(e.Body) <= maxLen {
		return e.Body
	}
	return e.Body[:maxLen] + "..."
}

// Events はEventのスライス
type Events []*Event

// Len は sort.Interface の実装
func (e Events) Len() int {
	return len(e)
}

// Less は Importance の降順（大きい方が先）でソート
func (e Events) Less(i, j int) bool {
	return e[i].Importance > e[j].Importance
}

// Swap は sort.Interface の実装
func (e Events) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

// FilterByImportance は指定した重要度以上のイベントのみを返します
func (e Events) FilterByImportance(minImportance int) Events {
	filtered := make(Events, 0)
	for _, event := range e {
		if event.Importance >= minImportance {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

// FilterBySource は指定したソースのイベントのみを返します
func (e Events) FilterBySource(source string) Events {
	filtered := make(Events, 0)
	for _, event := range e {
		if event.Source == source {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

// FilterByCategory は指定したカテゴリのイベントのみを返します
func (e Events) FilterByCategory(category string) Events {
	filtered := make(Events, 0)
	for _, event := range e {
		if event.Category == category {
			filtered = append(filtered, event)
		}
	}
	return filtered
}
