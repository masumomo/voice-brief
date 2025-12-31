package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/masumomo/voice-brief/internal/config"
	"github.com/masumomo/voice-brief/internal/fetcher"
	"github.com/masumomo/voice-brief/internal/logger"
	"github.com/masumomo/voice-brief/internal/model"
	"github.com/masumomo/voice-brief/internal/summarizer"
	"github.com/masumomo/voice-brief/internal/tts"
	"github.com/masumomo/voice-brief/internal/uploader"
)

const version = "v0.1.0-dev"

func main() {
	// .envファイルを読み込み（存在しない場合は無視）
	_ = godotenv.Load()

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	command := os.Args[1]

	switch command {
	case "version":
		fmt.Println("VoiceBrief", version)
		fmt.Println("音声で聞く、チームの最新情報")
	case "config":
		if len(os.Args) < 3 {
			fmt.Println("エラー: サブコマンドを指定してください (例: config check)")
			os.Exit(1)
		}
		handleConfigCommand(os.Args[2])
	case "run":
		handleRunCommand()
	case "doctor":
		handleDoctorCommand()
	default:
		fmt.Printf("エラー: 不明なコマンド '%s'\n", command)
		printUsage()
		os.Exit(1)
	}
}

func handleRunCommand() {
	// フラグの解析
	var isDaily, isWeekly bool
	var daysBack int
	var outDir string
	var dryRun bool
	var debugDump bool
	var logLevel string

	for i := 2; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch arg {
		case "--daily":
			isDaily = true
		case "--weekly":
			isWeekly = true
		case "--days":
			if i+1 < len(os.Args) {
				if n, err := fmt.Sscanf(os.Args[i+1], "%d", &daysBack); err != nil || n != 1 {
					fmt.Printf("エラー: --days には数値を指定してください\n")
					os.Exit(1)
				}
				if daysBack <= 0 {
					fmt.Printf("エラー: --days には1以上の数値を指定してください\n")
					os.Exit(1)
				}
				i++
			}
		case "--out-dir":
			if i+1 < len(os.Args) {
				outDir = os.Args[i+1]
				i++
			}
		case "--dry-run":
			dryRun = true
		case "--debug-dump":
			debugDump = true
		case "--log-level":
			if i+1 < len(os.Args) {
				logLevel = os.Args[i+1]
				i++
			}
		}
	}

	// 期間指定の排他チェック
	specifiedCount := 0
	if isDaily {
		specifiedCount++
	}
	if isWeekly {
		specifiedCount++
	}
	if daysBack > 0 {
		specifiedCount++
	}

	if specifiedCount > 1 {
		fmt.Println("エラー: --daily, --weekly, --days は同時に指定できません")
		os.Exit(1)
	}

	// デフォルト: 昨日一日分（--days 1 と同等）
	if specifiedCount == 0 {
		daysBack = 1
	}

	// BriefTypeの決定
	briefType := model.BriefTypeDaily
	if isWeekly {
		briefType = model.BriefTypeWeekly
	} else if daysBack >= 7 {
		// 7日以上の場合はWeekly形式で出力
		briefType = model.BriefTypeWeekly
	}

	// 設定ファイル読み込み
	configPath := getConfigPath()
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("❌ エラー: 設定ファイルの読み込みに失敗: %v\n", err)
		os.Exit(1)
	}

	// コマンドラインオプションで設定を上書き
	if outDir != "" {
		cfg.App.OutputDir = outDir
	}
	if logLevel != "" {
		cfg.App.LogLevel = logLevel
	}

	// ロガー初期化
	level, err := logger.ParseLevel(cfg.App.LogLevel)
	if err != nil {
		fmt.Printf("❌ エラー: 無効なログレベル: %v\n", err)
		os.Exit(1)
	}
	log := logger.New(level, false) // JSON形式はデフォルトOFF

	// 期間の計算（設定されたタイムゾーンを使用）
	loc := cfg.App.Location
	var since, until time.Time
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	if isDaily {
		// --daily: 過去24時間（従来の動作）
		since = now.Add(-time.Duration(cfg.Brief.DailyWindowHours) * time.Hour)
		until = now
	} else if isWeekly {
		// --weekly: 過去7日間（従来の動作）
		since = now.Add(-time.Duration(cfg.Brief.WeeklyDays*24) * time.Hour)
		until = now
	} else {
		// --days N または デフォルト: 過去N日分（日付単位）
		// 例: --days 1 → 昨日の0:00〜23:59:59
		// 例: --days 3 → 3日前の0:00〜昨日の23:59:59
		since = today.AddDate(0, 0, -daysBack)
		until = today.Add(-time.Nanosecond) // 昨日23:59:59.999...
	}

	log.Info("VoiceBrief starting", map[string]interface{}{
		"version":    version,
		"brief_type": string(briefType),
		"days_back":  daysBack,
		"since":      since.Format("2006-01-02 15:04:05"),
		"until":      until.Format("2006-01-02 15:04:05"),
		"dry_run":    dryRun,
	})

	// タイムアウト設定
	timeout := time.Duration(cfg.Runtime.APITimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// ブリーフィング生成
	fmt.Printf("🎙️  %s Briefing を生成中...\n\n", strings.Title(string(briefType)))

	// 期間の表示
	periodStr := fmt.Sprintf("%s 〜 %s", since.Format("2006-01-02"), until.Format("2006-01-02"))
	if daysBack == 1 && !isDaily && !isWeekly {
		periodStr = since.Format("2006-01-02") + "（昨日）"
	}

	// 1. データ取得
	fmt.Printf("📥 Step 1/4: データ取得中... [%s]\n", periodStr)
	log.Debug("Starting data fetch", map[string]interface{}{
		"brief_type": string(briefType),
		"since":      since.Format("2006-01-02 15:04:05"),
		"until":      until.Format("2006-01-02 15:04:05"),
		"timeout":    timeout.String(),
	})
	events, err := fetchEvents(ctx, cfg, since, until)
	if err != nil {
		log.Error("Failed to fetch events", map[string]interface{}{
			"error": err.Error(),
		})
		fmt.Printf("❌ エラー: データ取得に失敗: %v\n", err)
		os.Exit(1)
	}
	log.Info("Events fetched successfully", map[string]interface{}{
		"count": len(events),
	})
	fmt.Printf("✓ 合計 %d 件のイベントを取得\n\n", len(events))

	// デバッグダンプ（オプション）
	if debugDump {
		if err := dumpEventsToFile(cfg.App.OutputDir, events, briefType); err != nil {
			fmt.Printf("⚠️  警告: デバッグダンプに失敗: %v\n", err)
		}
	}

	// 2. 要約生成
	fmt.Println("📝 Step 2/4: ブリーフィング要約生成中...")
	log.Debug("Generating brief summary")
	brief, err := generateBrief(cfg, events, briefType)
	if err != nil {
		log.Error("Failed to generate brief", map[string]interface{}{
			"error": err.Error(),
		})
		fmt.Printf("❌ エラー: 要約生成に失敗: %v\n", err)
		os.Exit(1)
	}
	log.Info("Brief generated successfully", map[string]interface{}{
		"event_count": brief.GetEventCount(),
	})
	fmt.Printf("✓ 要約生成完了 (対象: %d 件)\n\n", brief.GetEventCount())

	// 3. ファイル出力（Markdown + Script + Events）
	fmt.Println("💾 Step 3/4: ファイル保存中...")
	markdownPath, scriptPath, err := saveMarkdown(cfg.App.OutputDir, brief)
	if err != nil {
		fmt.Printf("❌ エラー: ファイル保存に失敗: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Markdownを保存: %s\n", markdownPath)
	fmt.Printf("✓ スクリプトを保存: %s\n", scriptPath)

	// イベント詳細を保存
	eventsPath, err := saveEventsDetail(cfg.App.OutputDir, events, brief)
	if err != nil {
		fmt.Printf("⚠️  警告: イベント詳細の保存に失敗: %v\n", err)
	} else {
		fmt.Printf("✓ イベント詳細を保存: %s\n", eventsPath)
	}
	fmt.Println()

	// 4. 音声生成
	if dryRun {
		log.Info("Skipping audio generation (dry-run mode)")
		fmt.Println("🔇 Step 4/4: 音声生成スキップ (--dry-run)")
		fmt.Printf("\n✅ %s Briefing生成完了！\n", strings.Title(string(briefType)))
		fmt.Printf("📄 Markdown: %s\n", markdownPath)
		fmt.Printf("📝 Script: %s\n", scriptPath)
		return
	}

	fmt.Println("🎤 Step 4/4: 音声ファイル生成中...")
	log.Debug("Generating audio file")
	audioPath, err := generateAudio(ctx, cfg, brief, briefType)
	if err != nil {
		log.Error("Failed to generate audio", map[string]interface{}{
			"error": err.Error(),
		})
		fmt.Printf("❌ エラー: 音声生成に失敗: %v\n", err)
		os.Exit(1)
	}
	log.Info("Audio generated successfully", map[string]interface{}{
		"path": audioPath,
	})
	fmt.Printf("✓ 音声を生成: %s\n\n", audioPath)

	// 5. Slack投稿（オプション）
	if cfg.Slack.PostEnabled {
		fmt.Println("\n📤 Step 5/5: Slackに投稿中...")
		log.Debug("Uploading to Slack")

		// Briefに音声パスを設定
		brief.AudioPath = audioPath

		slackUploader := uploader.NewSlackUploader(&cfg.Slack, cfg.Slack.UploadAudio)
		if err := slackUploader.Upload(ctx, brief); err != nil {
			log.Warn("Slack投稿に失敗しました（処理は続行）", map[string]interface{}{
				"error": err.Error(),
			})
			fmt.Printf("⚠️  警告: Slack投稿に失敗: %v\n", err)
		} else {
			log.Info("Successfully posted to Slack")
			fmt.Printf("✓ Slackに投稿完了: %s\n", cfg.Slack.PostChannel)
		}
	}

	// 完了メッセージ
	log.Info("Brief generation completed", map[string]interface{}{
		"markdown": markdownPath,
		"audio":    audioPath,
	})
	fmt.Printf("\n✅ %s Briefing生成完了！\n", strings.Title(string(briefType)))
	fmt.Printf("📄 Markdown: %s\n", markdownPath)
	fmt.Printf("📝 Script: %s\n", scriptPath)
	fmt.Printf("🔊 Audio: %s\n", audioPath)
	if cfg.Slack.PostEnabled {
		fmt.Printf("📤 Slack: #%s\n", cfg.Slack.PostChannel)
	}
}

// fetchEvents はイベントを取得します
func fetchEvents(ctx context.Context, cfg *config.Config, since, until time.Time) (model.Events, error) {
	multiFetcher := fetcher.NewMultiFetcher(cfg)

	events, err := multiFetcher.Fetch(ctx, since)
	if err != nil {
		return nil, err
	}

	// untilでフィルタリング（指定期間内のイベントのみ）
	if !until.IsZero() {
		var filtered model.Events
		for _, event := range events {
			if event.Timestamp.Before(until) || event.Timestamp.Equal(until) {
				filtered = append(filtered, event)
			}
		}
		events = filtered
	}

	return events, nil
}

// generateBrief はブリーフィングを生成します
func generateBrief(cfg *config.Config, events model.Events, briefType model.BriefType) (*model.Brief, error) {
	// Provider に応じて Summarizer を選択
	switch cfg.Summarizer.Provider {
	case "gemini":
		// Gemini API Key を環境変数から取得
		apiKey := summarizer.GetAPIKey(cfg.Summarizer.GeminiAPIKey)
		if apiKey == "" {
			return nil, fmt.Errorf("Gemini API Key が設定されていません（環境変数: %s）", cfg.Summarizer.GeminiAPIKey)
		}

		sum, err := summarizer.NewGeminiSummarizer(
			apiKey,
			cfg.Summarizer.GeminiModel,
			cfg.Brief.MaxItemsDaily,
			cfg.Brief.MaxItemsWeekly,
		)
		if err != nil {
			return nil, fmt.Errorf("Gemini Summarizer の初期化に失敗: %w", err)
		}
		defer sum.Close()

		if briefType == model.BriefTypeDaily {
			return sum.GenerateDaily(events)
		}
		return sum.GenerateWeekly(events)

	case "openai":
		// OpenAI API Key を環境変数から取得
		apiKey := summarizer.GetAPIKey(cfg.Summarizer.OpenAIAPIKey)
		if apiKey == "" {
			return nil, fmt.Errorf("OpenAI API Key が設定されていません（環境変数: %s）", cfg.Summarizer.OpenAIAPIKey)
		}

		sum, err := summarizer.NewOpenAISummarizer(
			apiKey,
			cfg.Summarizer.OpenAIModel,
			cfg.Brief.MaxItemsDaily,
			cfg.Brief.MaxItemsWeekly,
		)
		if err != nil {
			return nil, fmt.Errorf("OpenAI Summarizer の初期化に失敗: %w", err)
		}

		var brief *model.Brief
		if briefType == model.BriefTypeDaily {
			brief, err = sum.GenerateDaily(events)
		} else {
			brief, err = sum.GenerateWeekly(events)
		}

		if err != nil {
			return nil, err
		}

		// コスト情報を表示
		total, prompt, completion := sum.GetTokenUsage()
		cost := sum.EstimateCost()
		fmt.Printf("💰 OpenAI コスト: $%.6f (Total: %d tokens, Prompt: %d, Completion: %d)\n",
			cost, total, prompt, completion)

		return brief, nil

	case "rule":
		fallthrough
	default:
		// ルールベース要約
		sum := summarizer.NewRuleSummarizer(cfg.Brief.MaxItemsDaily, cfg.Brief.MaxItemsWeekly)

		if briefType == model.BriefTypeDaily {
			return sum.GenerateDaily(events)
		}
		return sum.GenerateWeekly(events)
	}
}

// saveMarkdown はMarkdownを保存します
func saveMarkdown(outDir string, brief *model.Brief) (markdownPath, scriptPath string, err error) {
	// 出力パスを生成
	var subDir, baseName string
	if brief.Type == model.BriefTypeDaily {
		subDir = "daily"
		baseName = brief.GeneratedAt.Format("2006-01-02")
	} else {
		year, week := brief.WindowStart.ISOWeek()
		subDir = "weekly"
		baseName = fmt.Sprintf("%d-W%02d", year, week)
	}

	dirPath := filepath.Join(outDir, subDir)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", "", fmt.Errorf("ディレクトリ作成に失敗: %w", err)
	}

	// Markdownファイルを保存
	markdownPath = filepath.Join(dirPath, baseName+".md")
	if err := os.WriteFile(markdownPath, []byte(brief.ScriptMarkdown), 0644); err != nil {
		return "", "", fmt.Errorf("Markdownファイル書き込みに失敗: %w", err)
	}

	// スクリプトファイルを保存
	scriptPath = filepath.Join(dirPath, baseName+"_script.md")
	if err := os.WriteFile(scriptPath, []byte(brief.ScriptText), 0644); err != nil {
		return "", "", fmt.Errorf("スクリプトファイル書き込みに失敗: %w", err)
	}

	return markdownPath, scriptPath, nil
}

