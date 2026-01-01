package brief

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"

	"github.com/masumomo/voice-brief/internal/model"
	"github.com/masumomo/voice-brief/internal/util"
)

// GeminiSummarizer はGemini APIを使用した要約エンジン
type GeminiSummarizer struct {
	apiKey         string
	model          string
	maxItemsDaily  int
	maxItemsWeekly int
	client         *genai.Client
}

const (
	defaultGeminiModel          = "gemini-2.0-flash-lite"
	defaultGeminiMaxItemsDaily  = 8
	defaultGeminiMaxItemsWeekly = 25
)

// NewGeminiSummarizer は新しいGeminiSummarizerを作成します
func NewGeminiSummarizer(apiKey, model string, maxDaily, maxWeekly int) (*GeminiSummarizer, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Gemini API Key が設定されていません")
	}
	if model == "" {
		model = defaultGeminiModel
	}
	if maxDaily <= 0 {
		maxDaily = defaultGeminiMaxItemsDaily
	}
	if maxWeekly <= 0 {
		maxWeekly = defaultGeminiMaxItemsWeekly
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("Gemini クライアントの作成に失敗: %w", err)
	}

	return &GeminiSummarizer{
		apiKey:         apiKey,
		model:          model,
		maxItemsDaily:  maxDaily,
		maxItemsWeekly: maxWeekly,
		client:         client,
	}, nil
}

// Close はクライアントをクローズします
func (s *GeminiSummarizer) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

// GenerateDaily はDaily Briefingを生成します
// since: 期間の開始時刻, until: 期間の終了時刻
func (s *GeminiSummarizer) GenerateDaily(events model.Events, since, until time.Time) (*model.Brief, error) {
	brief := model.NewBrief(model.BriefTypeDaily, since, until)

	// イベントを重要度でソート
	sort.Sort(events)

	// 最大件数まで取得
	topEvents := events
	if len(topEvents) > s.maxItemsDaily {
		topEvents = topEvents[:s.maxItemsDaily]
	}
	brief.AddEvents(topEvents)

	// Gemini APIで要約生成
	markdown, scriptText, err := s.generateBriefWithGemini(brief, model.BriefTypeDaily)
	if err != nil {
		return nil, fmt.Errorf("Gemini API での要約生成に失敗: %w", err)
	}

	brief.ScriptMarkdown = markdown
	brief.ScriptText = scriptText

	return brief, nil
}

// GenerateWeekly はWeekly Briefingを生成します
// since: 期間の開始時刻, until: 期間の終了時刻
func (s *GeminiSummarizer) GenerateWeekly(events model.Events, since, until time.Time) (*model.Brief, error) {
	brief := model.NewBrief(model.BriefTypeWeekly, since, until)

	// イベントを重要度でソート
	sort.Sort(events)

	// 最大件数まで取得
	topEvents := events
	if len(topEvents) > s.maxItemsWeekly {
		topEvents = topEvents[:s.maxItemsWeekly]
	}
	brief.AddEvents(topEvents)

	// Gemini APIで要約生成
	markdown, scriptText, err := s.generateBriefWithGemini(brief, model.BriefTypeWeekly)
	if err != nil {
		return nil, fmt.Errorf("Gemini API での要約生成に失敗: %w", err)
	}

	brief.ScriptMarkdown = markdown
	brief.ScriptText = scriptText

	return brief, nil
}

// generateBriefWithGemini はGemini APIを使ってブリーフィングを生成します
func (s *GeminiSummarizer) generateBriefWithGemini(brief *model.Brief, briefType model.BriefType) (markdown, scriptText string, err error) {
	ctx := context.Background()

	// プロンプトを構築
	prompt := s.buildPrompt(brief, briefType)

	// API呼び出しログ
	fmt.Printf("🤖 Gemini API 呼び出し開始: model=%s, events=%d, prompt_length=%d\n",
		s.model, len(brief.Items), len(prompt))
	fmt.Println("─────────────────────────────────────────")
	fmt.Println("📝 プロンプト:")
	fmt.Println(prompt)
	fmt.Println("─────────────────────────────────────────")
	startTime := time.Now()

	// Gemini APIを呼び出し
	genModel := s.client.GenerativeModel(s.model)

	// 設定: 温度を低めに設定して一貫性を保つ
	genModel.SetTemperature(0.3)
	genModel.SetTopP(0.8)
	genModel.SetTopK(40)

	// レート制限対策
	time.Sleep(1 * time.Second)

	resp, err := genModel.GenerateContent(ctx, genai.Text(prompt))
	elapsed := time.Since(startTime)

	if err != nil {
		fmt.Printf("❌ Gemini API エラー: %v (elapsed=%v)\n", err, elapsed)
		return "", "", fmt.Errorf("Gemini API 呼び出しエラー: %w", err)
	}

	// レスポンスを解析
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		fmt.Printf("❌ Gemini API レスポンスが空 (elapsed=%v)\n", elapsed)
		return "", "", fmt.Errorf("Gemini API からの応答が空です")
	}

	// トークン使用量をログ出力
	if resp.UsageMetadata != nil {
		fmt.Printf("📊 Gemini API 完了: prompt_tokens=%d, response_tokens=%d, total_tokens=%d, elapsed=%v\n",
			resp.UsageMetadata.PromptTokenCount,
			resp.UsageMetadata.CandidatesTokenCount,
			resp.UsageMetadata.TotalTokenCount,
			elapsed)
	} else {
		fmt.Printf("📊 Gemini API 完了: elapsed=%v\n", elapsed)
	}

	// テキスト部分を取得
	var result strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			result.WriteString(string(text))
		}
	}

	generatedText := result.String()

	// Markdownとスクリプトテキストに分割
	// "---SCRIPT---" で区切る想定
	parts := strings.SplitN(generatedText, "---SCRIPT---", 2)
	if len(parts) == 2 {
		markdown = strings.TrimSpace(parts[0])
		scriptText = strings.TrimSpace(parts[1])
	} else {
		// 区切りがない場合は同じ内容を使用
		markdown = strings.TrimSpace(generatedText)
		scriptText = strings.TrimSpace(generatedText)
	}

	return markdown, scriptText, nil
}

