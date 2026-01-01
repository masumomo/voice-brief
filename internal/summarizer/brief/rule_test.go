package brief

import (
	"testing"
	"time"

	"github.com/masumomo/voice-brief/internal/model"
)

func TestGenerateDaily(t *testing.T) {
	summarizer := NewRuleSummarizer(8, 25)

	// テスト用イベント作成
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	since := today.Add(-24 * time.Hour)
	until := today
	events := model.Events{
		{
			ID:         "1",
			Source:     model.EventSourceSlack,
			Category:   model.EventCategoryDev,
			Timestamp:  now.Add(-1 * time.Hour),
			Title:      "PRレビュー依頼",
			Body:       "新機能のPRをレビューお願いします",
			Location:   "dev-channel",
			Author:     "user1",
			Importance: 70,
		},
		{
			ID:         "2",
			Source:     model.EventSourceNotion,
			Category:   model.EventCategoryBiz,
			Timestamp:  now.Add(-2 * time.Hour),
			Title:      "定例会議",
			Body:       "Status: Done",
			Location:   "Task Board",
			Author:     "user2",
			Importance: 50,
		},
	}

	brief, err := summarizer.GenerateDaily(events, since, until)
	if err != nil {
		t.Fatalf("GenerateDaily failed: %v", err)
	}

	if brief.Type != model.BriefTypeDaily {
		t.Errorf("BriefType = %s; want %s", brief.Type, model.BriefTypeDaily)
	}

	if len(brief.Items) != 2 {
		t.Errorf("Items count = %d; want 2", len(brief.Items))
	}

	if brief.ScriptMarkdown == "" {
		t.Error("ScriptMarkdown should not be empty")
	}

	if brief.ScriptText == "" {
		t.Error("ScriptText should not be empty")
	}

	// Markdownに必須要素が含まれているか確認
	if !contains(brief.ScriptMarkdown, "Daily Briefing") {
		t.Error("Markdown should contain 'Daily Briefing'")
	}

	// TTS用テキストに必須要素が含まれているか確認
	if !contains(brief.ScriptText, "デイリーブリーフィング") {
		t.Error("Script should contain 'デイリーブリーフィング'")
	}
}

func TestGenerateWeekly(t *testing.T) {
	summarizer := NewRuleSummarizer(8, 25)

	// テスト用イベント作成
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	since := today.Add(-7 * 24 * time.Hour)
	until := today
	events := model.Events{
		{
			ID:         "1",
			Source:     model.EventSourceSlack,
			Category:   model.EventCategoryIncident,
			Timestamp:  now.Add(-24 * time.Hour),
			Title:      "障害発生",
			Body:       "サーバーダウン",
			Location:   "ops-channel",
			Author:     "user1",
			Importance: 90,
		},
	}

	brief, err := summarizer.GenerateWeekly(events, since, until)
	if err != nil {
		t.Fatalf("GenerateWeekly failed: %v", err)
	}

	if brief.Type != model.BriefTypeWeekly {
		t.Errorf("BriefType = %s; want %s", brief.Type, model.BriefTypeWeekly)
	}

	if brief.ScriptMarkdown == "" {
		t.Error("ScriptMarkdown should not be empty")
	}

	if brief.ScriptText == "" {
		t.Error("ScriptText should not be empty")
	}

	// Markdownに必須要素が含まれているか確認
	if !contains(brief.ScriptMarkdown, "Weekly Briefing") {
		t.Error("Markdown should contain 'Weekly Briefing'")
	}

	// TTS用テキストに必須要素が含まれているか確認
	if !contains(brief.ScriptText, "ウィークリーブリーフィング") {
		t.Error("Script should contain 'ウィークリーブリーフィング'")
	}
}

func TestGenerateDaily_Empty(t *testing.T) {
	summarizer := NewRuleSummarizer(8, 25)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	since := today.Add(-24 * time.Hour)
	until := today
	events := model.Events{}

	brief, err := summarizer.GenerateDaily(events, since, until)
	if err != nil {
		t.Fatalf("GenerateDaily with empty events failed: %v", err)
	}

	if len(brief.Items) != 0 {
		t.Errorf("Items count = %d; want 0", len(brief.Items))
	}

	if !contains(brief.ScriptText, "更新がありません") {
		t.Error("Script should contain '更新がありません' for empty events")
	}
}