// generateAudio は音声ファイルを生成します
func generateAudio(ctx context.Context, cfg *config.Config, brief *model.Brief, briefType model.BriefType) (string, error) {
	// TTS設定
	ttsConfig := &tts.Config{
		Provider: cfg.TTS.Provider,
		Voice:    cfg.TTS.Voice,
		Rate:     cfg.TTS.Rate,
		Format:   cfg.TTS.Format,
	}

	// TTSインスタンス作成
	var ttsEngine tts.TTS
	switch cfg.TTS.Provider {
	case "say":
		ttsEngine = tts.NewSayTTS(ttsConfig)
	case "sapi":
		ttsEngine = tts.NewSAPITTS(*ttsConfig)
	case "google_tts":
		// Google Cloud認証情報を取得
		credentialsJSON := tts.GetCredentialsJSON(cfg.TTS.GoogleCredentialsJSONEnv)

		googleTTS, err := tts.NewGoogleTTS(ctx, ttsConfig, credentialsJSON)
		if err != nil {
			return "", fmt.Errorf("Google TTS の初期化に失敗: %w", err)
		}
		defer googleTTS.Close()
		ttsEngine = googleTTS
	case "openai_tts":
		// OpenAI API Key を環境変数から取得
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return "", fmt.Errorf("環境変数 OPENAI_API_KEY が設定されていません")
		}

		openaiTTS := tts.NewOpenAITTS(ttsConfig, apiKey)
		ttsEngine = openaiTTS
	default:
		return "", fmt.Errorf("未対応のTTSプロバイダー: %s", cfg.TTS.Provider)
	}

	// 出力パスを生成
	var subDir, filename string
	if briefType == model.BriefTypeDaily {
		subDir = "daily"
		filename = brief.GeneratedAt.Format("2006-01-02") + "." + cfg.TTS.Format
	} else {
		year, week := brief.WindowStart.ISOWeek()
		subDir = "weekly"
		filename = fmt.Sprintf("%d-W%02d.%s", year, week, cfg.TTS.Format)
	}

	dirPath := filepath.Join(cfg.App.OutputDir, subDir)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", fmt.Errorf("ディレクトリ作成に失敗: %w", err)
	}

	audioPath := filepath.Join(dirPath, filename)

	// 音声生成
	if err := ttsEngine.Generate(ctx, brief.ScriptText, audioPath); err != nil {
		return "", err
	}

	return audioPath, nil
}

