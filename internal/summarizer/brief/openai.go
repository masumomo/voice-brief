package brief

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"

	"github.com/masumomo/voice-brief/internal/model"
	"github.com/masumomo/voice-brief/internal/util"
)

// OpenAISummarizer はOpenAI APIを使用した要約エンジン
type OpenAISummarizer struct {
	apiKey         string
	model          string
	maxItemsDaily  int
	maxItemsWeekly int
	client         *openai.Client

	// コスト管理
	totalTokensUsed       int
	totalPromptTokens     int
	totalCompletionTokens int
}

const (
	defaultOpenAIModel          = "gpt-4o-mini"
	defaultOpenAIMaxItemsDaily  = 8
	defaultOpenAIMaxItemsWeekly = 25
)

// NewOpenAISummarizer は新しいOpenAISummarizerを作成します
func NewOpenAISummarizer(apiKey, model string, maxDaily, maxWeekly int) (*OpenAISummarizer, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API Key が設定されていません")
	}
	if model == "" {
		model = defaultOpenAIModel
	}
	if maxDaily <= 0 {
		maxDaily = defaultOpenAIMaxItemsDaily
	}
	if maxWeekly <= 0 {
		maxWeekly = defaultOpenAIMaxItemsWeekly
	}

	client := openai.NewClient(apiKey)

	return &OpenAISummarizer{
		apiKey:         apiKey,
		model:          model,
		maxItemsDaily:  maxDaily,
		maxItemsWeekly: maxWeekly,
		client:         client,
	}, nil
}

// GenerateDaily はDaily Briefingを生成します
// since: 期間の開始時刻, until: 期間の終了時刻
func (s *OpenAISummarizer) GenerateDaily(events model.Events, since, until time.Time) (*model.Brief, error) {
	brief := model.NewBrief(model.BriefTypeDaily, since, until)

	// イベントを重要度でソート
	sort.Sort(events)

	// 最大件数まで取得
	topEvents := events
	if len(topEvents) > s.maxItemsDaily {
		topEvents = topEvents[:s.maxItemsDaily]
	}
	brief.AddEvents(topEvents)

	// OpenAI APIで要約生成
	markdown, scriptText, err := s.generateBriefWithOpenAI(brief, model.BriefTypeDaily)
	if err != nil {
		return nil, fmt.Errorf("OpenAI API での要約生成に失敗: %w", err)
	}

	brief.ScriptMarkdown = markdown
	brief.ScriptText = scriptText

	return brief, nil
}

// GenerateWeekly はWeekly Briefingを生成します
// since: 期間の開始時刻, until: 期間の終了時刻
func (s *OpenAISummarizer) GenerateWeekly(events model.Events, since, until time.Time) (*model.Brief, error) {
	brief := model.NewBrief(model.BriefTypeWeekly, since, until)

	// イベントを重要度でソート
	sort.Sort(events)

	// 最大件数まで取得
	topEvents := events
	if len(topEvents) > s.maxItemsWeekly {
		topEvents = topEvents[:s.maxItemsWeekly]
	}
	brief.AddEvents(topEvents)

	// OpenAI APIで要約生成
	markdown, scriptText, err := s.generateBriefWithOpenAI(brief, model.BriefTypeWeekly)
	if err != nil {
		return nil, fmt.Errorf("OpenAI API での要約生成に失敗: %w", err)
	}

	brief.ScriptMarkdown = markdown
	brief.ScriptText = scriptText

	return brief, nil
}

