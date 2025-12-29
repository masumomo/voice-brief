// +build !darwin

package tts

import (
	"context"
	"fmt"
)

// SayTTS はmacOSのみで実装されます（このファイルはスタブ）
type SayTTS struct {
	config *Config
}

// NewSayTTS はmacOS以外では利用不可のスタブを返します
func NewSayTTS(config *Config) *SayTTS {
	return &SayTTS{config: config}
}

// Generate はmacOS以外では常にエラーを返します
func (s *SayTTS) Generate(ctx context.Context, text string, outputPath string) error {
	return fmt.Errorf("say TTS はmacOSでのみ利用可能です")
}

// GetProvider はプロバイダー名を返します
func (s *SayTTS) GetProvider() string {
	return "say"
}
