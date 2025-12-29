package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config はアプリケーション全体の設定を保持します
type Config struct {
	App        AppConfig        `yaml:"app"`
	Slack      SlackConfig      `yaml:"slack"`
	Notion     NotionConfig     `yaml:"notion"`
	GitHub     GitHubConfig     `yaml:"github"`
	Brief      BriefConfig      `yaml:"brief"`
	Summarizer SummarizerConfig `yaml:"summarizer"`
	TTS        TTSConfig        `yaml:"tts"`
	Runtime    RuntimeConfig    `yaml:"runtime"`
}

// AppConfig はアプリケーション基本設定
type AppConfig struct {
	OutputDir string `yaml:"output_dir"`
	LogLevel  string `yaml:"log_level"`
}

// SlackConfig はSlack関連設定
type SlackConfig struct {
	TokenEnv      string          `yaml:"token_env"`
	Token         string          `yaml:"-"` // 環境変数から読み込み
	Channels      []ChannelConfig `yaml:"channels"`
	Filters       FilterConfig    `yaml:"filters"`
	PostChannel   string          `yaml:"post_channel"`
	PostEnabled   bool            `yaml:"post_enabled"`
	UploadAudio   bool            `yaml:"upload_audio"`
}

// ChannelConfig はSlackチャンネル設定
type ChannelConfig struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

// FilterConfig はフィルタリング設定
type FilterConfig struct {
	ExcludeBots          bool     `yaml:"exclude_bots"`
	ExcludeShortMessages bool     `yaml:"exclude_short_messages"`
	MinLength            int      `yaml:"min_length"`
	ExcludeKeywords      []string `yaml:"exclude_keywords"`
}

// NotionConfig はNotion関連設定
type NotionConfig struct {
	TokenEnv  string           `yaml:"token_env"`
	Token     string           `yaml:"-"` // 環境変数から読み込み
	Databases []DatabaseConfig `yaml:"databases"`
}

// DatabaseConfig はNotionデータベース設定
type DatabaseConfig struct {
	ID                string            `yaml:"id"`
	Name              string            `yaml:"name"`
	Properties        []string          `yaml:"properties"`
	FetchPageContent  bool              `yaml:"fetch_page_content"`  // ページ本文を取得するか
	PropertyFilters   map[string]string `yaml:"property_filters"`    // プロパティフィルタ (例: Status: "In Progress")
	CategoryProperty  string            `yaml:"category_property"`   // カテゴリ判定に使うプロパティ
	ProjectProperty   string            `yaml:"project_property"`    // プロジェクト判定に使うプロパティ
}

// GitHubConfig はGitHub関連設定
type GitHubConfig struct {
	Enabled  bool   `yaml:"enabled"`
	TokenEnv string `yaml:"token_env"`
	Token    string `yaml:"-"` // 環境変数から読み込み
	Username string `yaml:"username"`
}

// BriefConfig はブリーフィング設定
type BriefConfig struct {
	DailyWindowHours int `yaml:"daily_window_hours"`
	WeeklyDays       int `yaml:"weekly_days"`
	MaxItemsDaily    int `yaml:"max_items_daily"`
	MaxItemsWeekly   int `yaml:"max_items_weekly"`
}

// SummarizerConfig は要約エンジン設定
type SummarizerConfig struct {
	Provider     string `yaml:"provider"` // "rule" or "gemini" or "openai"
	GeminiModel  string `yaml:"gemini_model"`
	GeminiAPIKey string `yaml:"gemini_api_key_env"` // 環境変数名
	OpenAIModel  string `yaml:"openai_model"`       // "gpt-4o-mini", "gpt-4o", etc.
	OpenAIAPIKey string `yaml:"openai_api_key_env"` // 環境変数名
}

// TTSConfig は音声合成設定
type TTSConfig struct {
	Provider                   string  `yaml:"provider"`                       // "say" or "google_tts"
	Voice                      string  `yaml:"voice"`
	Rate                       float64 `yaml:"rate"`
	Format                     string  `yaml:"format"`                         // "aiff", "m4a", "mp3"
	GoogleCredentialsJSONEnv   string  `yaml:"google_credentials_json_env"`    // Google認証情報JSON環境変数名
}