// buildPrompt はGemini API用のプロンプトを構築します
func (s *GeminiSummarizer) buildPrompt(brief *model.Brief, briefType model.BriefType) string {
	var sb strings.Builder

	// システム指示
	sb.WriteString("あなたは優秀なエグゼクティブアシスタントです。\n")
	sb.WriteString("以下のSlackメッセージとNotion更新情報から、簡潔で分かりやすいブリーフィングを作成してください。\n\n")

	// ブリーフィングタイプに応じた指示
	if briefType == model.BriefTypeDaily {
		sb.WriteString("## タスク: デイリーブリーフィング作成\n")
		sb.WriteString("過去24時間の重要な情報をまとめてください。\n\n")
	} else {
		sb.WriteString("## タスク: ウィークリーブリーフィング作成\n")
		sb.WriteString("過去1週間の重要な情報をまとめてください。\n\n")
	}

	// 出力形式の指定
	sb.WriteString("## 出力形式\n")
	sb.WriteString("1. Markdown形式で見やすいブリーフィングを作成\n")
	sb.WriteString("2. 次に「---SCRIPT---」という区切り文字を挿入\n")
	sb.WriteString("3. 音声読み上げ用の5分程度の自然な日本語スクリプトを作成（Markdownなし、箇条書き記号なし）\n\n")

	sb.WriteString("## ガイドライン\n")
	sb.WriteString("- 重要度の高い順に整理\n")
	sb.WriteString("- 簡潔に要点をまとめる（各項目1-2文）\n")
	sb.WriteString("- 音声スクリプトは「おはようございます」などの挨拶から始める\n")
	sb.WriteString("- 音声スクリプトは耳で聞いて理解しやすい自然な話し言葉で\n\n")

	// イベントデータ
	sb.WriteString("## 入力データ\n\n")

	// Slackメッセージ
	slackEvents := filterEventsBySource(brief.Items, "slack")
	if len(slackEvents) > 0 {
		sb.WriteString("### Slackメッセージ\n\n")
		for i, event := range slackEvents {
			sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, event.Source, util.SanitizeUTF8(event.Title)))
			if event.Body != "" {
				sb.WriteString(fmt.Sprintf("   内容: %s\n", util.SanitizeUTF8(event.Body)))
			}
			sb.WriteString(fmt.Sprintf("   時刻: %s\n", event.Timestamp.Format("2006-01-02 15:04")))
			sb.WriteString(fmt.Sprintf("   URL: %s\n", event.URL))
			sb.WriteString("\n")
		}
	}

	// Notion更新
	notionEvents := filterEventsBySource(brief.Items, "notion")
	if len(notionEvents) > 0 {
		sb.WriteString("### Notion更新\n\n")
		for i, event := range notionEvents {
			sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, event.Source, util.SanitizeUTF8(event.Title)))
			if event.Body != "" {
				sb.WriteString(fmt.Sprintf("   内容: %s\n", util.SanitizeUTF8(event.Body)))
			}
			sb.WriteString(fmt.Sprintf("   時刻: %s\n", event.Timestamp.Format("2006-01-02 15:04")))
			sb.WriteString(fmt.Sprintf("   URL: %s\n", event.URL))
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n---\n")
	sb.WriteString("それでは、上記のデータから簡潔で分かりやすいブリーフィングを作成してください。\n")

	return sb.String()
}

// filterEventsBySource はソース名でイベントをフィルタリングします
func filterEventsBySource(events model.Events, source string) model.Events {
	var filtered model.Events
	for _, event := range events {
		if strings.HasPrefix(strings.ToLower(event.Source), source) {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

// GetAPIKey は環境変数からGemini API Keyを取得します
func GetAPIKey(envName string) string {
	if envName == "" {
		envName = "GEMINI_API_KEY"
	}
	return os.Getenv(envName)
}
