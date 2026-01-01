package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config はアプリケーション全体の設定を保持します
type Config struct {
	App             AppConfig             `yaml:"app"`
	Slack           SlackConfig           `yaml:"slack"`
	Notion          NotionConfig          `yaml:"notion"`
	GitHub          GitHubConfig          `yaml:"github"`
	Brief           BriefConfig           `yaml:"brief"`
	Importance      ImportanceConfig      `yaml:"importance"`
	Categorizer     CategorizerConfig     `yaml:"categorizer"`
	EventSummarizer EventSummarizerConfig `yaml:"event_summarizer"`
	BriefSummarizer BriefSummarizerConfig `yaml:"brief_summarizer"`
	TTS             TTSConfig             `yaml:"tts"`
	Runtime         RuntimeConfig         `yaml:"runtime"`
}

// AppConfig はアプリケーション基本設定
type AppConfig struct {
	OutputDir string `yaml:"output_dir"`
	LogLevel  string `yaml:"log_level"`
	Timezone  string `yaml:"timezone"` // タイムゾーン（例: "Asia/Tokyo"）
	Location  *time.Location `yaml:"-"` // パース済みタイムゾーン
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
	ID               string            `yaml:"id"`
	Name             string            `yaml:"name"`
	Properties       []string          `yaml:"properties"`
	MaxContentBlocks *int              `yaml:"max_content_blocks"`  // 取得する本文ブロック数（0:取得しない、未指定:デフォルト5）
	PropertyFilters  map[string]string `yaml:"property_filters"`    // プロパティフィルタ (例: Status: "In Progress")
	CategoryProperty string            `yaml:"category_property"`   // カテゴリ判定に使うプロパティ
	ProjectProperty  string            `yaml:"project_property"`    // プロジェクト判定に使うプロパティ
}

// GetMaxContentBlocks は取得する本文ブロック数を返します（デフォルト: 5）
func (d *DatabaseConfig) GetMaxContentBlocks() int {
	if d.MaxContentBlocks == nil {
		return 5 // デフォルト5ブロック
	}
	return *d.MaxContentBlocks
}

// GitHubConfig はGitHub関連設定
type GitHubConfig struct {
	Enabled      bool     `yaml:"enabled"`
	TokenEnv     string   `yaml:"token_env"`
	Token        string   `yaml:"-"` // 環境変数から読み込み
	Username     string   `yaml:"username"`      // フィルタ用ユーザー名
	Repositories []string `yaml:"repositories"`  // 監視対象リポジトリ（owner/repo形式）
}

// BriefConfig はブリーフィング設定
type BriefConfig struct {
	DailyWindowHours int `yaml:"daily_window_hours"`
	WeeklyDays       int `yaml:"weekly_days"`
	MaxItemsDaily    int `yaml:"max_items_daily"`
	MaxItemsWeekly   int `yaml:"max_items_weekly"`
}

// ImportanceConfig は重要度計算の重み設定
type ImportanceConfig struct {
	Weights ImportanceWeights `yaml:"weights"`
}

// ImportanceWeights は各特徴量の重み
type ImportanceWeights struct {
	// テキスト特徴量
	TextLength     *float64 `yaml:"text_length"`
	TitleLength    *float64 `yaml:"title_length"`
	HasMention     *float64 `yaml:"has_mention"`
	HasQuestion    *float64 `yaml:"has_question"`
	HasExclamation *float64 `yaml:"has_exclamation"`
	HasURL         *float64 `yaml:"has_url"`
	KeywordScore   *float64 `yaml:"keyword_score"`

	// エンゲージメント特徴量
	CommentCount     *float64 `yaml:"comment_count"`
	UniqueCommenters *float64 `yaml:"unique_commenters"`
	TotalCommentLen  *float64 `yaml:"total_comment_len"`

	// カテゴリ特徴量
	IsIncident *float64 `yaml:"is_incident"`
	IsDev      *float64 `yaml:"is_dev"`
	IsBiz      *float64 `yaml:"is_biz"`
	IsOps      *float64 `yaml:"is_ops"`

	// ソース特徴量
	IsSlack  *float64 `yaml:"is_slack"`
	IsNotion *float64 `yaml:"is_notion"`
	IsGitHub *float64 `yaml:"is_github"`

	// 時間特徴量
	HourOfDay       *float64 `yaml:"hour_of_day"`
	IsBusinessHours *float64 `yaml:"is_business_hours"`
	IsWeekday       *float64 `yaml:"is_weekday"`

	// バイアス
	Bias *float64 `yaml:"bias"`
}

