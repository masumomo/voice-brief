package summarizer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/masumomo/voice-brief/internal/model"
	"google.golang.org/api/option"
)

// GeminiEventSummarizer はGemini APIを使用したイベント要約器
type GeminiEventSummarizer struct {
	apiKey        string
	model         string
	maxSummaryLen int
	client        *genai.Client
}

// NewGeminiEventSummarizer は新しいGeminiEventSummarizerを作成します
func NewGeminiEventSummarizer(apiKey, modelName string, maxSummaryLen int) (*GeminiEventSummarizer, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Gemini API Key が設定されていません")
	}
	if modelName == "" {
		modelName = "gemini-2.0-flash-exp"
	}
	if maxSummaryLen <= 0 {
		maxSummaryLen = 200
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("Gemini クライアントの作成に失敗: %w", err)
	}

	return &GeminiEventSummarizer{
		apiKey:        apiKey,
		model:         modelName,
		maxSummaryLen: maxSummaryLen,
		client:        client,
	}, nil
}

// Close はクライアントをクローズします
func (s *GeminiEventSummarizer) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

// Summarize は単一イベントを要約します
func (s *GeminiEventSummarizer) Summarize(event *model.Event) error {
	if event == nil {
		return fmt.Errorf("event is nil")
	}

	// 既にSummaryがある場合はスキップ
	if event.Summary != "" {
		return nil
	}

	// Bodyが短い場合はそのまま使用
	if len(event.Body) <= s.maxSummaryLen {
		event.Summary = event.Body
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	prompt := s.buildPrompt(event)

	genModel := s.client.GenerativeModel(s.model)
	genModel.SetTemperature(0.3)

	resp, err := genModel.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		// エラー時はルールベースのフォールバック
		event.Summary = s.fallbackSummary(event)
		return nil
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		event.Summary = s.fallbackSummary(event)
		return nil
	}

	// レスポンスから要約を抽出
	var result strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			result.WriteString(string(text))
		}
	}

	summary := strings.TrimSpace(result.String())
	if len(summary) > s.maxSummaryLen {
		summary = summary[:s.maxSummaryLen] + "..."
	}

	event.Summary = summary
	return nil
}

// SummarizeAll は複数イベントを一括要約します
func (s *GeminiEventSummarizer) SummarizeAll(events model.Events) error {
	for _, event := range events {
		if err := s.Summarize(event); err != nil {
			// エラーでも継続（Best Effort）
			continue
		}
	}
	return nil
}

// buildPrompt はイベント要約用のプロンプトを構築します
func (s *GeminiEventSummarizer) buildPrompt(event *model.Event) string {
	commentsInfo := ""
	if len(event.Comments) > 0 {
		commentsInfo = fmt.Sprintf("\n\nコメント数: %d件", len(event.Comments))
		// 最初の数件のコメントを含める
		for i, comment := range event.Comments {
			if i >= 3 {
				commentsInfo += fmt.Sprintf("\n... 他 %d件", len(event.Comments)-3)
				break
			}
			text := comment.Text
			if len(text) > 100 {
				text = text[:100] + "..."
			}
			commentsInfo += fmt.Sprintf("\n- %s: %s", comment.Author, text)
		}
	}

	return fmt.Sprintf(`以下のイベント情報を%d文字以内で簡潔に要約してください。
要点のみを抽出し、音声で読み上げやすい自然な日本語で回答してください。

タイトル: %s
ソース: %s
場所: %s
本文:
%s%s

要約:`,
		s.maxSummaryLen,
		event.Title,
		event.Source,
		event.Location,
		event.Body,
		commentsInfo,
	)
}

// fallbackSummary はエラー時のフォールバック要約を生成します
func (s *GeminiEventSummarizer) fallbackSummary(event *model.Event) string {
	text := event.Body
	text = strings.ReplaceAll(text, "\n", " ")
	for strings.Contains(text, "  ") {
		text = strings.ReplaceAll(text, "  ", " ")
	}
	text = strings.TrimSpace(text)

	if len(text) <= s.maxSummaryLen {
		return text
	}
	return text[:s.maxSummaryLen] + "..."
}