// generateBriefWithOpenAI はOpenAI APIを使ってブリーフィングを生成します
func (s *OpenAISummarizer) generateBriefWithOpenAI(brief *model.Brief, briefType model.BriefType) (markdown, scriptText string, err error) {
	ctx := context.Background()

	// プロンプト構築
	systemPrompt := s.getSystemPrompt(briefType)
	prompt := s.buildPrompt(brief, briefType)

	// API呼び出しログ
	fmt.Printf("🤖 OpenAI API 呼び出し開始: model=%s, events=%d, prompt_length=%d\n",
		s.model, len(brief.Items), len(prompt))
	fmt.Println("─────────────────────────────────────────")
	fmt.Println("📝 システムプロンプト:")
	fmt.Println(systemPrompt)
	fmt.Println("─────────────────────────────────────────")
	fmt.Println("📝 ユーザープロンプト:")
	fmt.Println(prompt)
	fmt.Println("─────────────────────────────────────────")
	startTime := time.Now()

	// レート制限対策
	time.Sleep(1 * time.Second)

	// OpenAI Chat Completion API呼び出し
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
					Content: prompt,
				},
			},
			Temperature: 0.3,  // 一貫性を重視
			MaxTokens:   2000, // 出力トークン数制限
		},
	)
	elapsed := time.Since(startTime)

	if err != nil {
		fmt.Printf("❌ OpenAI API エラー: %v (elapsed=%v)\n", err, elapsed)
		return "", "", fmt.Errorf("OpenAI API呼び出しに失敗: %w", err)
	}

	// トークン使用量を記録
	s.totalTokensUsed += resp.Usage.TotalTokens
	s.totalPromptTokens += resp.Usage.PromptTokens
	s.totalCompletionTokens += resp.Usage.CompletionTokens

	fmt.Printf("📊 OpenAI API 完了: prompt_tokens=%d, completion_tokens=%d, total_tokens=%d, elapsed=%v\n",
		resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens, elapsed)

	if len(resp.Choices) == 0 {
		return "", "", fmt.Errorf("OpenAI APIからのレスポンスが空です")
	}

	content := resp.Choices[0].Message.Content

	// レスポンスをMarkdownとScriptに分割
	// "---SCRIPT---" 区切りを期待
	parts := strings.Split(content, "---SCRIPT---")
	if len(parts) == 2 {
		markdown = strings.TrimSpace(parts[0])
		scriptText = strings.TrimSpace(parts[1])
	} else {
		// 区切りがない場合は全体をMarkdownとして扱い、Scriptは同じものを使用
		markdown = strings.TrimSpace(content)
		scriptText = markdown
	}

	return markdown, scriptText, nil
}

// getSystemPrompt はシステムプロンプトを返します
func (s *OpenAISummarizer) getSystemPrompt(briefType model.BriefType) string {
	if briefType == model.BriefTypeDaily {
		return `あなたはビジネスパーソン向けの音声ブリーフィングアシスタントです。
SlackとNotionから収集した情報を、簡潔で分かりやすい音声原稿にまとめてください。

出力形式:
1. Markdown形式のブリーフィング（視覚的に読みやすく）
2. "---SCRIPT---"区切り
3. 音声読み上げ用のプレーンテキスト（記号なし、自然な話し言葉）

重要なポイント:
- 重要度の高い情報から順に紹介
- 各項目は30秒以内で説明
- 専門用語は必要に応じて説明
- 全体で5分前後の長さ
- 現在時刻は ` + time.Now().Format("15:04") + ` です。この時間帯に適した挨拶から音声スクリプトを始めてください
- 音声スクリプトは自然な日本語の話し言葉で
- 少し皮肉っぽくクスッと笑えるトーンで。たまにツッコミどころを満載の一言を入れてください。ただし情報は正確に伝えること`
	}

	return `あなたはビジネスパーソン向けの音声ブリーフィングアシスタントです。
1週間分のSlackとNotionの重要な情報を、週次レビュー用にまとめてください。

出力形式:
1. Markdown形式のブリーフィング（視覚的に読みやすく）
2. "---SCRIPT---"区切り
3. 音声読み上げ用のプレーンテキスト（記号なし、自然な話し言葉）

重要なポイント:
- カテゴリごとに整理（Incident, Dev, Biz, Ops等）
- 全体のトレンドや傾向を分析
- 次週へのアクションアイテムを提案
- 全体で10分前後の長さ
- 現在時刻は ` + time.Now().Format("15:04") + ` です。この時間帯に適した挨拶から音声スクリプトを始めてください
- 音声スクリプトは自然な日本語の話し言葉で
- 少し皮肉っぽくクスッと笑えるトーンで。たまにツッコミどころを満載の一言を入れてください。ただし情報は正確に伝えること`
}

