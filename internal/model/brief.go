package model

import (
	"time"
)

// BriefType はブリーフィングのタイプ
type BriefType string

const (
	BriefTypeDaily  BriefType = "daily"
	BriefTypeWeekly BriefType = "weekly"
)

// Brief はブリーフィング出力モデル
type Brief struct {
	Type           BriefType // "daily" | "weekly"
	WindowStart    time.Time // 対象期間の開始
	WindowEnd      time.Time // 対象期間の終了
	ScriptText     string    // TTS用プレーンテキスト
	ScriptMarkdown string    // 保存用Markdown
	AudioPath      string    // 生成音声ファイルパス
	Items          Events    // 採用されたイベント一覧
	GeneratedAt    time.Time // 生成日時
}

// Section はブリーフィング原稿のセクション
type Section struct {
	Title string   // セクションタイトル（例: "今日の主な動き"）
	Items []string // セクション内のアイテム
}

// NewBrief は新しいBriefを作成します
func NewBrief(briefType BriefType, start, end time.Time) *Brief {
	return &Brief{
		Type:        briefType,
		WindowStart: start,
		WindowEnd:   end,
		Items:       make(Events, 0),
		GeneratedAt: time.Now(),
	}
}

// AddEvent はBriefにイベントを追加します
func (b *Brief) AddEvent(event *Event) {
	b.Items = append(b.Items, event)
}

// AddEvents は複数のイベントを追加します
func (b *Brief) AddEvents(events Events) {
	b.Items = append(b.Items, events...)
}

// GetEventCount はイベント数を返します
func (b *Brief) GetEventCount() int {
	return len(b.Items)
}

// GetEventsBySource は指定ソースのイベントを返します
func (b *Brief) GetEventsBySource(source string) Events {
	return b.Items.FilterBySource(source)
}

// GetEventsByCategory は指定カテゴリのイベントを返します
func (b *Brief) GetEventsByCategory(category string) Events {
	return b.Items.FilterByCategory(category)
}

// GetTopEvents は重要度順でトップN件のイベントを返します
func (b *Brief) GetTopEvents(n int) Events {
	if len(b.Items) <= n {
		return b.Items
	}
	return b.Items[:n]
}

// GetDurationString は対象期間を文字列で返します
func (b *Brief) GetDurationString() string {
	if b.Type == BriefTypeDaily {
		return b.WindowStart.Format("2006年01月02日")
	}
	// Weekly
	_, week := b.WindowStart.ISOWeek()
	return b.WindowStart.Format("2006年01月02日") + "〜" + b.WindowEnd.Format("01月02日") +
		" (第" + string(rune('0'+week/10)) + string(rune('0'+week%10)) + "週)"
}
