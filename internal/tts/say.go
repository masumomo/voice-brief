// +build darwin

package tts

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SayTTS はmacOS標準のsayコマンドを使用したTTS
type SayTTS struct {
	config *Config
}

// NewSayTTS は新しいSayTTSを作成します
func NewSayTTS(config *Config) *SayTTS {
	if config.Voice == "" {
		config.Voice = "Kyoko"
	}
	if config.Rate <= 0 {
		config.Rate = 1.0
	}
	if config.Format == "" {
		config.Format = "m4a"
	}
	return &SayTTS{
		config: config,
	}
}

// Generate はテキストから音声ファイルを生成します
func (s *SayTTS) Generate(ctx context.Context, text string, outputPath string) error {
	// 出力ディレクトリが存在することを確認
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("出力ディレクトリの作成に失敗: %w", err)
	}

	// sayコマンドの存在確認
	if _, err := exec.LookPath("say"); err != nil {
		return fmt.Errorf("sayコマンドが見つかりません（macOS専用機能）: %w", err)
	}

	// まずAIFF形式で生成
	aiffPath := strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".aiff"

	// sayコマンド実行
	args := []string{
		"-v", s.config.Voice,
		"-r", fmt.Sprintf("%.0f", s.config.Rate*200), // sayのrateは単語/分（デフォルト200）
		"-o", aiffPath,
		text,
	}

	cmd := exec.CommandContext(ctx, "say", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sayコマンドの実行に失敗: %w, output: %s", err, string(output))
	}

	fmt.Printf("✓ AIFF音声ファイルを生成: %s\n", aiffPath)

	// AIFF形式のままでよければ完了
	if s.config.Format == "aiff" {
		if aiffPath != outputPath {
			if err := os.Rename(aiffPath, outputPath); err != nil {
				return fmt.Errorf("ファイル名の変更に失敗: %w", err)
			}
		}
		return nil
	}

	// M4AまたはMP3に変換
	if err := s.convertAudio(ctx, aiffPath, outputPath); err != nil {
		return err
	}

	// 変換後、元のAIFFファイルを削除
	if err := os.Remove(aiffPath); err != nil {
		fmt.Printf("⚠️  警告: 一時ファイルの削除に失敗: %v\n", err)
	}

	return nil
}

// convertAudio はAIFFを指定形式に変換します
func (s *SayTTS) convertAudio(ctx context.Context, inputPath, outputPath string) error {
	// ffmpegの存在確認
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		// ffmpegがない場合はAIFFのままにする
		fmt.Printf("⚠️  警告: ffmpegが見つかりません。AIFF形式のまま出力します\n")
		if err := os.Rename(inputPath, outputPath); err != nil {
			return fmt.Errorf("ファイル名の変更に失敗: %w", err)
		}
		return nil
	}

	// 変換コマンド
	var args []string
	switch s.config.Format {
	case "m4a":
		args = []string{
			"-i", inputPath,
			"-c:a", "aac",
			"-b:a", "64k",
			"-y", // 上書き
			outputPath,
		}
	case "mp3":
		args = []string{
			"-i", inputPath,
			"-c:a", "libmp3lame",
			"-b:a", "128k",
			"-y", // 上書き
			outputPath,
		}
	default:
		return fmt.Errorf("未対応の出力形式: %s", s.config.Format)
	}

	cmd := exec.CommandContext(ctx, ffmpegPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpegでの変換に失敗: %w, output: %s", err, string(output))
	}

	fmt.Printf("✓ %s形式に変換完了: %s\n", strings.ToUpper(s.config.Format), outputPath)
	return nil
}

// GetProvider はプロバイダー名を返します
func (s *SayTTS) GetProvider() string {
	return "say"
}

// CheckDependencies はsayコマンドとffmpegの依存関係をチェックします
func CheckDependencies(requireFFmpeg bool) error {
	// sayコマンドの確認
	if _, err := exec.LookPath("say"); err != nil {
		return fmt.Errorf("sayコマンドが見つかりません（macOS専用機能）")
	}

	// ffmpegの確認（オプション）
	if requireFFmpeg {
		if _, err := exec.LookPath("ffmpeg"); err != nil {
			return fmt.Errorf("ffmpegが見つかりません（brew install ffmpeg でインストール可能）")
		}
	}

	return nil
}

// GetAvailableVoices は利用可能な音声のリストを返します（簡易版）
func GetAvailableVoices() []string {
	// macOSの主要な日本語音声
	return []string{
		"Kyoko",  // 日本語女性
		"Otoya",  // 日本語男性
		"Samantha", // 英語女性
		"Alex",     // 英語男性
	}
}