// RuntimeConfig は実行時設定
type RuntimeConfig struct {
	MaxConcurrency    int `yaml:"max_concurrency"`
	APITimeoutSeconds int `yaml:"api_timeout_seconds"`
}

// Load は設定ファイルを読み込みます
func Load(configPath string) (*Config, error) {
	// 設定ファイルを読み込み
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("設定ファイルの読み込みに失敗: %w", err)
	}

	// YAMLをパース
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("設定ファイルのパースに失敗: %w", err)
	}

	// 環境変数からトークンを読み込み
	if err := cfg.loadTokensFromEnv(); err != nil {
		return nil, err
	}

	// バリデーション
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// loadTokensFromEnv は環境変数からトークンを読み込みます
func (c *Config) loadTokensFromEnv() error {
	// Slackトークン
	if c.Slack.TokenEnv != "" {
		token := os.Getenv(c.Slack.TokenEnv)
		if token == "" {
			return fmt.Errorf("環境変数 %s が設定されていません", c.Slack.TokenEnv)
		}
		c.Slack.Token = token
	}

	// Notionトークン
	if c.Notion.TokenEnv != "" {
		token := os.Getenv(c.Notion.TokenEnv)
		if token == "" {
			return fmt.Errorf("環境変数 %s が設定されていません", c.Notion.TokenEnv)
		}
		c.Notion.Token = token
	}

	// GitHubトークン（オプション）
	if c.GitHub.Enabled && c.GitHub.TokenEnv != "" {
		token := os.Getenv(c.GitHub.TokenEnv)
		if token == "" {
			return fmt.Errorf("環境変数 %s が設定されていません（GitHub連携が有効です）", c.GitHub.TokenEnv)
		}
		c.GitHub.Token = token
	}

	// Gemini API Key（オプション）
	if c.Summarizer.Provider == "gemini" && c.Summarizer.GeminiAPIKey != "" {
		apiKey := os.Getenv(c.Summarizer.GeminiAPIKey)
		if apiKey == "" {
			return fmt.Errorf("環境変数 %s が設定されていません（Gemini providerを使用する場合は必須）", c.Summarizer.GeminiAPIKey)
		}
	}

	// OpenAI API Key（オプション）
	if c.Summarizer.Provider == "openai" && c.Summarizer.OpenAIAPIKey != "" {
		apiKey := os.Getenv(c.Summarizer.OpenAIAPIKey)
		if apiKey == "" {
			return fmt.Errorf("環境変数 %s が設定されていません（OpenAI Summarizer使用時は必須）", c.Summarizer.OpenAIAPIKey)
		}
	}

	// OpenAI API Key for TTS（オプション）
	if c.TTS.Provider == "openai_tts" {
		apiKey := os.Getenv("VOICE_BRIEF_OPENAI_API_KEY")
		if apiKey == "" {
			return fmt.Errorf("環境変数 VOICE_BRIEF_OPENAI_API_KEY が設定されていません（OpenAI TTS使用時は必須）")
		}
	}

	return nil
}

