package summarizer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/masumomo/voice-brief/internal/model"
	openai "github.com/sashabaranov/go-openai"
)

// OpenAIEventSummarizer はOpenAI APIを使用したイベント要約器
type OpenAIEventSummarizer struct {
	apiKey        string
	model         string
	maxSummaryLen int
	client        *openai.Client
}

// NewOpenAIEventSummarizer は新しいOpenAIEventSummarizerを作成します
func NewOpenAIEventSummarizer(apiKey, modelName string, maxSummaryLen int) (*OpenAIEventSummarizer, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API Key が設定されていません")
	}
	if modelName == "" {
		modelName = "gpt-4o-mini" // コスト効率重視
	}
	if maxSummaryLen <= 0 {
		maxSummaryLen = 200
	}

	client := openai.NewClient(apiKey)

	return &OpenAIEventSummarizer{
		apiKey:        apiKey,
		model:         modelName,
		maxSummaryLen: maxSummaryLen,
		client:        client,
	}, nil
}

// Summarize は単一イベントを要約します
func (s *OpenAIEventSummarizer) Summarize(event *model.Event) error {
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

	systemPrompt := fmt.Sprintf(`あなたはイベント要約の専門家です。
与えられたイベント情報を%d文字以内で簡潔に要約してください。
音声で読み上げやすい自然な日本語で、要点のみを抽出してください。
要約のみを回答し、余計な説明は不要です。`, s.maxSummaryLen)

	userPrompt := s.buildPrompt(event)

	resp, err := s.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: s.model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: userPrompt,
				},
			},
			Temperature: 0.3,
			MaxTokens:   200, // 要約なので短い
		},
	)

	if err != nil {
		// エラー時はルールベースのフォールバック
		event.Summary = s.fallbackSummary(event)
		return nil
	}

	if len(resp.Choices) == 0 {
		event.Summary = s.fallbackSummary(event)
		return nil
	}

	summary := strings.TrimSpace(resp.Choices[0].Message.Content)
	if len(summary) > s.maxSummaryLen {
		summary = summary[:s.maxSummaryLen] + "..."
	}

	event.Summary = summary
	return nil
}

// SummarizeAll は複数イベントを一括要約します
func (s *OpenAIEventSummarizer) SummarizeAll(events model.Events) error {
	for _, event := range events {
		if err := s.Summarize(event); err != nil {
			// エラーでも継続（Best Effort）
			continue
		}
	}
	return nil
}

// buildPrompt はイベント要約用のプロンプトを構築します
func (s *OpenAIEventSummarizer) buildPrompt(event *model.Event) string {
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

	return fmt.Sprintf(`タイトル: %s
ソース: %s
場所: %s
本文:
%s%s`,
		event.Title,
		event.Source,
		event.Location,
		event.Body,
		commentsInfo,
	)
}

// fallbackSummary はエラー時のフォールバック要約を生成します
func (s *OpenAIEventSummarizer) fallbackSummary(event *model.Event) string {
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
