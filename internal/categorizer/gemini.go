package categorizer

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"

	"github.com/masumomo/voice-brief/internal/model"
	"github.com/masumomo/voice-brief/internal/util"
)

// GeminiCategorizer はGemini APIを使用したカテゴリ判定
type GeminiCategorizer struct {
	apiKey string
	model  string
	client *genai.Client
}

// NewGeminiCategorizer は新しいGeminiCategorizerを作成します
func NewGeminiCategorizer(apiKey, modelName string) (*GeminiCategorizer, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Gemini API Key が設定されていません")
	}
	if modelName == "" {
		modelName = "gemini-2.0-flash-exp"
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("Gemini クライアントの作成に失敗: %w", err)
	}

	return &GeminiCategorizer{
		apiKey: apiKey,
		model:  modelName,
		client: client,
	}, nil
}

// Close はクライアントをクローズします
func (c *GeminiCategorizer) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// Categorize はイベントのカテゴリを判定します
func (c *GeminiCategorizer) Categorize(event *model.Event) string {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	prompt := c.buildPrompt(event)

	genModel := c.client.GenerativeModel(c.model)
	genModel.SetTemperature(0.1) // 一貫性を重視

	resp, err := genModel.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		fmt.Printf("⚠️  Gemini カテゴリ判定エラー: %v\n", err)
		return model.EventCategoryOther
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return model.EventCategoryOther
	}

	// レスポンスからカテゴリを抽出
	var result strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			result.WriteString(string(text))
		}
	}

	return c.parseCategory(result.String())
}

// CategorizeAll は複数イベントのカテゴリを一括判定します
func (c *GeminiCategorizer) CategorizeAll(events model.Events) {
	for _, event := range events {
		if event.Category == "" || event.Category == model.EventCategoryOther {
			event.Category = c.Categorize(event)
		}
	}
}

// buildPrompt はカテゴリ判定用のプロンプトを構築します
func (c *GeminiCategorizer) buildPrompt(event *model.Event) string {
	return fmt.Sprintf(`以下のイベント情報から、最も適切なカテゴリを1つだけ選んでください。

カテゴリ一覧:
- incident: 障害、緊急対応、バグ修正、エラー
- dev: 開発、機能実装、コードレビュー、設計
- biz: ビジネス、営業、マーケティング、顧客対応
- ops: 運用、インフラ、デプロイ、監視
- other: 上記に該当しないもの

イベント情報:
- ソース: %s
- 場所: %s
- タイトル: %s
- 本文: %s

回答はカテゴリ名のみ（incident, dev, biz, ops, other のいずれか）を返してください。`,
		event.Source,
		util.SanitizeUTF8(event.Location),
		util.SanitizeUTF8(event.Title),
		util.SanitizeUTF8(util.TruncateText(event.Body, 500)),
	)
}

// parseCategory はLLMレスポンスからカテゴリを抽出します
func (c *GeminiCategorizer) parseCategory(response string) string {
	response = strings.ToLower(strings.TrimSpace(response))

	// 完全一致を優先
	validCategories := []string{
		model.EventCategoryIncident,
		model.EventCategoryDev,
		model.EventCategoryBiz,
		model.EventCategoryOps,
		model.EventCategoryOther,
	}

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

// GetGeminiAPIKey は環境変数からGemini API Keyを取得します
func GetGeminiAPIKey(envName string) string {
	if envName == "" {
		envName = "GEMINI_API_KEY"
	}
	return os.Getenv(envName)
}
