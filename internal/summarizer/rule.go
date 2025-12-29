package summarizer

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/masumomo/voice-brief/internal/model"
)

// RuleSummarizer はルールベースの要約エンジン
type RuleSummarizer struct {
	maxItemsDaily  int
	maxItemsWeekly int
}

// NewRuleSummarizer は新しいRuleSummarizerを作成します
func NewRuleSummarizer(maxDaily, maxWeekly int) *RuleSummarizer {
	if maxDaily <= 0 {
		maxDaily = 8
	}
	if maxWeekly <= 0 {
		maxWeekly = 25
	}
	return &RuleSummarizer{
		maxItemsDaily:  maxDaily,
		maxItemsWeekly: maxWeekly,
	}
}

// GenerateDaily はDaily Briefingを生成します
func (s *RuleSummarizer) GenerateDaily(events model.Events) (*model.Brief, error) {
	now := time.Now()
	start := now.Add(-24 * time.Hour)

	brief := model.NewBrief(model.BriefTypeDaily, start, now)

	// イベントを重要度でソート
	sort.Sort(events)

	// 最大件数まで取得
	topEvents := events
	if len(topEvents) > s.maxItemsDaily {
		topEvents = topEvents[:s.maxItemsDaily]
	}
	brief.AddEvents(topEvents)

	// Markdown生成
	brief.ScriptMarkdown = s.generateDailyMarkdown(brief)

	// TTS用テキスト生成
	brief.ScriptText = s.generateDailyScript(brief)

	return brief, nil
}

// GenerateWeekly はWeekly Briefingを生成します
func (s *RuleSummarizer) GenerateWeekly(events model.Events) (*model.Brief, error) {
	now := time.Now()
	start := now.Add(-7 * 24 * time.Hour)

	brief := model.NewBrief(model.BriefTypeWeekly, start, now)

	// イベントを重要度でソート
	sort.Sort(events)

	// 最大件数まで取得
	topEvents := events
	if len(topEvents) > s.maxItemsWeekly {
		topEvents = topEvents[:s.maxItemsWeekly]
	}
	brief.AddEvents(topEvents)

	// Markdown生成
	brief.ScriptMarkdown = s.generateWeeklyMarkdown(brief)

	// TTS用テキスト生成
	brief.ScriptText = s.generateWeeklyScript(brief)

	return brief, nil
}

// generateDailyMarkdown はDaily用のMarkdownを生成します
func (s *RuleSummarizer) generateDailyMarkdown(brief *model.Brief) string {
	var sb strings.Builder

	// ヘッダー
	sb.WriteString(fmt.Sprintf("# Daily Briefing - %s\n\n", brief.GetDurationString()))
	sb.WriteString(fmt.Sprintf("生成日時: %s\n\n", brief.GeneratedAt.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("対象期間: %s 〜 %s\n\n",
		brief.WindowStart.Format("2006-01-02 15:04"),
		brief.WindowEnd.Format("2006-01-02 15:04")))
	sb.WriteString(fmt.Sprintf("イベント総数: %d件\n\n", brief.GetEventCount()))
	sb.WriteString("---\n\n")

	// セクション1: 今日の主な動き
	sb.WriteString("## 今日の主な動き\n\n")
	topEvents := brief.GetTopEvents(3)
	if len(topEvents) == 0 {
		sb.WriteString("更新はありません。\n\n")
	} else {
		for i, event := range topEvents {
			sb.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, event.Title))
			sb.WriteString(fmt.Sprintf("- **ソース**: %s / %s\n", event.Source, event.Location))
			sb.WriteString(fmt.Sprintf("- **発生時刻**: %s\n", event.Timestamp.Format("15:04")))
			sb.WriteString(fmt.Sprintf("- **重要度**: %d\n", event.Importance))
			if event.Author != "" && event.Author != "Unknown" {
				sb.WriteString(fmt.Sprintf("- **投稿者**: %s\n", event.Author))
			}
			if event.Body != "" {
				sb.WriteString(fmt.Sprintf("- **内容**: %s\n", event.Body))
			}
			if event.URL != "" {
				sb.WriteString(fmt.Sprintf("- **リンク**: %s\n", event.URL))
			}
			sb.WriteString("\n")
		}
	}

	// セクション2: カテゴリ別サマリー
	sb.WriteString("## カテゴリ別サマリー\n\n")
	s.writeCategorySummary(&sb, brief)

	// セクション3: ソース別サマリー
	sb.WriteString("## ソース別サマリー\n\n")
	s.writeSourceSummary(&sb, brief)

	return sb.String()
}