// dumpEventsToFile はイベントをデバッグ用にJSONダンプします
func dumpEventsToFile(outDir string, events model.Events, briefType model.BriefType) error {
	debugDir := filepath.Join(outDir, "debug")
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		return err
	}

	filename := fmt.Sprintf("%s_%s.json", string(briefType), time.Now().Format("2006-01-02_15-04-05"))
	filePath := filepath.Join(debugDir, filename)

	// 簡易的にイベント情報を出力
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("{\n  \"count\": %d,\n  \"events\": [\n", len(events)))
	for i, event := range events {
		sb.WriteString(fmt.Sprintf("    {\n"))
		sb.WriteString(fmt.Sprintf("      \"id\": \"%s\",\n", event.ID))
		sb.WriteString(fmt.Sprintf("      \"source\": \"%s\",\n", event.Source))
		sb.WriteString(fmt.Sprintf("      \"category\": \"%s\",\n", event.Category))
		sb.WriteString(fmt.Sprintf("      \"title\": \"%s\",\n", strings.ReplaceAll(event.Title, "\"", "\\\"")))
		sb.WriteString(fmt.Sprintf("      \"importance\": %d\n", event.Importance))
		if i < len(events)-1 {
			sb.WriteString("    },\n")
		} else {
			sb.WriteString("    }\n")
		}
	}
	sb.WriteString("  ]\n}\n")

	if err := os.WriteFile(filePath, []byte(sb.String()), 0644); err != nil {
		return err
	}

	fmt.Printf("✓ デバッグダンプを保存: %s\n\n", filePath)
	return nil
}

