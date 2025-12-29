// +build windows

package tts

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SAPITTS はWindows SAPI (Speech API) による音声合成
type SAPITTS struct {
	config Config
}

// NewSAPITTS は新しいSAPITTSを作成します
func NewSAPITTS(config Config) *SAPITTS {
	return &SAPITTS{
		config: config,
	}
}

// Generate はテキストから音声ファイルを生成します
// Windows SAPI を PowerShell経由で呼び出します
func (s *SAPITTS) Generate(ctx context.Context, text string, outputPath string) error {
	// PowerShellスクリプトを生成
	psScript := s.buildPowerShellScript(text, outputPath)

	// 一時ファイルにスクリプトを保存
	tmpFile, err := os.CreateTemp("", "tts-*.ps1")
	if err != nil {
		return fmt.Errorf("一時ファイルの作成に失敗: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(psScript); err != nil {
		tmpFile.Close()
		return fmt.Errorf("スクリプトの書き込みに失敗: %w", err)
	}
	tmpFile.Close()

	// PowerShellで実行
	cmd := exec.CommandContext(ctx, "powershell", "-ExecutionPolicy", "Bypass", "-File", tmpFile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("SAPI実行に失敗: %w\n出力: %s", err, string(output))
	}

	// 出力ファイルの存在確認
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return fmt.Errorf("音声ファイルの生成に失敗: %s", outputPath)
	}

	return nil
}

// GetProvider はプロバイダー名を返します
func (s *SAPITTS) GetProvider() string {
	return "sapi"
}

// buildPowerShellScript はSAPIを呼び出すPowerShellスクリプトを生成します
func (s *SAPITTS) buildPowerShellScript(text, outputPath string) string {
	// テキスト中の特殊文字をエスケープ
	escapedText := strings.ReplaceAll(text, `"`, `""`)
	escapedText = strings.ReplaceAll(escapedText, "`", "``")

	// 音声名のマッピング（macOS互換性のため）
	voiceName := s.mapVoiceName(s.config.Voice)

	// 速度の設定 (-10 〜 10の範囲、デフォルト0）
	// config.Rate は 0.5 〜 2.0 の範囲と想定し、0を中心にマッピング
	// Rate 1.0 = Speed 0
	// Rate 0.5 = Speed -5
	// Rate 2.0 = Speed 5
	speed := int((s.config.Rate - 1.0) * 5)
	if speed < -10 {
		speed = -10
	}
	if speed > 10 {
		speed = 10
	}

	// 出力形式の決定
	format := s.getAudioFormat()

	// PowerShellスクリプト
	// SAPI.SpVoice と SAPI.SpFileStream を使用してWAVファイル生成
	script := fmt.Sprintf(`
# SAPI (Speech API) を使用した音声合成
Add-Type -AssemblyName System.Speech

$voice = New-Object System.Speech.Synthesis.SpeechSynthesizer
$voice.Rate = %d

# 音声の選択（オプション）
# $voice.SelectVoice("%s")

# WAVファイルに出力
$voice.SetOutputToWaveFile("%s")
$voice.Speak("%s")
$voice.Dispose()

Write-Host "音声ファイルを生成しました: %s"
`, speed, voiceName, outputPath, escapedText, outputPath)

	// MP3形式の場合は追加の変換が必要（ffmpegがあれば）
	if format == "mp3" {
		wavPath := strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".wav"
		script += fmt.Sprintf(`
# MP3に変換（ffmpegが必要）
if (Get-Command ffmpeg -ErrorAction SilentlyContinue) {
    ffmpeg -i "%s" -codec:a libmp3lame -qscale:a 2 "%s" -y
    Remove-Item "%s"
    Write-Host "MP3に変換しました: %s"
} else {
    Write-Warning "ffmpegが見つからないため、WAV形式で出力しました"
}
`, wavPath, outputPath, wavPath, outputPath)
	}

	return script
}

// mapVoiceName はmacOS音声名をWindows音声名にマッピングします
func (s *SAPITTS) mapVoiceName(voice string) string {
	// macOS互換のマッピング
	// 実際のWindows音声名は環境により異なるため、一般的な名前を使用
	mapping := map[string]string{
		"Kyoko": "Microsoft Haruka Desktop",   // 日本語女性音声
		"Otoya": "Microsoft Ichiro Desktop",   // 日本語男性音声
		"Samantha": "Microsoft Zira Desktop",  // 英語女性音声
		"Alex": "Microsoft David Desktop",     // 英語男性音声
	}

	if windowsVoice, ok := mapping[voice]; ok {
		return windowsVoice
	}

	// デフォルトは指定された音声名をそのまま使用
	return voice
}

// getAudioFormat は出力形式を返します
func (s *SAPITTS) getAudioFormat() string {
	switch s.config.Format {
	case "mp3":
		return "mp3"
	case "wav":
		return "wav"
	default:
		return "wav" // デフォルトはWAV
	}
}
