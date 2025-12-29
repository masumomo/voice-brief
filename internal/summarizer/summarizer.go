package summarizer

import (
	"github.com/masumomo/voice-brief/internal/model"
)

// Summarizer はイベントからブリーフィングを生成するインターフェース
type Summarizer interface {
	// GenerateDaily はDaily Briefingを生成します
	GenerateDaily(events model.Events) (*model.Brief, error)

	// GenerateWeekly はWeekly Briefingを生成します
	GenerateWeekly(events model.Events) (*model.Brief, error)
}
