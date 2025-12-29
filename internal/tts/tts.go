package tts

import (
	"context"
)

// TTS は音声合成のインターフェース
type TTS interface {
	// Generate はテキストから音声ファイルを生成します
	Generate(ctx context.Context, text string, outputPath string) error

	// GetProvider はプロバイダー名を返します
	GetProvider() string
}

// Config はTTS設定
type Config struct {
	Provider string  // "say" (macOS) | "sapi" (Windows) | "google_tts" | "openai_tts"
	Voice    string  // 音声名（say/sapi: Kyoko, Otoya等）
	Rate     float64 // 読み上げ速度
	Format   string  // 出力形式: "aiff", "m4a", "mp3", "wav"
}