// GetWithDefaults はデフォルト値を適用した重みを返します
func (w *ImportanceWeights) GetWithDefaults() *ImportanceWeights {
	result := &ImportanceWeights{}

	// デフォルト値
	defaults := map[string]float64{
		"text_length":       5.0,
		"title_length":      2.0,
		"has_mention":       15.0,
		"has_question":      5.0,
		"has_exclamation":   3.0,
		"has_url":           2.0,
		"keyword_score":     20.0,
		"comment_count":     15.0,
		"unique_commenters": 10.0,
		"total_comment_len": 5.0,
		"is_incident":       30.0,
		"is_dev":            5.0,
		"is_biz":            3.0,
		"is_ops":            8.0,
		"is_slack":          0.0,
		"is_notion":         2.0,
		"is_github":         3.0,
		"hour_of_day":       0.0,
		"is_business_hours": 2.0,
		"is_weekday":        1.0,
		"bias":              30.0,
	}

	result.TextLength = getOrDefault(w.TextLength, defaults["text_length"])
	result.TitleLength = getOrDefault(w.TitleLength, defaults["title_length"])
	result.HasMention = getOrDefault(w.HasMention, defaults["has_mention"])
	result.HasQuestion = getOrDefault(w.HasQuestion, defaults["has_question"])
	result.HasExclamation = getOrDefault(w.HasExclamation, defaults["has_exclamation"])
	result.HasURL = getOrDefault(w.HasURL, defaults["has_url"])
	result.KeywordScore = getOrDefault(w.KeywordScore, defaults["keyword_score"])
	result.CommentCount = getOrDefault(w.CommentCount, defaults["comment_count"])
	result.UniqueCommenters = getOrDefault(w.UniqueCommenters, defaults["unique_commenters"])
	result.TotalCommentLen = getOrDefault(w.TotalCommentLen, defaults["total_comment_len"])
	result.IsIncident = getOrDefault(w.IsIncident, defaults["is_incident"])
	result.IsDev = getOrDefault(w.IsDev, defaults["is_dev"])
	result.IsBiz = getOrDefault(w.IsBiz, defaults["is_biz"])
	result.IsOps = getOrDefault(w.IsOps, defaults["is_ops"])
	result.IsSlack = getOrDefault(w.IsSlack, defaults["is_slack"])
	result.IsNotion = getOrDefault(w.IsNotion, defaults["is_notion"])
	result.IsGitHub = getOrDefault(w.IsGitHub, defaults["is_github"])
	result.HourOfDay = getOrDefault(w.HourOfDay, defaults["hour_of_day"])
	result.IsBusinessHours = getOrDefault(w.IsBusinessHours, defaults["is_business_hours"])
	result.IsWeekday = getOrDefault(w.IsWeekday, defaults["is_weekday"])
	result.Bias = getOrDefault(w.Bias, defaults["bias"])

	return result
}

// getOrDefault はポインタがnilの場合デフォルト値を返します
func getOrDefault(ptr *float64, defaultVal float64) *float64 {
	if ptr != nil {
		return ptr
	}
	return &defaultVal
}

// CategorizerConfig はカテゴリ判定エンジン設定
type CategorizerConfig struct {
	Provider     string `yaml:"provider"`            // "rule" or "gemini" or "openai"
	GeminiModel  string `yaml:"gemini_model"`        // Gemini使用時のモデル
	GeminiAPIKey string `yaml:"gemini_api_key_env"`  // 環境変数名
	OpenAIModel  string `yaml:"openai_model"`        // OpenAI使用時のモデル
	OpenAIAPIKey string `yaml:"openai_api_key_env"`  // 環境変数名
}

// EventSummarizerConfig はイベント要約エンジン設定
type EventSummarizerConfig struct {
	Provider      string `yaml:"provider"`            // "rule" or "gemini" or "openai"
	MaxSummaryLen int    `yaml:"max_summary_len"`     // 要約の最大文字数
	GeminiModel   string `yaml:"gemini_model"`        // Gemini使用時のモデル
	GeminiAPIKey  string `yaml:"gemini_api_key_env"`  // 環境変数名
	OpenAIModel   string `yaml:"openai_model"`        // OpenAI使用時のモデル
	OpenAIAPIKey  string `yaml:"openai_api_key_env"`  // 環境変数名
}

