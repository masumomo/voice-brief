package model

import (
	"sort"
	"testing"
	"time"
)

func TestNewEvent(t *testing.T) {
	event := NewEvent(EventSourceSlack)

	if event.Source != EventSourceSlack {
		t.Errorf("Source = %s; want %s", event.Source, EventSourceSlack)
	}
	if event.Category != EventCategoryOther {
		t.Errorf("Category = %s; want %s", event.Category, EventCategoryOther)
	}
	if event.Importance != 50 {
		t.Errorf("Importance = %d; want 50", event.Importance)
	}
	if event.Refs == nil {
		t.Error("Refs should be initialized")
	}
}

func TestEventsSort(t *testing.T) {
	events := Events{
		{ID: "1", Importance: 30},
		{ID: "2", Importance: 80},
		{ID: "3", Importance: 50},
	}

	sort.Sort(events)

	if events[0].ID != "2" {
		t.Errorf("First event ID = %s; want 2 (highest importance)", events[0].ID)
	}
	if events[1].ID != "3" {
		t.Errorf("Second event ID = %s; want 3", events[1].ID)
	}
	if events[2].ID != "1" {
		t.Errorf("Third event ID = %s; want 1 (lowest importance)", events[2].ID)
	}
}

func TestFilterByImportance(t *testing.T) {
	events := Events{
		{ID: "1", Importance: 30},
		{ID: "2", Importance: 80},
		{ID: "3", Importance: 50},
	}

	filtered := events.FilterByImportance(50)

	if len(filtered) != 2 {
		t.Errorf("Filtered count = %d; want 2", len(filtered))
	}

	for _, e := range filtered {
		if e.Importance < 50 {
			t.Errorf("Event %s has importance %d; want >= 50", e.ID, e.Importance)
		}
	}
}

func TestFilterBySource(t *testing.T) {
	events := Events{
		{ID: "1", Source: EventSourceSlack},
		{ID: "2", Source: EventSourceNotion},
		{ID: "3", Source: EventSourceSlack},
	}

	filtered := events.FilterBySource(EventSourceSlack)

	if len(filtered) != 2 {
		t.Errorf("Filtered count = %d; want 2", len(filtered))
	}

	for _, e := range filtered {
		if e.Source != EventSourceSlack {
			t.Errorf("Event %s has source %s; want %s", e.ID, e.Source, EventSourceSlack)
		}
	}
}

func TestFilterByCategory(t *testing.T) {
	now := time.Now()
	events := Events{
		{ID: "1", Category: EventCategoryDev, Timestamp: now},
		{ID: "2", Category: EventCategoryBiz, Timestamp: now},
		{ID: "3", Category: EventCategoryDev, Timestamp: now},
	}

	filtered := events.FilterByCategory(EventCategoryDev)

	if len(filtered) != 2 {
		t.Errorf("Filtered count = %d; want 2", len(filtered))
	}

	for _, e := range filtered {
		if e.Category != EventCategoryDev {
			t.Errorf("Event %s has category %s; want %s", e.ID, e.Category, EventCategoryDev)
		}
	}
}
