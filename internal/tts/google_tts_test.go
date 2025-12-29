package tts

import (
	"os"
	"testing"
)

func TestParseVoiceName(t *testing.T) {
	tests := []struct {
		name               string
		voice              string
		wantLanguageCode   string
		wantVoiceName      string
	}{
		{
			name:             "Google Cloud TTS形式",
			voice:            "ja-JP-Neural2-B",
			wantLanguageCode: "ja-JP",
			wantVoiceName:    "ja-JP-Neural2-B",
		},
		{
			name:             "macOS say互換 - Kyoko",
			voice:            "Kyoko",
			wantLanguageCode: "ja-JP",
			wantVoiceName:    "ja-JP-Neural2-B",
		},
		{
			name:             "macOS say互換 - Otoya",
			voice:            "Otoya",
			wantLanguageCode: "ja-JP",
			wantVoiceName:    "ja-JP-Neural2-C",
		},
		{
			name:             "不明な音声名",
			voice:            "Unknown",
			wantLanguageCode: "ja-JP",
			wantVoiceName:    "ja-JP-Neural2-B",
		},
		{
			name:             "英語音声",
			voice:            "en-US-Neural2-A",
			wantLanguageCode: "en-US",
			wantVoiceName:    "en-US-Neural2-A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLanguageCode, gotVoiceName := parseVoiceName(tt.voice)

			if gotLanguageCode != tt.wantLanguageCode {
				t.Errorf("parseVoiceName() languageCode = %v, want %v", gotLanguageCode, tt.wantLanguageCode)
			}
			if gotVoiceName != tt.wantVoiceName {
				t.Errorf("parseVoiceName() voiceName = %v, want %v", gotVoiceName, tt.wantVoiceName)
			}
		})
	}
}

func TestGetCredentialsJSON(t *testing.T) {
	tests := []struct {
		name    string
		envName string
		envVal  string
		want    string
	}{
		{
			name:    "デフォルト環境変数名",
			envName: "",
			envVal:  `{"type": "service_account"}`,
			want:    `{"type": "service_account"}`,
		},
		{
			name:    "カスタム環境変数名",
			envName: "CUSTOM_GOOGLE_CREDS",
			envVal:  `{"type": "custom"}`,
			want:    `{"type": "custom"}`,
		},
		{
			name:    "環境変数が未設定",
			envName: "NON_EXISTENT_KEY",
			envVal:  "",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 環境変数をセットアップ
			if tt.envName == "" {
				tt.envName = "GOOGLE_APPLICATION_CREDENTIALS_JSON"
			}

			// 既存の値を保存
			oldVal := os.Getenv(tt.envName)
			defer func() {
				if oldVal != "" {
					os.Setenv(tt.envName, oldVal)
				} else {
					os.Unsetenv(tt.envName)
				}
			}()

			// テスト用の値をセット
			if tt.envVal != "" {
				os.Setenv(tt.envName, tt.envVal)
			} else {
				os.Unsetenv(tt.envName)
			}

			// テスト実行
			got := GetCredentialsJSON(tt.envName)
			if got != tt.want {
				t.Errorf("GetCredentialsJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGoogleTTS_GetProvider(t *testing.T) {
	g := &GoogleTTS{
		config: &Config{Provider: "google_tts"},
	}

	want := "google_tts"
	if got := g.GetProvider(); got != want {
		t.Errorf("GetProvider() = %v, want %v", got, want)
	}
}

func TestGoogleTTS_GetAudioEncoding(t *testing.T) {
	tests := []struct {
		name   string
		format string
	}{
		{"MP3形式", "mp3"},
		{"OGG形式", "ogg"},
		{"WAV形式", "wav"},
		{"LINEAR16形式", "linear16"},
		{"デフォルト（不明な形式）", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &GoogleTTS{
				config: &Config{Format: tt.format},
			}

			// エンコーディング取得（エラーにならないことを確認）
			encoding := g.getAudioEncoding()
			if encoding == 0 {
				t.Error("getAudioEncoding() returned 0 (invalid encoding)")
			}
		})
	}
}

// Note: 実際のGoogle Cloud TTS APIを呼び出すテストは、
// 認証情報が必要なため、統合テストとして別途実装する必要があります。
// ここでは、ヘルパー関数のユニットテストのみを実装しています。
