package summarizer

import (
	"github.com/masumomo/voice-brief/internal/model"
)

// EventSummarizer は個々のイベントを要約するインターフェース
// Body + Comments から Summary を生成
type EventSummarizer interface {
	// Summarize は単一イベントを要約します
	Summarize(event *model.Event) error

	// SummarizeAll は複数イベントを一括要約します
	SummarizeAll(events model.Events) error
}

// BriefSummarizer はイベント群からブリーフィングを生成するインターフェース
type BriefSummarizer interface {
	// GenerateDaily はDaily Briefingを生成します
	GenerateDaily(events model.Events) (*model.Brief, error)

	// GenerateWeekly はWeekly Briefingを生成します
	GenerateWeekly(events model.Events) (*model.Brief, error)
}