// BriefSummarizerConfig はブリーフィング要約エンジン設定
type BriefSummarizerConfig struct {
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
	if c.BriefSummarizer.Provider == "gemini" && c.BriefSummarizer.GeminiAPIKey != "" {
		apiKey := os.Getenv(c.BriefSummarizer.GeminiAPIKey)
		if apiKey == "" {
			return fmt.Errorf("環境変数 %s が設定されていません（Gemini providerを使用する場合は必須）", c.BriefSummarizer.GeminiAPIKey)
		}
	}

	// OpenAI API Key（オプション）
	if c.BriefSummarizer.Provider == "openai" && c.BriefSummarizer.OpenAIAPIKey != "" {
		apiKey := os.Getenv(c.BriefSummarizer.OpenAIAPIKey)
		if apiKey == "" {
			return fmt.Errorf("環境変数 %s が設定されていません（OpenAI BriefSummarizer使用時は必須）", c.BriefSummarizer.OpenAIAPIKey)
		}
	}

	// Categorizer Gemini API Key（オプション）
	if c.Categorizer.Provider == "gemini" && c.Categorizer.GeminiAPIKey != "" {
		apiKey := os.Getenv(c.Categorizer.GeminiAPIKey)
		if apiKey == "" {
			return fmt.Errorf("環境変数 %s が設定されていません（Gemini Categorizer使用時は必須）", c.Categorizer.GeminiAPIKey)
		}
	}

	// Categorizer OpenAI API Key（オプション）
	if c.Categorizer.Provider == "openai" && c.Categorizer.OpenAIAPIKey != "" {
		apiKey := os.Getenv(c.Categorizer.OpenAIAPIKey)
		if apiKey == "" {
			return fmt.Errorf("環境変数 %s が設定されていません（OpenAI Categorizer使用時は必須）", c.Categorizer.OpenAIAPIKey)
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

	// タイムゾーンの検証とパース
	if c.App.Timezone == "" {
		c.App.Timezone = "Local" // デフォルト: システムのローカルタイムゾーン
	}
	if c.App.Timezone == "Local" {
		c.App.Location = time.Local
	} else {
		loc, err := time.LoadLocation(c.App.Timezone)
		if err != nil {
			return fmt.Errorf("app.timezone が不正です: %w", err)
		}
		c.App.Location = loc
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

	// Categorizer設定の検証
	if c.Categorizer.Provider == "" {
		c.Categorizer.Provider = "rule" // デフォルト値
	}
	validCategorizerProviders := []string{"rule", "gemini", "openai"}
	if !contains(validCategorizerProviders, c.Categorizer.Provider) {
		return fmt.Errorf("categorizer.provider は rule, gemini, openai のいずれかである必要があります")
	}
	// Gemini使用時のデフォルトモデル
	if c.Categorizer.Provider == "gemini" && c.Categorizer.GeminiModel == "" {
		c.Categorizer.GeminiModel = "gemini-2.0-flash-exp" // デフォルト値
	}
	// OpenAI使用時のデフォルトモデル
	if c.Categorizer.Provider == "openai" && c.Categorizer.OpenAIModel == "" {
		c.Categorizer.OpenAIModel = "gpt-4o-mini" // デフォルト値（コスト効率重視）
	}

	// EventSummarizer設定の検証
	if c.EventSummarizer.Provider == "" {
		c.EventSummarizer.Provider = "rule" // デフォルト値
	}
	validEventSummarizerProviders := []string{"rule", "gemini", "openai"}
	if !contains(validEventSummarizerProviders, c.EventSummarizer.Provider) {
		return fmt.Errorf("event_summarizer.provider は rule, gemini, openai のいずれかである必要があります")
	}
	if c.EventSummarizer.MaxSummaryLen <= 0 {
		c.EventSummarizer.MaxSummaryLen = 200 // デフォルト値
	}
	// Gemini使用時のデフォルトモデル
	if c.EventSummarizer.Provider == "gemini" && c.EventSummarizer.GeminiModel == "" {
		c.EventSummarizer.GeminiModel = "gemini-2.0-flash-exp"
	}
	// OpenAI使用時のデフォルトモデル
	if c.EventSummarizer.Provider == "openai" && c.EventSummarizer.OpenAIModel == "" {
		c.EventSummarizer.OpenAIModel = "gpt-4o-mini"
	}

	// BriefSummarizer設定の検証
	if c.BriefSummarizer.Provider == "" {
		c.BriefSummarizer.Provider = "rule" // デフォルト値
	}
	validBriefSummarizerProviders := []string{"rule", "gemini", "openai"}
	if !contains(validBriefSummarizerProviders, c.BriefSummarizer.Provider) {
		return fmt.Errorf("brief_summarizer.provider は rule, gemini, openai のいずれかである必要があります")
	}
	// Gemini使用時のデフォルトモデル
	if c.BriefSummarizer.Provider == "gemini" && c.BriefSummarizer.GeminiModel == "" {
		c.BriefSummarizer.GeminiModel = "gemini-2.0-flash-exp" // デフォルト値
	}
	// OpenAI使用時のデフォルトモデル
	if c.BriefSummarizer.Provider == "openai" && c.BriefSummarizer.OpenAIModel == "" {
		c.BriefSummarizer.OpenAIModel = "gpt-4o-mini" // デフォルト値（コスト効率重視）
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
