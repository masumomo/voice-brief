package model

import (
	"testing"
	"time"
)

func TestNewBrief(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	brief := NewBrief(BriefTypeDaily, start, end)

	if brief.Type != BriefTypeDaily {
		t.Errorf("expected Type=daily, got %s", brief.Type)
	}
	if !brief.WindowStart.Equal(start) {
		t.Errorf("expected WindowStart=%v, got %v", start, brief.WindowStart)
	}
	if !brief.WindowEnd.Equal(end) {
		t.Errorf("expected WindowEnd=%v, got %v", end, brief.WindowEnd)
	}
	if len(brief.Items) != 0 {
		t.Errorf("expected empty Items, got length %d", len(brief.Items))
	}
}

func TestBrief_AddEvent(t *testing.T) {
	brief := NewBrief(BriefTypeDaily, time.Now(), time.Now())
	event := &Event{
		ID:    "test-1",
		Title: "Test Event",
	}

	brief.AddEvent(event)

	if len(brief.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(brief.Items))
	}
	if brief.Items[0].ID != "test-1" {
		t.Errorf("expected ID=test-1, got %s", brief.Items[0].ID)
	}
}

func TestBrief_AddEvents(t *testing.T) {
	brief := NewBrief(BriefTypeDaily, time.Now(), time.Now())
	events := Events{
		{ID: "1", Title: "Event 1"},
		{ID: "2", Title: "Event 2"},
		{ID: "3", Title: "Event 3"},
	}

	brief.AddEvents(events)

	if len(brief.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(brief.Items))
	}
}

func TestBrief_GetEventCount(t *testing.T) {
	brief := NewBrief(BriefTypeDaily, time.Now(), time.Now())

	if brief.GetEventCount() != 0 {
		t.Errorf("expected count=0, got %d", brief.GetEventCount())
	}

	brief.AddEvents(Events{
		{ID: "1"},
		{ID: "2"},
	})

	if brief.GetEventCount() != 2 {
		t.Errorf("expected count=2, got %d", brief.GetEventCount())
	}
}

func TestBrief_GetEventsBySource(t *testing.T) {
	brief := NewBrief(BriefTypeDaily, time.Now(), time.Now())
	brief.AddEvents(Events{
		{ID: "1", Source: "slack"},
		{ID: "2", Source: "notion"},
		{ID: "3", Source: "slack"},
	})

	slackEvents := brief.GetEventsBySource("slack")

	if len(slackEvents) != 2 {
		t.Errorf("expected 2 slack events, got %d", len(slackEvents))
	}
	for _, event := range slackEvents {
		if event.Source != "slack" {
			t.Errorf("expected source=slack, got %s", event.Source)
		}
	}
}

func TestBrief_GetEventsByCategory(t *testing.T) {
	brief := NewBrief(BriefTypeDaily, time.Now(), time.Now())
	brief.AddEvents(Events{
		{ID: "1", Category: "discussion"},
		{ID: "2", Category: "decision"},
		{ID: "3", Category: "discussion"},
	})

	discussions := brief.GetEventsByCategory("discussion")

	if len(discussions) != 2 {
		t.Errorf("expected 2 discussion events, got %d", len(discussions))
	}
}

func TestBrief_GetTopEvents(t *testing.T) {
	brief := NewBrief(BriefTypeDaily, time.Now(), time.Now())
	brief.AddEvents(Events{
		{ID: "1", Importance: 100},
		{ID: "2", Importance: 90},
		{ID: "3", Importance: 80},
		{ID: "4", Importance: 70},
	})

	tests := []struct {
		name     string
		n        int
		expected int
	}{
		{"top 2", 2, 2},
		{"top 5 (more than available)", 5, 4},
		{"top 0", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			top := brief.GetTopEvents(tt.n)
			if len(top) != tt.expected {
				t.Errorf("expected %d events, got %d", tt.expected, len(top))
			}
		})
	}
}

func TestBrief_GetDurationString_Daily(t *testing.T) {
	start := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	brief := NewBrief(BriefTypeDaily, start, start.Add(24*time.Hour))

	durationStr := brief.GetDurationString()

	expected := "2025年01月15日"
	if durationStr != expected {
		t.Errorf("expected %s, got %s", expected, durationStr)
	}
}

func TestBrief_GetDurationString_Weekly(t *testing.T) {
	start := time.Date(2025, 1, 6, 0, 0, 0, 0, time.UTC) // 2025年第2週の月曜日
	end := start.Add(7 * 24 * time.Hour)
	brief := NewBrief(BriefTypeWeekly, start, end)

	durationStr := brief.GetDurationString()

	// "2025年01月06日〜01月13日 (第02週)" のような形式を期待
	if len(durationStr) == 0 {
		t.Errorf("expected non-empty duration string")
	}

	// 少なくとも年月日が含まれていることを確認
	if !contains(durationStr, "2025年01月06日") {
		t.Errorf("expected duration string to contain start date, got %s", durationStr)
	}
}

// helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && contains(s[1:], substr)
}
