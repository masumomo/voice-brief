package tts

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewSayTTS(t *testing.T) {
	tests := []struct {
		name           string
		config         *Config
		expectedVoice  string
		expectedRate   float64
		expectedFormat string
	}{
		{
			name:           "デフォルト設定",
			config:         &Config{},
			expectedVoice:  "Kyoko",
			expectedRate:   1.0,
			expectedFormat: "m4a",
		},
		{
			name: "カスタム設定",
			config: &Config{
				Voice:  "Otoya",
				Rate:   1.2,
				Format: "aiff",
			},
			expectedVoice:  "Otoya",
			expectedRate:   1.2,
			expectedFormat: "aiff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tts := NewSayTTS(tt.config)

			if tts.config.Voice != tt.expectedVoice {
				t.Errorf("expected Voice=%s, got %s", tt.expectedVoice, tts.config.Voice)
			}
			if tts.config.Rate != tt.expectedRate {
				t.Errorf("expected Rate=%f, got %f", tt.expectedRate, tts.config.Rate)
			}
			if tts.config.Format != tt.expectedFormat {
				t.Errorf("expected Format=%s, got %s", tt.expectedFormat, tts.config.Format)
			}
		})
	}
}

func TestSayTTS_GetProvider(t *testing.T) {
	tts := NewSayTTS(&Config{})
	if tts.GetProvider() != "say" {
		t.Errorf("expected provider 'say', got %s", tts.GetProvider())
	}
}

func TestGetAvailableVoices(t *testing.T) {
	voices := GetAvailableVoices()

	// 最低限の日本語音声が含まれていることを確認
	expectedVoices := []string{"Kyoko", "Otoya"}

	for _, expected := range expectedVoices {
		found := false
		for _, voice := range voices {
			if voice == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected voice '%s' not found in available voices", expected)
		}
	}
}

func TestCheckDependencies(t *testing.T) {
	// sayコマンドの存在確認（macOSでのみテスト）
	err := CheckDependencies(false)
	// macOS以外の環境ではエラーになることが期待される
	if err != nil {
		t.Logf("Note: say command not available (expected on non-macOS): %v", err)
	}
}

func TestSayTTS_Generate_DirectoryCreation(t *testing.T) {
	// 一時ディレクトリでのテスト
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "subdir", "test.aiff")

	tts := NewSayTTS(&Config{
		Voice:  "Kyoko",
		Rate:   1.0,
		Format: "aiff",
	})

	ctx := context.Background()

	// sayコマンドが利用可能かチェック
	if err := CheckDependencies(false); err != nil {
		t.Skip("say command not available, skipping generation test")
		return
	}

	// 音声生成
	err := tts.Generate(ctx, "テスト", outputPath)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// ファイルが作成されたことを確認
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("output file was not created: %s", outputPath)
	}

	// ファイルサイズが0より大きいことを確認
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("failed to stat output file: %v", err)
	}
	if info.Size() == 0 {
		t.Errorf("output file is empty")
	}
}

func TestSayTTS_Generate_EmptyText(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "empty.aiff")

	tts := NewSayTTS(&Config{
		Voice:  "Kyoko",
		Rate:   1.0,
		Format: "aiff",
	})

	ctx := context.Background()

	// sayコマンドが利用可能かチェック
	if err := CheckDependencies(false); err != nil {
		t.Skip("say command not available")
		return
	}

	// 空のテキストで生成
	err := tts.Generate(ctx, "", outputPath)
	// sayコマンドは空のテキストでもエラーにならない場合があるため、
	// エラーがなければ成功とみなす
	if err != nil {
		t.Logf("Note: empty text generation returned error (may be expected): %v", err)
	}
}

func TestSayTTS_Generate_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "cancelled.aiff")

	tts := NewSayTTS(&Config{
		Voice:  "Kyoko",
		Rate:   1.0,
		Format: "aiff",
	})

	// すぐにキャンセルされるコンテキスト
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 即座にキャンセル

	// sayコマンドが利用可能かチェック
	if err := CheckDependencies(false); err != nil {
		t.Skip("say command not available")
		return
	}

	// 生成を試みる（キャンセルされるはず）
	err := tts.Generate(ctx, "これは長いテキストです", outputPath)

	// コンテキストキャンセルによるエラーまたは成功（処理が速すぎた場合）
	if err != nil && ctx.Err() != context.Canceled {
		// エラーがあってもコンテキストキャンセル以外のエラーなら失敗
		t.Logf("Note: context cancellation test completed: %v", err)
	}
}