func TestGenerateWeekly_Empty(t *testing.T) {
	summarizer := NewRuleSummarizer(8, 25)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	since := today.Add(-7 * 24 * time.Hour)
	until := today
	events := model.Events{}

	brief, err := summarizer.GenerateWeekly(events, since, until)
	if err != nil {
		t.Fatalf("GenerateWeekly with empty events failed: %v", err)
	}

	if len(brief.Items) != 0 {
		t.Errorf("Items count = %d; want 0", len(brief.Items))
	}

	if !contains(brief.ScriptText, "更新がありません") {
		t.Error("Script should contain '更新がありません' for empty events")
	}
}

func TestNewRuleSummarizer_Defaults(t *testing.T) {
	// 負の値を渡すとデフォルト値になる
	summarizer := NewRuleSummarizer(-1, -1)

	if summarizer.maxItemsDaily != 8 {
		t.Errorf("maxItemsDaily = %d; want 8", summarizer.maxItemsDaily)
	}
	if summarizer.maxItemsWeekly != 25 {
		t.Errorf("maxItemsWeekly = %d; want 25", summarizer.maxItemsWeekly)
	}

	// ゼロも同様
	summarizer2 := NewRuleSummarizer(0, 0)
	if summarizer2.maxItemsDaily != 8 {
		t.Errorf("maxItemsDaily = %d; want 8", summarizer2.maxItemsDaily)
	}
}

func TestNewRuleSummarizer_Custom(t *testing.T) {
	summarizer := NewRuleSummarizer(5, 15)

	if summarizer.maxItemsDaily != 5 {
		t.Errorf("maxItemsDaily = %d; want 5", summarizer.maxItemsDaily)
	}
	if summarizer.maxItemsWeekly != 15 {
		t.Errorf("maxItemsWeekly = %d; want 15", summarizer.maxItemsWeekly)
	}
}

func TestGenerateDaily_MaxItems(t *testing.T) {
	summarizer := NewRuleSummarizer(3, 25) // maxDaily=3

	// 5件のイベントを作成
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	since := today.Add(-24 * time.Hour)
	until := today
	events := model.Events{
		{ID: "1", Title: "Event 1", Importance: 90, Timestamp: now},
		{ID: "2", Title: "Event 2", Importance: 80, Timestamp: now},
		{ID: "3", Title: "Event 3", Importance: 70, Timestamp: now},
		{ID: "4", Title: "Event 4", Importance: 60, Timestamp: now},
		{ID: "5", Title: "Event 5", Importance: 50, Timestamp: now},
	}

	brief, err := summarizer.GenerateDaily(events, since, until)
	if err != nil {
		t.Fatalf("GenerateDaily failed: %v", err)
	}

	// maxItems=3なので3件まで
	if len(brief.Items) != 3 {
		t.Errorf("Items count = %d; want 3 (maxItems)", len(brief.Items))
	}

	// 重要度順にソートされていることを確認
	for i := 0; i < len(brief.Items)-1; i++ {
		if brief.Items[i].Importance < brief.Items[i+1].Importance {
			t.Errorf("Items should be sorted by importance descending")
		}
	}
}

func TestGenerateWeekly_MaxItems(t *testing.T) {
	summarizer := NewRuleSummarizer(8, 2) // maxWeekly=2

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	since := today.Add(-7 * 24 * time.Hour)
	until := today
	events := model.Events{
		{ID: "1", Title: "Event 1", Importance: 90, Timestamp: now},
		{ID: "2", Title: "Event 2", Importance: 80, Timestamp: now},
		{ID: "3", Title: "Event 3", Importance: 70, Timestamp: now},
	}

	brief, err := summarizer.GenerateWeekly(events, since, until)
	if err != nil {
		t.Fatalf("GenerateWeekly failed: %v", err)
	}

	if len(brief.Items) != 2 {
		t.Errorf("Items count = %d; want 2 (maxItems)", len(brief.Items))
	}
}

func TestGenerateDaily_CategorySummary(t *testing.T) {
	summarizer := NewRuleSummarizer(10, 25)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	since := today.Add(-24 * time.Hour)
	until := today
	events := model.Events{
		{ID: "1", Title: "Incident", Category: model.EventCategoryIncident, Importance: 90, Timestamp: now},
		{ID: "2", Title: "Dev task", Category: model.EventCategoryDev, Importance: 80, Timestamp: now},
		{ID: "3", Title: "Biz update", Category: model.EventCategoryBiz, Importance: 70, Timestamp: now},
	}

	brief, err := summarizer.GenerateDaily(events, since, until)
	if err != nil {
		t.Fatalf("GenerateDaily failed: %v", err)
	}

	// カテゴリサマリーに各カテゴリが含まれていることを確認
	if !contains(brief.ScriptMarkdown, "障害・インシデント") {
		t.Error("Markdown should contain '障害・インシデント' category")
	}
	if !contains(brief.ScriptMarkdown, "開発") {
		t.Error("Markdown should contain '開発' category")
	}
	if !contains(brief.ScriptMarkdown, "ビジネス") {
		t.Error("Markdown should contain 'ビジネス' category")
	}
}

