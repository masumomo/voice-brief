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

	brief, err := summarizer.GenerateDaily(events)
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

	brief, err := summarizer.GenerateWeekly(events)
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
	events := model.Events{}

	brief, err := summarizer.GenerateDaily(events)
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
