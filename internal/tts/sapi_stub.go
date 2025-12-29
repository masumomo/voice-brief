// +build !windows

package tts

import (
	"context"
	"fmt"
)

// SAPITTS はWindowsのみで実装されます（このファイルはスタブ）
type SAPITTS struct {
	config Config
}

// NewSAPITTS はWindows以外では利用不可のスタブを返します
func NewSAPITTS(config Config) *SAPITTS {
	return &SAPITTS{config: config}
}

// Generate はWindows以外では常にエラーを返します
func (s *SAPITTS) Generate(ctx context.Context, text string, outputPath string) error {
	return fmt.Errorf("SAPI TTS はWindowsでのみ利用可能です")
}

// GetProvider はプロバイダー名を返します
func (s *SAPITTS) GetProvider() string {
	return "sapi"
}
