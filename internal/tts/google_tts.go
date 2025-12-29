package tts

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"google.golang.org/api/option"
)

// GoogleTTS はGoogle Cloud Text-to-Speechを使用したTTS
type GoogleTTS struct {
	config *Config
	client *texttospeech.Client
}

// NewGoogleTTS は新しいGoogleTTSを作成します
func NewGoogleTTS(ctx context.Context, config *Config, credentialsJSON string) (*GoogleTTS, error) {
	if config.Voice == "" {
		config.Voice = "ja-JP-Neural2-B" // デフォルト: 日本語女性音声（WaveNet）
	}
	if config.Rate <= 0 {
		config.Rate = 1.0
	}
	if config.Format == "" {
		config.Format = "mp3"
	}

	var opts []option.ClientOption
	if credentialsJSON != "" {
		opts = append(opts, option.WithCredentialsJSON([]byte(credentialsJSON)))
	}
	// credentialsが指定されていない場合は、デフォルトの認証を使用
	// (GOOGLE_APPLICATION_CREDENTIALS環境変数または gcloud auth application-default login)

	client, err := texttospeech.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("Google TTS クライアントの作成に失敗: %w", err)
	}

	return &GoogleTTS{
		config: config,
		client: client,
	}, nil
}

// Close はクライアントをクローズします
func (g *GoogleTTS) Close() error {
	if g.client != nil {
		return g.client.Close()
	}
	return nil
}

// Generate はテキストから音声ファイルを生成します
func (g *GoogleTTS) Generate(ctx context.Context, text string, outputPath string) error {
	// 出力ディレクトリが存在することを確認
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("出力ディレクトリの作成に失敗: %w", err)
	}

	// 音声の言語コードと名前を解析
	languageCode, voiceName := parseVoiceName(g.config.Voice)

	// リクエストを構築
	req := &texttospeechpb.SynthesizeSpeechRequest{
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{
				Text: text,
			},
		},
		Voice: &texttospeechpb.VoiceSelectionParams{
			LanguageCode: languageCode,
			Name:         voiceName,
		},
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding: g.getAudioEncoding(),
			SpeakingRate:  g.config.Rate,
			Pitch:         0.0, // デフォルト
		},
	}

	// Google Cloud TTS APIを呼び出し
	resp, err := g.client.SynthesizeSpeech(ctx, req)
	if err != nil {
		return fmt.Errorf("Google Cloud TTS API 呼び出しエラー: %w", err)
	}

	// 音声データをファイルに保存
	if err := os.WriteFile(outputPath, resp.AudioContent, 0644); err != nil {
		return fmt.Errorf("音声ファイルの保存に失敗: %w", err)
	}

	return nil
}

// GetProvider はプロバイダー名を返します
func (g *GoogleTTS) GetProvider() string {
	return "google_tts"
}

// getAudioEncoding は設定されたフォーマットに応じたAudioEncodingを返します
func (g *GoogleTTS) getAudioEncoding() texttospeechpb.AudioEncoding {
	switch g.config.Format {
	case "mp3":
		return texttospeechpb.AudioEncoding_MP3
	case "ogg":
		return texttospeechpb.AudioEncoding_OGG_OPUS
	case "linear16", "wav":
		return texttospeechpb.AudioEncoding_LINEAR16
	default:
		return texttospeechpb.AudioEncoding_MP3
	}
}

// parseVoiceName は音声名からLanguageCodeとVoiceNameを抽出します
// 例: "ja-JP-Neural2-B" -> ("ja-JP", "ja-JP-Neural2-B")
// 例: "Kyoko" -> ("ja-JP", "ja-JP-Neural2-B")  // macOS say互換
func parseVoiceName(voice string) (languageCode, voiceName string) {
	// Google Cloud TTS形式の音声名（例: ja-JP-Neural2-B）
	if len(voice) > 5 && voice[2] == '-' && voice[5] == '-' {
		languageCode = voice[:5] // "ja-JP"
		voiceName = voice
		return
	}

	// macOS say互換の音声名をマッピング
	switch voice {
	case "Kyoko":
		return "ja-JP", "ja-JP-Neural2-B" // 女性音声
	case "Otoya":
		return "ja-JP", "ja-JP-Neural2-C" // 男性音声
	default:
		// デフォルト: 日本語女性音声
		return "ja-JP", "ja-JP-Neural2-B"
	}
}

// GetCredentialsJSON は環境変数からGoogle Cloud認証情報を取得します
func GetCredentialsJSON(envName string) string {
	if envName == "" {
		envName = "GOOGLE_APPLICATION_CREDENTIALS_JSON"
	}
	return os.Getenv(envName)
}

// 利用可能な日本語音声のリスト（参考）
// WaveNet音声（高品質）:
//   - ja-JP-Neural2-B: 女性
//   - ja-JP-Neural2-C: 男性
//   - ja-JP-Neural2-D: 男性
//
// Standard音声（標準品質）:
//   - ja-JP-Standard-A: 女性
//   - ja-JP-Standard-B: 女性
//   - ja-JP-Standard-C: 男性
//   - ja-JP-Standard-D: 男性
//
// 料金:
//   - WaveNet: $16/100万文字
//   - Standard: $4/100万文字
//   - 無料枠: 毎月100万文字（WaveNetとStandard合計）