// saveEventsDetail はイベントの詳細をMarkdown形式で保存します
func saveEventsDetail(outDir string, events model.Events, brief *model.Brief) (string, error) {
	// 出力パスを生成
	var subDir, baseName string
	if brief.Type == model.BriefTypeDaily {
		subDir = "daily"
		baseName = brief.GeneratedAt.Format("2006-01-02")
	} else {
		year, week := brief.WindowStart.ISOWeek()
		subDir = "weekly"
		baseName = fmt.Sprintf("%d-W%02d", year, week)
	}

	dirPath := filepath.Join(outDir, subDir)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", fmt.Errorf("ディレクトリ作成に失敗: %w", err)
	}

	filePath := filepath.Join(dirPath, baseName+"_events.md")

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# 取得イベント詳細\n\n"))
	sb.WriteString(fmt.Sprintf("- **期間**: %s 〜 %s\n",
		brief.WindowStart.Format("2006-01-02 15:04"),
		brief.WindowEnd.Format("2006-01-02 15:04")))
	sb.WriteString(fmt.Sprintf("- **取得件数**: %d件\n\n", len(events)))
	sb.WriteString("---\n\n")

	// ソース別にグループ化
	sourceEvents := make(map[string]model.Events)
	for _, event := range events {
		sourceEvents[event.Source] = append(sourceEvents[event.Source], event)
	}

	for source, evts := range sourceEvents {
		sb.WriteString(fmt.Sprintf("## %s (%d件)\n\n", source, len(evts)))

		for i, event := range evts {
			sb.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, event.Title))
			sb.WriteString(fmt.Sprintf("- **ID**: `%s`\n", event.ID))
			sb.WriteString(fmt.Sprintf("- **ソース**: %s\n", event.Source))
			sb.WriteString(fmt.Sprintf("- **場所**: %s\n", event.Location))
			sb.WriteString(fmt.Sprintf("- **カテゴリ**: %s\n", event.Category))
			sb.WriteString(fmt.Sprintf("- **重要度**: %d/100\n", event.Importance))
			sb.WriteString(fmt.Sprintf("- **時刻**: %s\n", event.Timestamp.Format("2006-01-02 15:04:05")))
			sb.WriteString(fmt.Sprintf("- **投稿者**: %s\n", event.Author))
			if event.URL != "" {
				sb.WriteString(fmt.Sprintf("- **URL**: %s\n", event.URL))
			}

			// Notion固有の詳細情報
			if event.Source == "notion" {
				if changeType, ok := event.Refs["change_type"]; ok {
					sb.WriteString(fmt.Sprintf("- **変更種別**: %s\n", changeType))
				}
				if createdAt, ok := event.Refs["created_at"]; ok {
					sb.WriteString(fmt.Sprintf("- **作成日時**: %s\n", createdAt))
				}
				if updatedAt, ok := event.Refs["updated_at"]; ok {
					sb.WriteString(fmt.Sprintf("- **更新日時**: %s\n", updatedAt))
				}
				if properties, ok := event.Refs["properties"]; ok && properties != "" {
					sb.WriteString(fmt.Sprintf("- **プロパティ**: %s\n", properties))
				}
			}
			sb.WriteString("\n")

			if event.Body != "" {
				sb.WriteString("**本文:**\n\n")
				sb.WriteString("```\n")
				sb.WriteString(event.Body)
				sb.WriteString("\n```\n\n")
			}

			sb.WriteString("---\n\n")
		}
	}

	if err := os.WriteFile(filePath, []byte(sb.String()), 0644); err != nil {
		return "", fmt.Errorf("イベント詳細ファイル書き込みに失敗: %w", err)
	}

	return filePath, nil
}

