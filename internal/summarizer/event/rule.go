package event

import (
	"fmt"
	"os"
	"strings"

	"github.com/masumomo/voice-brief/internal/model"
)

// GetAPIKey は環境変数からAPI Keyを取得します
func GetAPIKey(envName string) string {
	if envName == "" {
		return ""
	}
	return os.Getenv(envName)
}

// RuleEventSummarizer はルールベースのイベント要約器
type RuleEventSummarizer struct {
	maxSummaryLen int // 要約の最大文字数
}

// NewRuleEventSummarizer は新しいRuleEventSummarizerを作成します
func NewRuleEventSummarizer(maxSummaryLen int) *RuleEventSummarizer {
	if maxSummaryLen <= 0 {
		maxSummaryLen = 200 // デフォルト
	}
	return &RuleEventSummarizer{
		maxSummaryLen: maxSummaryLen,
	}
}

// Summarize は単一イベントを要約します
func (s *RuleEventSummarizer) Summarize(event *model.Event) error {
	if event == nil {
		return fmt.Errorf("event is nil")
	}

	// 既にSummaryがある場合はスキップ
	if event.Summary != "" {
		return nil
	}

	// Bodyから要約を生成
	summary := s.generateSummary(event)
	event.Summary = summary

	return nil
}

// SummarizeAll は複数イベントを一括要約します
func (s *RuleEventSummarizer) SummarizeAll(events model.Events) error {
	for _, event := range events {
		if err := s.Summarize(event); err != nil {
			// エラーでも継続（Best Effort）
			continue
		}
	}
	return nil
}

// generateSummary はイベントから要約テキストを生成します
func (s *RuleEventSummarizer) generateSummary(event *model.Event) string {
	var parts []string

	// Bodyの先頭部分を抽出
	bodyPreview := s.truncateText(event.Body, s.maxSummaryLen-50)
	if bodyPreview != "" {
		parts = append(parts, bodyPreview)
	}

	// コメントがある場合は件数を追記
	if len(event.Comments) > 0 {
		commentNote := fmt.Sprintf("（%d件のコメント）", len(event.Comments))
		parts = append(parts, commentNote)
	}

	if len(parts) == 0 {
		return event.Title // Bodyがない場合はTitleを使用
	}

	summary := strings.Join(parts, " ")
	return s.truncateText(summary, s.maxSummaryLen)
}

// truncateText はテキストを指定長で切り詰めます
func (s *RuleEventSummarizer) truncateText(text string, maxLen int) string {
	// 改行を空白に置換
	text = strings.ReplaceAll(text, "\n", " ")
	// 連続する空白を1つに
	for strings.Contains(text, "  ") {
		text = strings.ReplaceAll(text, "  ", " ")
	}
	text = strings.TrimSpace(text)

	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}
