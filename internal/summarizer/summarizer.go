package summarizer

import (
	"time"

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
	// since: 期間の開始時刻, until: 期間の終了時刻
	GenerateDaily(events model.Events, since, until time.Time) (*model.Brief, error)

	// GenerateWeekly はWeekly Briefingを生成します
	// since: 期間の開始時刻, until: 期間の終了時刻
	GenerateWeekly(events model.Events, since, until time.Time) (*model.Brief, error)
}
