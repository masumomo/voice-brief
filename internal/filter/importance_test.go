package filter

import (
	"testing"
	"time"

	"github.com/masumomo/voice-brief/internal/model"
)

func TestCalculate_HighPriorityKeyword(t *testing.T) {
	calc := NewRuleBasedCalculator()
	event := &model.Event{
		Title:     "緊急：本番環境で障害発生",
		Body:      "詳細を確認中です",
		Timestamp: time.Now(),
	}

	importance := calc.Calculate(event)

	if importance <= 50 {
		t.Errorf("Importance = %d; want > 50 (high priority keyword)", importance)
	}
}

func TestCalculate_LowPriorityKeyword(t *testing.T) {
	calc := NewRuleBasedCalculator()
	event := &model.Event{
		Title:     "Re: 打ち合わせ",
		Body:      "了解です",
		Timestamp: time.Now(),
	}

	importance := calc.Calculate(event)

	if importance >= 50 {
		t.Errorf("Importance = %d; want < 50 (low priority keyword)", importance)
	}
}

func TestCalculate_Mention(t *testing.T) {
	calc := NewRuleBasedCalculator()
	event := &model.Event{
		Title:     "レビュー依頼",
		Body:      "@user さん、こちらのPRをレビューお願いします",
		Timestamp: time.Now(),
	}

	importance := calc.Calculate(event)

	if importance <= 50 {
		t.Errorf("Importance = %d; want > 50 (mention included)", importance)
	}
}

func TestCalculate_ShortMessage(t *testing.T) {
	calc := NewRuleBasedCalculator()
	event := &model.Event{
		Title:     "確認",
		Body:      "OK",
		Timestamp: time.Now(),
	}

	importance := calc.Calculate(event)

	if importance >= 50 {
		t.Errorf("Importance = %d; want < 50 (short message)", importance)
	}
}

func TestCalculate_IncidentCategory(t *testing.T) {
	calc := NewRuleBasedCalculator()
	event := &model.Event{
		Title:     "システム停止",
		Body:      "調査中",
		Category:  model.EventCategoryIncident,
		Timestamp: time.Now(),
	}

	importance := calc.Calculate(event)

	if importance <= 70 {
		t.Errorf("Importance = %d; want > 70 (incident category)", importance)
	}
}

func TestCalculate_Range(t *testing.T) {
	calc := NewRuleBasedCalculator()
	event := &model.Event{
		Title:     "緊急緊急緊急",
		Body:      "超重要なメッセージです @channel",
		Category:  model.EventCategoryIncident,
		Timestamp: time.Now(),
	}

	importance := calc.Calculate(event)

	if importance < 0 || importance > 100 {
		t.Errorf("Importance = %d; want 0-100 range", importance)
	}
}

func TestCalculateAll(t *testing.T) {
	calc := NewRuleBasedCalculator()
	events := model.Events{
		{Title: "緊急", Body: "障害発生", Category: model.EventCategoryIncident},
		{Title: "通常", Body: "定例会議のリマインド", Category: model.EventCategoryBiz},
		{Title: "確認", Body: "了解", Category: model.EventCategoryOther},
	}

	CalculateAll(events, calc)

	if events[0].Importance <= events[1].Importance {
		t.Errorf("Event 0 importance (%d) should be > Event 1 (%d)", events[0].Importance, events[1].Importance)
	}
	if events[1].Importance <= events[2].Importance {
		t.Errorf("Event 1 importance (%d) should be > Event 2 (%d)", events[1].Importance, events[2].Importance)
	}
}

func TestFilterByKeywords(t *testing.T) {
	events := model.Events{
		{Title: "打ち合わせ", Body: "了解です"},
		{Title: "レビュー", Body: "確認しました"},
		{Title: "障害", Body: "調査中"},
	}

	filtered := FilterByKeywords(events, []string{"了解"})

	if len(filtered) != 2 {
		t.Errorf("Filtered count = %d; want 2", len(filtered))
	}

	for _, e := range filtered {
		if e.Title == "打ち合わせ" {
			t.Error("Event with '了解' should be filtered out")
		}
	}
}

func TestFilterShortMessages(t *testing.T) {
	events := model.Events{
		{Title: "短い", Body: "OK"},
		{Title: "普通", Body: "これは普通の長さのメッセージです"},
		{Title: "長い", Body: "これはとても長いメッセージで詳細な情報が含まれています"},
	}

	filtered := FilterShortMessages(events, 10)

	if len(filtered) != 2 {
		t.Errorf("Filtered count = %d; want 2", len(filtered))
	}

	for _, e := range filtered {
		if len(e.Body) < 10 {
			t.Errorf("Event with body '%s' (len=%d) should be filtered out", e.Body, len(e.Body))
		}
	}
}
