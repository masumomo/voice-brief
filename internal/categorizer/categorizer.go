package categorizer

import "github.com/masumomo/voice-brief/internal/model"

// Categorizer はイベントのカテゴリを判定するインターフェース
// 将来的にはLLMベースの実装も追加可能
type Categorizer interface {
	// Categorize はイベントのカテゴリを判定します
	Categorize(event *model.Event) string

	// CategorizeAll は複数イベントのカテゴリを一括判定します
	CategorizeAll(events model.Events)
}

// CategoryInput はカテゴリ判定に使用する入力情報
type CategoryInput struct {
	Source   string            // "slack" | "notion" | "github"
	Title    string            // イベントタイトル
	Body     string            // 本文
	Location string            // チャンネル名/DB名など
	Metadata map[string]string // 追加のメタデータ（プロパティ、ラベルなど）
}

// NewCategoryInputFromEvent はEventからCategoryInputを作成します
func NewCategoryInputFromEvent(event *model.Event) *CategoryInput {
	return &CategoryInput{
		Source:   event.Source,
		Title:    event.Title,
		Body:     event.Body,
		Location: event.Location,
		Metadata: event.Refs,
	}
}
