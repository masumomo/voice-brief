package filter

import (
	"strings"

	"github.com/masumomo/voice-brief/internal/model"
)

// ImportanceCalculator は重要度を計算するインターフェース
type ImportanceCalculator interface {
	Calculate(event *model.Event) int
}

// RuleBasedCalculator はルールベースの重要度計算
type RuleBasedCalculator struct {
	// 重要キーワード（これらを含むと重要度が上がる）
	HighPriorityKeywords []string
	// 除外キーワード（これらを含むと重要度が下がる）
	LowPriorityKeywords []string
}

// NewRuleBasedCalculator は新しいRuleBasedCalculatorを作成します
func NewRuleBasedCalculator() *RuleBasedCalculator {
	return &RuleBasedCalculator{
		HighPriorityKeywords: []string{
			"緊急", "障害", "エラー", "失敗", "ブロック", "停止",
			"urgent", "critical", "error", "failure", "blocked", "down",
			"重要", "注意", "確認", "承認", "レビュー",
			"important", "attention", "review", "approve",
		},
		LowPriorityKeywords: []string{
			"了解", "承知", "ありがとう", "thanks", "ok", "👍",
			"参加しました", "退出しました", "joined", "left",
		},
	}
}

// Calculate はイベントの重要度を計算します（0-100）
func (c *RuleBasedCalculator) Calculate(event *model.Event) int {
	importance := 50 // ベーススコア

	// タイトルと本文を連結して検索対象とする
	text := strings.ToLower(event.Title + " " + event.Body)

	// 高優先度キーワードチェック
	for _, keyword := range c.HighPriorityKeywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			importance += 30
			break // 最初の1つだけカウント
		}
	}

	// 低優先度キーワードチェック
	for _, keyword := range c.LowPriorityKeywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			importance -= 20
			break
		}
	}

	// メンション含む場合は重要度アップ
	if strings.Contains(text, "@") || strings.Contains(event.Body, "<!channel>") || strings.Contains(event.Body, "<!here>") {
		importance += 20
	}

	// 短文は重要度ダウン
	if len(event.Body) < 10 {
		importance -= 20
	}

	// カテゴリ別調整
	switch event.Category {
	case model.EventCategoryIncident:
		importance += 40
	case model.EventCategoryDev:
		importance += 10
	case model.EventCategoryBiz:
		importance += 5
	}

	// スレッド数が多い場合は重要度アップ（Refsに格納されている想定）
	if threadCount, ok := event.Refs["thread_count"]; ok {
		if len(threadCount) > 0 {
			// 簡易的に文字列の数値をチェック
			if threadCount >= "5" {
				importance += 10
			}
		}
	}

	// 0-100の範囲に収める
	if importance < 0 {
		importance = 0
	}
	if importance > 100 {
		importance = 100
	}

	return importance
}

// CalculateAll は複数のイベントの重要度を一括計算します
func CalculateAll(events model.Events, calculator ImportanceCalculator) {
	for _, event := range events {
		event.Importance = calculator.Calculate(event)
	}
}

// FilterByKeywords はキーワードに基づいてイベントをフィルタします
func FilterByKeywords(events model.Events, excludeKeywords []string) model.Events {
	if len(excludeKeywords) == 0 {
		return events
	}

	filtered := make(model.Events, 0)
	for _, event := range events {
		text := strings.ToLower(event.Title + " " + event.Body)
		excluded := false

		for _, keyword := range excludeKeywords {
			if strings.Contains(text, strings.ToLower(keyword)) {
				excluded = true
				break
			}
		}

		if !excluded {
			filtered = append(filtered, event)
		}
	}

	return filtered
}

// FilterShortMessages は短いメッセージを除外します
func FilterShortMessages(events model.Events, minLength int) model.Events {
	if minLength <= 0 {
		return events
	}

	filtered := make(model.Events, 0)
	for _, event := range events {
		if len(event.Body) >= minLength {
			filtered = append(filtered, event)
		}
	}

	return filtered
}