// buildPrompt はユーザープロンプトを構築します
func (s *OpenAISummarizer) buildPrompt(brief *model.Brief, briefType model.BriefType) string {
	var sb strings.Builder

	if briefType == model.BriefTypeDaily {
		sb.WriteString(fmt.Sprintf("# 本日のブリーフィング (%s)\n\n", brief.WindowStart.Format("2006-01-02")))
	} else {
		sb.WriteString(fmt.Sprintf("# 今週のブリーフィング (%s - %s)\n\n",
			brief.WindowStart.Format("2006-01-02"),
			brief.WindowEnd.Format("2006-01-02")))
	}

	sb.WriteString(fmt.Sprintf("収集したイベント数: %d件\n\n", len(brief.Items)))

	// カテゴリ別にグループ化
	categoryEvents := make(map[string]model.Events)
	for _, event := range brief.Items {
		category := event.Category
		if category == "" {
			category = model.EventCategoryOther
		}
		categoryEvents[category] = append(categoryEvents[category], event)
	}

	// カテゴリごとにイベントを記載
	categoryOrder := []string{
		model.EventCategoryIncident,
		model.EventCategoryDev,
		model.EventCategoryBiz,
		model.EventCategoryOps,
		model.EventCategoryOther,
	}

	for _, category := range categoryOrder {
		events, exists := categoryEvents[category]
		if !exists || len(events) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("## カテゴリ: %s (%d件)\n\n", category, len(events)))

		for i, event := range events {
			sb.WriteString(fmt.Sprintf("### %d. %s\n", i+1, util.SanitizeUTF8(event.Title)))
			sb.WriteString(fmt.Sprintf("- **ソース**: %s (%s)\n", event.Source, util.SanitizeUTF8(event.Location)))
			sb.WriteString(fmt.Sprintf("- **時刻**: %s\n", event.Timestamp.Format("2006-01-02 15:04")))
			sb.WriteString(fmt.Sprintf("- **重要度**: %d/100\n", event.Importance))
			if event.Body != "" {
				// 本文が長い場合は省略
				body := util.SanitizeUTF8(event.Body)
				if len(body) > 200 {
					body = body[:200] + "..."
				}
				sb.WriteString(fmt.Sprintf("- **内容**: %s\n", body))
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n---\n\n")
	sb.WriteString("上記の情報を元に、視覚的に読みやすいMarkdownブリーフィングと、")
	sb.WriteString("音声読み上げ用の自然な話し言葉のスクリプトを生成してください。\n")
	sb.WriteString("Markdownと音声スクリプトの間には \"---SCRIPT---\" を挿入してください。")

	return sb.String()
}

// GetTokenUsage はトークン使用量を返します
func (s *OpenAISummarizer) GetTokenUsage() (total, prompt, completion int) {
	return s.totalTokensUsed, s.totalPromptTokens, s.totalCompletionTokens
}

// EstimateCost はコストを見積もります（USD）
func (s *OpenAISummarizer) EstimateCost() float64 {
	// gpt-4o-mini の料金 (2024年12月現在)
	// Input: $0.150 / 1M tokens
	// Output: $0.600 / 1M tokens

	// gpt-4o の料金
	// Input: $2.50 / 1M tokens
	// Output: $10.00 / 1M tokens

	var inputPrice, outputPrice float64

	switch s.model {
	case "gpt-4o-mini":
		inputPrice = 0.150 / 1_000_000
		outputPrice = 0.600 / 1_000_000
	case "gpt-4o":
		inputPrice = 2.50 / 1_000_000
		outputPrice = 10.00 / 1_000_000
	case "gpt-4-turbo", "gpt-4-turbo-preview":
		inputPrice = 10.00 / 1_000_000
		outputPrice = 30.00 / 1_000_000
	case "gpt-3.5-turbo":
		inputPrice = 0.50 / 1_000_000
		outputPrice = 1.50 / 1_000_000
	default:
		// デフォルトはgpt-4o-miniの価格
		inputPrice = 0.150 / 1_000_000
		outputPrice = 0.600 / 1_000_000
	}

	inputCost := float64(s.totalPromptTokens) * inputPrice
	outputCost := float64(s.totalCompletionTokens) * outputPrice

	return inputCost + outputCost
}