// Validate は設定の妥当性を検証します
func (c *Config) Validate() error {
	// App設定の検証
	if c.App.OutputDir == "" {
		return fmt.Errorf("app.output_dir が設定されていません")
	}
	if c.App.LogLevel == "" {
		c.App.LogLevel = "info" // デフォルト値
	}
	validLogLevels := []string{"debug", "info", "warn", "error"}
	if !contains(validLogLevels, c.App.LogLevel) {
		return fmt.Errorf("app.log_level は debug, info, warn, error のいずれかである必要があります")
	}

	// Slack設定の検証
	if c.Slack.TokenEnv == "" {
		return fmt.Errorf("slack.token_env が設定されていません")
	}
	if len(c.Slack.Channels) == 0 {
		return fmt.Errorf("slack.channels が設定されていません（最低1つは必要）")
	}
	for i, ch := range c.Slack.Channels {
		if ch.ID == "" {
			return fmt.Errorf("slack.channels[%d].id が設定されていません", i)
		}
		if !strings.HasPrefix(ch.ID, "C") && !strings.HasPrefix(ch.ID, "G") {
			return fmt.Errorf("slack.channels[%d].id は 'C' または 'G' で始まる必要があります", i)
		}
	}

	// Notion設定の検証
	if c.Notion.TokenEnv == "" {
		return fmt.Errorf("notion.token_env が設定されていません")
	}
	if len(c.Notion.Databases) == 0 {
		return fmt.Errorf("notion.databases が設定されていません（最低1つは必要）")
	}
	for i, db := range c.Notion.Databases {
		if db.ID == "" {
			return fmt.Errorf("notion.databases[%d].id が設定されていません", i)
		}
	}

	// Brief設定の検証
	if c.Brief.DailyWindowHours <= 0 {
		c.Brief.DailyWindowHours = 24 // デフォルト値
	}
	if c.Brief.WeeklyDays <= 0 {
		c.Brief.WeeklyDays = 7 // デフォルト値
	}
	if c.Brief.MaxItemsDaily <= 0 {
		c.Brief.MaxItemsDaily = 8 // デフォルト値
	}
	if c.Brief.MaxItemsWeekly <= 0 {
		c.Brief.MaxItemsWeekly = 25 // デフォルト値
	}

	// Summarizer設定の検証
	if c.Summarizer.Provider == "" {
		c.Summarizer.Provider = "rule" // デフォルト値
	}
	validSummarizerProviders := []string{"rule", "gemini", "openai"}
	if !contains(validSummarizerProviders, c.Summarizer.Provider) {
		return fmt.Errorf("summarizer.provider は rule, gemini, openai のいずれかである必要があります")
	}
	// Gemini使用時のデフォルトモデル
	if c.Summarizer.Provider == "gemini" && c.Summarizer.GeminiModel == "" {
		c.Summarizer.GeminiModel = "gemini-2.0-flash-exp" // デフォルト値
	}
	// OpenAI使用時のデフォルトモデル
	if c.Summarizer.Provider == "openai" && c.Summarizer.OpenAIModel == "" {
		c.Summarizer.OpenAIModel = "gpt-4o-mini" // デフォルト値（コスト効率重視）
	}

	// TTS設定の検証
	if c.TTS.Provider == "" {
		c.TTS.Provider = "say" // デフォルト値
	}
	validTTSProviders := []string{"say", "sapi", "google_tts", "openai_tts"}
	if !contains(validTTSProviders, c.TTS.Provider) {
		return fmt.Errorf("tts.provider は say, sapi, google_tts, openai_tts のいずれかである必要があります")
	}
	if c.TTS.Voice == "" {
		c.TTS.Voice = "Kyoko" // デフォルト値
	}
	if c.TTS.Rate <= 0 {
		c.TTS.Rate = 1.0 // デフォルト値
	}
	if c.TTS.Format == "" {
		c.TTS.Format = "m4a" // デフォルト値
	}
	validFormats := []string{"aiff", "m4a", "mp3"}
	if !contains(validFormats, c.TTS.Format) {
		return fmt.Errorf("tts.format は aiff, m4a, mp3 のいずれかである必要があります")
	}

	// Runtime設定の検証
	if c.Runtime.MaxConcurrency <= 0 {
		c.Runtime.MaxConcurrency = 5 // デフォルト値
	}
	if c.Runtime.APITimeoutSeconds <= 0 {
		c.Runtime.APITimeoutSeconds = 30 // デフォルト値
	}

	return nil
}

// ValidateTokens はトークンの形式を検証します
func (c *Config) ValidateTokens() error {
	// Slackトークンの形式確認
	if c.Slack.Token != "" {
		if !strings.HasPrefix(c.Slack.Token, "xoxb-") {
			return fmt.Errorf("Slack Bot Token は 'xoxb-' で始まる必要があります")
		}
	}

	// Notionトークンの形式確認
	if c.Notion.Token != "" {
		if !strings.HasPrefix(c.Notion.Token, "secret_") {
			return fmt.Errorf("Notion Token は 'secret_' で始まる必要があります")
		}
	}

	// GitHubトークンの形式確認（オプション）
	if c.GitHub.Enabled && c.GitHub.Token != "" {
		if !strings.HasPrefix(c.GitHub.Token, "ghp_") && !strings.HasPrefix(c.GitHub.Token, "github_pat_") {
			return fmt.Errorf("GitHub Token は 'ghp_' または 'github_pat_' で始まる必要があります")
		}
	}

	return nil
}

// contains はスライスに要素が含まれているかチェックします
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