func handleConfigCommand(subcommand string) {
	if subcommand != "check" {
		fmt.Printf("エラー: 不明なサブコマンド 'config %s'\n", subcommand)
		os.Exit(1)
	}

	// 設定ファイルパスを取得（デフォルト: ./config.yaml）
	configPath := getConfigPath()

	fmt.Printf("設定ファイルを確認中: %s\n", configPath)

	// 設定ファイルの存在確認
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("❌ エラー: 設定ファイルが見つかりません: %s\n", configPath)
		fmt.Println("\n次のコマンドで設定ファイルを作成してください:")
		fmt.Println("  cp config.example.yaml config.yaml")
		os.Exit(1)
	}

	// 設定ファイルを読み込み
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("❌ エラー: 設定ファイルの読み込みに失敗: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ 設定ファイルの読み込みに成功")

	// 基本設定の確認
	fmt.Printf("✓ 出力ディレクトリ: %s\n", cfg.App.OutputDir)
	fmt.Printf("✓ ログレベル: %s\n", cfg.App.LogLevel)
	fmt.Printf("✓ タイムゾーン: %s\n", cfg.App.Location.String())

	// Slack設定の確認
	fmt.Printf("✓ Slack Token: 環境変数 %s から読み込み済み\n", cfg.Slack.TokenEnv)
	fmt.Printf("✓ Slack監視チャンネル数: %d\n", len(cfg.Slack.Channels))
	for i, ch := range cfg.Slack.Channels {
		fmt.Printf("  %d. %s (%s)\n", i+1, ch.Name, ch.ID)
	}

	// Notion設定の確認
	fmt.Printf("✓ Notion Token: 環境変数 %s から読み込み済み\n", cfg.Notion.TokenEnv)
	fmt.Printf("✓ Notion監視データベース数: %d\n", len(cfg.Notion.Databases))
	for i, db := range cfg.Notion.Databases {
		fmt.Printf("  %d. %s (%s)\n", i+1, db.Name, db.ID)
	}

	// GitHub設定の確認（オプション）
	if cfg.GitHub.Enabled {
		fmt.Printf("✓ GitHub連携: 有効 (ユーザー: %s)\n", cfg.GitHub.Username)
	} else {
		fmt.Println("✓ GitHub連携: 無効")
	}

	// Brief設定の確認
	fmt.Printf("✓ Daily期間: %d時間\n", cfg.Brief.DailyWindowHours)
	fmt.Printf("✓ Weekly期間: %d日\n", cfg.Brief.WeeklyDays)

	// Summarizer/TTS設定の確認
	fmt.Printf("✓ 要約エンジン: %s\n", cfg.Summarizer.Provider)
	fmt.Printf("✓ 音声合成: %s (音声: %s, 速度: %.1fx)\n", cfg.TTS.Provider, cfg.TTS.Voice, cfg.TTS.Rate)

	// トークンの形式検証
	fmt.Println("\nトークンの形式を検証中...")
	if err := cfg.ValidateTokens(); err != nil {
		fmt.Printf("❌ エラー: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ すべてのトークンが正しい形式です")

	// 出力ディレクトリの確認
	fmt.Println("\n出力ディレクトリを確認中...")
	if _, err := os.Stat(cfg.App.OutputDir); os.IsNotExist(err) {
		fmt.Printf("⚠  警告: 出力ディレクトリが存在しません: %s\n", cfg.App.OutputDir)
		fmt.Println("   初回実行時に自動作成されます")
	} else {
		fmt.Printf("✓ 出力ディレクトリが存在します: %s\n", cfg.App.OutputDir)
	}

	fmt.Println("\n✅ すべての設定チェックが完了しました！")
}

