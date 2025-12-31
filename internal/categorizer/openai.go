package categorizer

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/masumomo/voice-brief/internal/model"
	openai "github.com/sashabaranov/go-openai"
)

// OpenAICategorizer はOpenAI APIを使用したカテゴリ判定
type OpenAICategorizer struct {
	apiKey string
	model  string
	client *openai.Client
}

// NewOpenAICategorizer は新しいOpenAICategorizerを作成します
func NewOpenAICategorizer(apiKey, modelName string) (*OpenAICategorizer, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API Key が設定されていません")
	}
	if modelName == "" {
		modelName = "gpt-4o-mini" // コスト効率重視
	}

	client := openai.NewClient(apiKey)

	return &OpenAICategorizer{
		apiKey: apiKey,
		model:  modelName,
		client: client,
	}, nil
}

// Categorize はイベントのカテゴリを判定します
func (c *OpenAICategorizer) Categorize(event *model.Event) string {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	systemPrompt := `あなたはイベント分類の専門家です。与えられたイベント情報を以下のカテゴリのいずれかに分類してください。
回答はカテゴリ名のみを返してください。

カテゴリ:
- incident: 障害、緊急対応、バグ修正、エラー
- dev: 開発、機能実装、コードレビュー、設計
- biz: ビジネス、営業、マーケティング、顧客対応
- ops: 運用、インフラ、デプロイ、監視
- other: 上記に該当しないもの`

	userPrompt := c.buildPrompt(event)

	resp, err := c.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: c.model,
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
			Temperature: 0.1, // 一貫性を重視
			MaxTokens:   10,  // カテゴリ名のみなので短い
		},
	)

	if err != nil {
		fmt.Printf("⚠️  OpenAI カテゴリ判定エラー: %v\n", err)
		return model.EventCategoryOther
	}

	if len(resp.Choices) == 0 {
		return model.EventCategoryOther
	}

	return c.parseCategory(resp.Choices[0].Message.Content)
}

// CategorizeAll は複数イベントのカテゴリを一括判定します
func (c *OpenAICategorizer) CategorizeAll(events model.Events) {
	for _, event := range events {
		if event.Category == "" || event.Category == model.EventCategoryOther {
			event.Category = c.Categorize(event)
		}
	}
}

// buildPrompt はカテゴリ判定用のプロンプトを構築します
func (c *OpenAICategorizer) buildPrompt(event *model.Event) string {
	body := event.Body
	if len(body) > 500 {
		body = body[:500] + "..."
	}

	return fmt.Sprintf(`イベント情報:
- ソース: %s
- 場所: %s
- タイトル: %s
- 本文: %s

カテゴリ名のみ回答:`,
		event.Source,
		event.Location,
		event.Title,
		body,
	)
}

// parseCategory はLLMレスポンスからカテゴリを抽出します
func (c *OpenAICategorizer) parseCategory(response string) string {
	response = strings.ToLower(strings.TrimSpace(response))

	validCategories := []string{
		model.EventCategoryIncident,
		model.EventCategoryDev,
		model.EventCategoryBiz,
		model.EventCategoryOps,
		model.EventCategoryOther,
	}

	// 完全一致を優先
	for _, cat := range validCategories {
		if response == cat {
			return cat
		}
	}

	// 部分一致
	for _, cat := range validCategories {
		if strings.Contains(response, cat) {
			return cat
		}
	}

	return model.EventCategoryOther
}

// GetOpenAIAPIKey は環境変数からOpenAI API Keyを取得します
func GetOpenAIAPIKey(envName string) string {
	if envName == "" {
		envName = "OPENAI_API_KEY"
	}
	return os.Getenv(envName)
}
