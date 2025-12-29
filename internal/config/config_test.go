package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	// テスト用の設定ファイルを作成
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
app:
  output_dir: "./out"
  log_level: "info"

slack:
  token_env: "TEST_SLACK_TOKEN"
  channels:
    - id: "C01234567"
      name: "general"

notion:
  token_env: "TEST_NOTION_TOKEN"
  databases:
    - id: "db-uuid-xxxx"
      name: "Test DB"
      properties: ["Status"]

github:
  enabled: false

brief:
  daily_window_hours: 24
  weekly_days: 7
  max_items_daily: 8
  max_items_weekly: 25

summarizer:
  provider: "rule"

tts:
  provider: "say"
  voice: "Kyoko"
  rate: 1.1
  format: "m4a"

runtime:
  max_concurrency: 5
  api_timeout_seconds: 30
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("テスト用設定ファイルの作成に失敗: %v", err)
	}

	// 環境変数を設定
	os.Setenv("TEST_SLACK_TOKEN", "xoxb-test-token")
	os.Setenv("TEST_NOTION_TOKEN", "secret_test-token")
	defer func() {
		os.Unsetenv("TEST_SLACK_TOKEN")
		os.Unsetenv("TEST_NOTION_TOKEN")
	}()

	// 設定ファイルを読み込み
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("設定の読み込みに失敗: %v", err)
	}

	// 検証
	if cfg.App.OutputDir != "./out" {
		t.Errorf("App.OutputDir = %s; want ./out", cfg.App.OutputDir)
	}
	if cfg.Slack.Token != "xoxb-test-token" {
		t.Errorf("Slack.Token = %s; want xoxb-test-token", cfg.Slack.Token)
	}
	if cfg.Notion.Token != "secret_test-token" {
		t.Errorf("Notion.Token = %s; want secret_test-token", cfg.Notion.Token)
	}
	if len(cfg.Slack.Channels) != 1 {
		t.Errorf("len(Slack.Channels) = %d; want 1", len(cfg.Slack.Channels))
	}
	if cfg.TTS.Provider != "say" {
		t.Errorf("TTS.Provider = %s; want say", cfg.TTS.Provider)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				App: AppConfig{
					OutputDir: "./out",
					LogLevel:  "info",
				},
				Slack: SlackConfig{
					TokenEnv: "SLACK_TOKEN",
					Token:    "xoxb-test",
					Channels: []ChannelConfig{
						{ID: "C123456", Name: "test"},
					},
				},
				Notion: NotionConfig{
					TokenEnv: "NOTION_TOKEN",
					Token:    "secret_test",
					Databases: []DatabaseConfig{
						{ID: "db-123", Name: "test"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing output_dir",
			cfg: Config{
				App: AppConfig{
					LogLevel: "info",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid log_level",
			cfg: Config{
				App: AppConfig{
					OutputDir: "./out",
					LogLevel:  "invalid",
				},
			},
			wantErr: true,
		},
		{
			name: "missing slack channels",
			cfg: Config{
				App: AppConfig{
					OutputDir: "./out",
					LogLevel:  "info",
				},
				Slack: SlackConfig{
					TokenEnv: "SLACK_TOKEN",
					Channels: []ChannelConfig{},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateTokens(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid tokens",
			cfg: Config{
				Slack: SlackConfig{
					Token: "xoxb-123456-abcdef",
				},
				Notion: NotionConfig{
					Token: "secret_abc123",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid slack token",
			cfg: Config{
				Slack: SlackConfig{
					Token: "invalid-token",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid notion token",
			cfg: Config{
				Notion: NotionConfig{
					Token: "invalid-token",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.ValidateTokens()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTokens() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