func handleDoctorCommand() {
	// --source フラグを確認
	sourceFilter := ""
	for i, arg := range os.Args {
		if arg == "--source" && i+1 < len(os.Args) {
			sourceFilter = os.Args[i+1]
			break
		}
	}

	configPath := getConfigPath()

	fmt.Println("API接続を確認中...")
	fmt.Printf("設定ファイル: %s\n\n", configPath)

	// 設定ファイルを読み込み
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("❌ エラー: 設定ファイルの読み込みに失敗: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	hasError := false

	// 全ソース並列テスト（sourceFilterが指定されていない場合）
	if sourceFilter == "" {
		fmt.Println("🚀 全ソース並列接続テスト中...")
		multiFetcher := fetcher.NewMultiFetcher(cfg)

		if err := multiFetcher.TestAllConnections(ctx); err != nil {
			fmt.Printf("❌ 並列接続テストに失敗: %v\n", err)
			hasError = true
		} else {
			fmt.Println("✓ 全ソース並列接続テスト成功")
			fmt.Println()

			// 並列取得のデモ
			fmt.Println("📥 過去1時間のデータを並列取得中...")
			since := time.Now().Add(-1 * time.Hour)
			events, err := multiFetcher.Fetch(ctx, since)
			if err != nil {
				fmt.Printf("⚠️  警告: 並列取得に失敗: %v\n", err)
			} else {
				fmt.Printf("\n✅ 並列取得完了: 合計 %d 件\n", len(events))
				if len(events) > 0 {
					fmt.Println("\nイベントサンプル（最大5件、重要度順）:")
					// 重要度でソート
					sort.Sort(events)
					for i, event := range events {
						if i >= 5 {
							break
						}
						fmt.Printf("  %d. [%s/%s] %s (重要度: %d)\n",
							i+1, event.Source, event.Location, event.Title, event.Importance)
					}
				}
			}
		}
		fmt.Println()
		fmt.Println("========================================")
		fmt.Println()
	}

	// Slack接続確認（個別テスト）
	if sourceFilter == "" || sourceFilter == "slack" {
		fmt.Println("📡 Slack API 接続確認中...")
		slackFetcher := fetcher.NewSlackFetcher(&cfg.Slack)

		if err := slackFetcher.TestConnection(ctx); err != nil {
			fmt.Printf("❌ Slack API接続に失敗: %v\n", err)
			hasError = true
		} else {
			fmt.Println("✓ Slack API接続成功")

			// 各チャンネルからテスト取得
			fmt.Printf("✓ 監視対象チャンネル数: %d\n", len(cfg.Slack.Channels))
			for i, ch := range cfg.Slack.Channels {
				fmt.Printf("  %d. %s (%s)\n", i+1, ch.Name, ch.ID)
			}

			// 簡易的に過去1時間のメッセージを取得してみる
			fmt.Println("\n過去1時間のメッセージを取得中...")
			since := time.Now().Add(-1 * time.Hour)
			events, err := slackFetcher.Fetch(ctx, since)
			if err != nil {
				fmt.Printf("⚠️  警告: メッセージ取得に失敗: %v\n", err)
			} else {
				fmt.Printf("✓ %d件のメッセージを取得しました\n", len(events))
				if len(events) > 0 {
					fmt.Println("\n最新メッセージのサンプル（最大3件）:")
					for i, event := range events {
						if i >= 3 {
							break
						}
						fmt.Printf("  - [%s] %s: %s\n", event.Location, event.Author, event.Title)
					}
				}
			}
		}
		fmt.Println()
	}

	// Notion接続確認
	if sourceFilter == "" || sourceFilter == "notion" {
		fmt.Println("📡 Notion API 接続確認中...")
		notionFetcher := fetcher.NewNotionFetcher(&cfg.Notion)

		if err := notionFetcher.TestConnection(ctx); err != nil {
			fmt.Printf("❌ Notion API接続に失敗: %v\n", err)
			hasError = true
		} else {
			fmt.Println("✓ Notion API接続成功")

			// 各データベースからテスト取得
			fmt.Printf("✓ 監視対象データベース数: %d\n", len(cfg.Notion.Databases))
			for i, db := range cfg.Notion.Databases {
				fmt.Printf("  %d. %s (%s)\n", i+1, db.Name, db.ID)
			}

			// 簡易的に過去1日のページを取得してみる
			fmt.Println("\n過去1日の更新ページを取得中...")
			since := time.Now().Add(-24 * time.Hour)
			events, err := notionFetcher.Fetch(ctx, since)
			if err != nil {
				fmt.Printf("⚠️  警告: ページ取得に失敗: %v\n", err)
			} else {
				fmt.Printf("✓ %d件のページを取得しました\n", len(events))
				if len(events) > 0 {
					fmt.Println("\n最新ページのサンプル（最大3件）:")
					for i, event := range events {
						if i >= 3 {
							break
						}
						fmt.Printf("  - [%s] %s: %s\n", event.Location, event.Author, event.Title)
					}
				}
			}
		}
		fmt.Println()
	}

	// GitHub接続確認（オプション）
	if cfg.GitHub.Enabled && (sourceFilter == "" || sourceFilter == "github") {
		fmt.Println("📡 GitHub API 接続確認中...")
		fmt.Println("⚠️  TODO: GitHub接続確認は Phase 4+ で実装予定")
		fmt.Println()
	}

	if hasError {
		fmt.Println("❌ 一部のAPI接続に失敗しました")
		os.Exit(1)
	} else {
		fmt.Println("✅ すべてのAPI接続が成功しました！")
	}
}

func getConfigPath() string {
	// --config フラグを確認
	for i, arg := range os.Args {
		if arg == "--config" && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
	}
	return "./config.yaml"
}

func printUsage() {
	fmt.Println("VoiceBrief", version)
	fmt.Println("音声で聞く、チームの最新情報")
	fmt.Println("\nUsage:")
	fmt.Println("  voicebrief run                  Briefingを生成（デフォルト: 昨日一日分）")
	fmt.Println("  voicebrief run --days N         過去N日分のBriefingを生成")
	fmt.Println("  voicebrief run --daily          Daily Briefing（過去24時間）")
	fmt.Println("  voicebrief run --weekly         Weekly Briefing（過去7日間）")
	fmt.Println("  voicebrief config check         設定ファイルを検証")
	fmt.Println("  voicebrief doctor               API接続を確認")
	fmt.Println("  voicebrief version              バージョン情報を表示")
	fmt.Println("\nOptions:")
	fmt.Println("  --days N                        過去N日分のデータを取得 (default: 1)")
	fmt.Println("  --config PATH                   設定ファイルパス (default: ./config.yaml)")
	fmt.Println("  --out-dir PATH                  出力ディレクトリ (default: ./out)")
	fmt.Println("  --dry-run                       音声生成・投稿をスキップ")
	fmt.Println("  --debug-dump                    生データをout/debug/に保存")
	fmt.Println("  --log-level LEVEL               ログレベル (debug|info|warn|error)")
	fmt.Println("\nExamples:")
	fmt.Println("  voicebrief run                  # 昨日一日分を取得")
	fmt.Println("  voicebrief run --days 3         # 過去3日分を取得")
	fmt.Println("  voicebrief run --dry-run        # 音声なしで原稿のみ生成")
}
