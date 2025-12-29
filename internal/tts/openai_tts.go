package tts

import (
	"context"
	"fmt"
	"os"

	openai "github.com/sashabaranov/go-openai"
)

// OpenAITTS はOpenAI TTS APIによる音声合成
type OpenAITTS struct {
	config Config
	client *openai.Client
	apiKey string

	// コスト管理
	charactersUsed int
}

// NewOpenAITTS は新しいOpenAITTSを作成します
func NewOpenAITTS(config *Config, apiKey string) *OpenAITTS {
	if config.Voice == "" {
		config.Voice = "alloy" // デフォルト音声
	}
	if config.Rate <= 0 {
		config.Rate = 1.0
	}
	if config.Format == "" {
		config.Format = "mp3"
	}

	client := openai.NewClient(apiKey)

	return &OpenAITTS{
		config: *config,
		client: client,
		apiKey: apiKey,
	}
}

// Generate はテキストから音声ファイルを生成します
func (o *OpenAITTS) Generate(ctx context.Context, text string, outputPath string) error {
	// OpenAI TTS API呼び出し
	req := openai.CreateSpeechRequest{
		Model: openai.TTSModel1, // tts-1 (高速) または tts-1-hd (高品質)
		Input: text,
		Voice: openai.SpeechVoice(o.config.Voice),
		Speed: o.config.Rate, // 0.25 - 4.0
	}

	// ResponseFormatの設定
	switch o.config.Format {
	case "mp3":
		req.ResponseFormat = openai.SpeechResponseFormatMp3
	case "opus":
		req.ResponseFormat = openai.SpeechResponseFormatOpus
	case "aac":
		req.ResponseFormat = openai.SpeechResponseFormatAac
	case "flac":
		req.ResponseFormat = openai.SpeechResponseFormatFlac
	default:
		req.ResponseFormat = openai.SpeechResponseFormatMp3
	}

	// API呼び出し
	resp, err := o.client.CreateSpeech(ctx, req)
	if err != nil {
		return fmt.Errorf("OpenAI TTS API呼び出しに失敗: %w", err)
	}
	defer resp.Close()

	// ファイルに保存
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("音声ファイルの作成に失敗: %w", err)
	}
	defer file.Close()

	// レスポンスをファイルに書き込み
	written, err := file.ReadFrom(resp)
	if err != nil {
		return fmt.Errorf("音声ファイルの書き込みに失敗: %w", err)
	}

	// 文字数を記録（コスト計算用）
	o.charactersUsed += len(text)

	fmt.Printf("✓ OpenAI TTS音声ファイルを生成: %s (%.2f KB)\n", outputPath, float64(written)/1024)

	return nil
}

// GetProvider はプロバイダー名を返します
func (o *OpenAITTS) GetProvider() string {
	return "openai_tts"
}

// GetCharactersUsed は使用文字数を返します
func (o *OpenAITTS) GetCharactersUsed() int {
	return o.charactersUsed
}

// EstimateCost はコストを見積もります（USD）
func (o *OpenAITTS) EstimateCost() float64 {
	// OpenAI TTS料金 (2024年12月現在)
	// tts-1: $15.00 / 1M characters
	// tts-1-hd: $30.00 / 1M characters

	pricePerChar := 15.00 / 1_000_000 // tts-1の料金
	if o.config.Provider == "tts-1-hd" {
		pricePerChar = 30.00 / 1_000_000
	}

	return float64(o.charactersUsed) * pricePerChar
}

// MapVoiceName はmacOS音声名をOpenAI音声名にマッピングします
func MapVoiceName(voice string) string {
	// macOS互換のマッピング
	mapping := map[string]string{
		"Kyoko":    "nova",   // 日本語に適した女性音声
		"Otoya":    "onyx",   // 日本語に適した男性音声
		"Samantha": "nova",   // 英語女性音声
		"Alex":     "onyx",   // 英語男性音声
	}

	if openaiVoice, ok := mapping[voice]; ok {
		return openaiVoice
	}

	// デフォルトはそのまま使用（alloy, echo, fable, onyx, nova, shimmer）
	return voice
}

// GetAvailableOpenAIVoices はOpenAI TTSで利用可能な音声のリストを返します
func GetAvailableOpenAIVoices() []string {
	return []string{
		"alloy",   // 中性的
		"echo",    // 男性的
		"fable",   // 英国英語的
		"onyx",    // 深い男性声
		"nova",    // 女性的
		"shimmer", // 明るい女性声
	}
}