func TestGenerateDaily_SourceSummary(t *testing.T) {
	summarizer := NewRuleSummarizer(10, 25)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	since := today.Add(-24 * time.Hour)
	until := today
	events := model.Events{
		{ID: "1", Title: "Slack msg", Source: model.EventSourceSlack, Importance: 90, Timestamp: now},
		{ID: "2", Title: "Notion page", Source: model.EventSourceNotion, Importance: 80, Timestamp: now},
		{ID: "3", Title: "GitHub PR", Source: model.EventSourceGitHub, Importance: 70, Timestamp: now},
	}

	brief, err := summarizer.GenerateDaily(events, since, until)
	if err != nil {
		t.Fatalf("GenerateDaily failed: %v", err)
	}

	// ソースサマリーに各ソースが含まれていることを確認
	if !contains(brief.ScriptMarkdown, "Slack") {
		t.Error("Markdown should contain 'Slack' source")
	}
	if !contains(brief.ScriptMarkdown, "Notion") {
		t.Error("Markdown should contain 'Notion' source")
	}
	if !contains(brief.ScriptMarkdown, "GitHub") {
		t.Error("Markdown should contain 'GitHub' source")
	}
}

func TestGenerateDaily_ScriptWithIncident(t *testing.T) {
	summarizer := NewRuleSummarizer(10, 25)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	since := today.Add(-24 * time.Hour)
	until := today
	events := model.Events{
		{ID: "1", Title: "障害発生", Category: model.EventCategoryIncident, Importance: 90, Timestamp: now, Location: "ops"},
	}

	brief, err := summarizer.GenerateDaily(events, since, until)
	if err != nil {
		t.Fatalf("GenerateDaily failed: %v", err)
	}

	// インシデントがある場合のスクリプト
	if !contains(brief.ScriptText, "障害やインシデント関連") {
		t.Error("Script should mention incident when incident events exist")
	}
	if !contains(brief.ScriptText, "確認が必要") {
		t.Error("Script should mention '確認が必要' for incidents")
	}
}

func TestGenerateDaily_EventWithURL(t *testing.T) {
	summarizer := NewRuleSummarizer(10, 25)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	since := today.Add(-24 * time.Hour)
	until := today
	events := model.Events{
		{
			ID:         "1",
			Title:      "PR Review",
			Source:     model.EventSourceSlack,
			Category:   model.EventCategoryDev,
			Importance: 90,
			Timestamp:  now,
			Location:   "dev",
			URL:        "https://github.com/org/repo/pull/123",
		},
	}

	brief, err := summarizer.GenerateDaily(events, since, until)
	if err != nil {
		t.Fatalf("GenerateDaily failed: %v", err)
	}

	// URLがMarkdownに含まれていることを確認
	if !contains(brief.ScriptMarkdown, "https://github.com") {
		t.Error("Markdown should contain the URL")
	}
	if !contains(brief.ScriptMarkdown, "リンク") {
		t.Error("Markdown should contain 'リンク' label for URL")
	}
}

func TestGenerateWeekly_TopEvents(t *testing.T) {
	summarizer := NewRuleSummarizer(8, 25)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	since := today.Add(-7 * 24 * time.Hour)
	until := today
	events := model.Events{
		{ID: "1", Title: "Most important", Importance: 100, Timestamp: now, Location: "channel1"},
		{ID: "2", Title: "Second", Importance: 90, Timestamp: now, Location: "channel2"},
		{ID: "3", Title: "Third", Importance: 80, Timestamp: now, Location: "channel3"},
		{ID: "4", Title: "Fourth", Importance: 70, Timestamp: now, Location: "channel4"},
		{ID: "5", Title: "Fifth", Importance: 60, Timestamp: now, Location: "channel5"},
		{ID: "6", Title: "Sixth", Importance: 50, Timestamp: now, Location: "channel6"},
	}

	brief, err := summarizer.GenerateWeekly(events, since, until)
	if err != nil {
		t.Fatalf("GenerateWeekly failed: %v", err)
	}

	// Top 5がMarkdownに含まれていることを確認
	if !contains(brief.ScriptMarkdown, "Top 5") {
		t.Error("Markdown should contain 'Top 5'")
	}
	if !contains(brief.ScriptMarkdown, "Most important") {
		t.Error("Markdown should contain top event title")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