// generateWeeklyMarkdown はWeekly用のMarkdownを生成します
func (s *RuleSummarizer) generateWeeklyMarkdown(brief *model.Brief) string {
	var sb strings.Builder

	// ヘッダー
	sb.WriteString(fmt.Sprintf("# Weekly Briefing - %s\n\n", brief.GetDurationString()))
	sb.WriteString(fmt.Sprintf("生成日時: %s\n\n", brief.GeneratedAt.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("イベント総数: %d件\n\n", brief.GetEventCount()))
	sb.WriteString("---\n\n")

	// セクション1: 今週の流れ
	sb.WriteString("## 今週の流れ\n\n")
	sb.WriteString(fmt.Sprintf("今週は%d件のイベントがありました。", brief.GetEventCount()))
	sb.WriteString("主要な動きをカテゴリ別に振り返ります。\n\n")

	// セクション2: 重要な出来事（トップ5）
	sb.WriteString("## 重要な出来事 (Top 5)\n\n")
	topEvents := brief.GetTopEvents(5)
	for i, event := range topEvents {
		sb.WriteString(fmt.Sprintf("%d. **%s** (%s) - 重要度: %d\n",
			i+1, event.Title, event.Location, event.Importance))
	}
	sb.WriteString("\n")

	// セクション3: カテゴリ別サマリー
	sb.WriteString("## カテゴリ別サマリー\n\n")
	s.writeCategorySummary(&sb, brief)

	// セクション4: ソース別サマリー
	sb.WriteString("## ソース別サマリー\n\n")
	s.writeSourceSummary(&sb, brief)

	return sb.String()
}

// generateDailyScript はDaily用のTTSスクリプトを生成します
func (s *RuleSummarizer) generateDailyScript(brief *model.Brief) string {
	var sb strings.Builder

	// イントロ
	sb.WriteString(fmt.Sprintf("%sのデイリーブリーフィングです。\n", brief.GetDurationString()))

	eventCount := brief.GetEventCount()
	if eventCount == 0 {
		sb.WriteString("本日は更新がありませんでした。\n")
		sb.WriteString("以上です。\n")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("合計%d件の更新がありました。\n\n", eventCount))

	// 主な動き
	sb.WriteString("今日の主な動きです。\n")
	topEvents := brief.GetTopEvents(3)
	for i, event := range topEvents {
		sb.WriteString(fmt.Sprintf("%d件目。", i+1))
		sb.WriteString(fmt.Sprintf("%sより、%s。", event.Location, event.Title))
		if event.Body != "" && len(event.Body) < 100 {
			sb.WriteString(fmt.Sprintf("内容は、%s。", event.Body))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	// カテゴリ別概要
	incidentEvents := brief.GetEventsByCategory(model.EventCategoryIncident)
	if len(incidentEvents) > 0 {
		sb.WriteString(fmt.Sprintf("障害やインシデント関連が%d件あります。確認が必要です。\n", len(incidentEvents)))
	}

	devEvents := brief.GetEventsByCategory(model.EventCategoryDev)
	if len(devEvents) > 0 {
		sb.WriteString(fmt.Sprintf("開発関連が%d件あります。\n", len(devEvents)))
	}

	// エンディング
	sb.WriteString("\n以上、デイリーブリーフィングでした。\n")

	return sb.String()
}

// generateWeeklyScript はWeekly用のTTSスクリプトを生成します
func (s *RuleSummarizer) generateWeeklyScript(brief *model.Brief) string {
	var sb strings.Builder

	// イントロ
	sb.WriteString(fmt.Sprintf("%sのウィークリーブリーフィングです。\n", brief.GetDurationString()))

	eventCount := brief.GetEventCount()
	if eventCount == 0 {
		sb.WriteString("今週は更新がありませんでした。\n")
		sb.WriteString("以上です。\n")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("今週は合計%d件の更新がありました。\n\n", eventCount))

	// 重要な出来事
	sb.WriteString("今週の重要な出来事です。\n")
	topEvents := brief.GetTopEvents(5)
	for i, event := range topEvents {
		sb.WriteString(fmt.Sprintf("%d件目。%s。", i+1, event.Title))
		sb.WriteString(fmt.Sprintf("%sからの更新です。\n", event.Location))
	}
	sb.WriteString("\n")

	// カテゴリ別サマリー
	s.writeCategoryScriptSummary(&sb, brief)

	// エンディング
	sb.WriteString("\n以上、ウィークリーブリーフィングでした。\n")

	return sb.String()
}

// writeCategorySummary はカテゴリ別サマリーをMarkdownに書き込みます
func (s *RuleSummarizer) writeCategorySummary(sb *strings.Builder, brief *model.Brief) {
	categories := []string{
		model.EventCategoryIncident,
		model.EventCategoryDev,
		model.EventCategoryBiz,
		model.EventCategoryOps,
		model.EventCategoryOther,
	}

	categoryNames := map[string]string{
		model.EventCategoryIncident: "障害・インシデント",
		model.EventCategoryDev:      "開発",
		model.EventCategoryBiz:      "ビジネス",
		model.EventCategoryOps:      "運用",
		model.EventCategoryOther:    "その他",
	}

	for _, cat := range categories {
		events := brief.GetEventsByCategory(cat)
		if len(events) > 0 {
			sb.WriteString(fmt.Sprintf("- **%s**: %d件\n", categoryNames[cat], len(events)))
		}
	}
	sb.WriteString("\n")
}

// writeSourceSummary はソース別サマリーをMarkdownに書き込みます
func (s *RuleSummarizer) writeSourceSummary(sb *strings.Builder, brief *model.Brief) {
	slackEvents := brief.GetEventsBySource(model.EventSourceSlack)
	notionEvents := brief.GetEventsBySource(model.EventSourceNotion)
	githubEvents := brief.GetEventsBySource(model.EventSourceGitHub)

	if len(slackEvents) > 0 {
		sb.WriteString(fmt.Sprintf("- **Slack**: %d件\n", len(slackEvents)))
	}
	if len(notionEvents) > 0 {
		sb.WriteString(fmt.Sprintf("- **Notion**: %d件\n", len(notionEvents)))
	}
	if len(githubEvents) > 0 {
		sb.WriteString(fmt.Sprintf("- **GitHub**: %d件\n", len(githubEvents)))
	}
	sb.WriteString("\n")
}

// writeCategoryScriptSummary はカテゴリ別サマリーをスクリプトに書き込みます
func (s *RuleSummarizer) writeCategoryScriptSummary(sb *strings.Builder, brief *model.Brief) {
	incidentEvents := brief.GetEventsByCategory(model.EventCategoryIncident)
	if len(incidentEvents) > 0 {
		sb.WriteString(fmt.Sprintf("障害やインシデント関連が%d件ありました。\n", len(incidentEvents)))
	}

	devEvents := brief.GetEventsByCategory(model.EventCategoryDev)
	if len(devEvents) > 0 {
		sb.WriteString(fmt.Sprintf("開発関連が%d件ありました。\n", len(devEvents)))
	}

	bizEvents := brief.GetEventsByCategory(model.EventCategoryBiz)
	if len(bizEvents) > 0 {
		sb.WriteString(fmt.Sprintf("ビジネス関連が%d件ありました。\n", len(bizEvents)))
	}
}
