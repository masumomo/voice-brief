package importance

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

// FeatureBasedCalculator Tests

func TestFeatureBasedCalculator_HighPriorityKeyword(t *testing.T) {
	calc := NewFeatureBasedCalculator(nil)
	event := &model.Event{
		Title:     "緊急：本番環境で障害発生",
		Body:      "詳細を確認中です",
		Source:    model.EventSourceSlack,
		Category:  model.EventCategoryIncident,
		Timestamp: time.Now(),
	}

	importance := calc.Calculate(event)

	if importance <= 50 {
		t.Errorf("Importance = %d; want > 50 (high priority keyword + incident)", importance)
	}
}

func TestFeatureBasedCalculator_LowPriorityKeyword(t *testing.T) {
	calc := NewFeatureBasedCalculator(nil)
	event := &model.Event{
		Title:     "Re: 打ち合わせ",
		Body:      "了解です",
		Source:    model.EventSourceSlack,
		Category:  model.EventCategoryOther,
		Timestamp: time.Now(),
	}

	importance := calc.Calculate(event)

	// 低優先度キーワードでベーススコアが下がる
	if importance >= 60 {
		t.Errorf("Importance = %d; want < 60 (low priority keyword)", importance)
	}
}

func TestFeatureBasedCalculator_WithMention(t *testing.T) {
	calc := NewFeatureBasedCalculator(nil)
	event := &model.Event{
		Title:     "レビュー依頼",
		Body:      "@user さん、こちらのPRをレビューお願いします",
		Source:    model.EventSourceSlack,
		Timestamp: time.Now(),
	}

	importance := calc.Calculate(event)

	// メンション付きで重要度が上がる
	if importance <= 40 {
		t.Errorf("Importance = %d; want > 40 (has mention)", importance)
	}
}

func TestFeatureBasedCalculator_WithComments(t *testing.T) {
	calc := NewFeatureBasedCalculator(nil)
	event := &model.Event{
		Title:     "技術的な議論",
		Body:      "アーキテクチャについて議論しましょう",
		Source:    model.EventSourceSlack,
		Timestamp: time.Now(),
		Comments: []model.Comment{
			{Author: "user1", Text: "賛成です"},
			{Author: "user2", Text: "こちらの方がいいかも"},
			{Author: "user3", Text: "両方検討しましょう"},
			{Author: "user1", Text: "了解です"},
		},
	}

	importance := calc.Calculate(event)

	// コメントが多いと重要度が上がる
	if importance <= 40 {
		t.Errorf("Importance = %d; want > 40 (has many comments)", importance)
	}
}

func TestFeatureBasedCalculator_Range(t *testing.T) {
	calc := NewFeatureBasedCalculator(nil)

	// 非常に高い重要度になりそうなイベント
	event := &model.Event{
		Title:     "緊急緊急緊急",
		Body:      "超重要なメッセージです @channel https://example.com",
		Category:  model.EventCategoryIncident,
		Source:    model.EventSourceSlack,
		Timestamp: time.Now(),
		Comments: []model.Comment{
			{Author: "user1", Text: "確認します"},
			{Author: "user2", Text: "対応中"},
			{Author: "user3", Text: "完了"},
		},
	}

	importance := calc.Calculate(event)

	if importance < 0 || importance > 100 {
		t.Errorf("Importance = %d; want 0-100 range", importance)
	}
}

func TestFeatureBasedCalculator_ExtractFeatures(t *testing.T) {
	calc := NewFeatureBasedCalculator(nil)
	event := &model.Event{
		Title:     "緊急：障害発生",
		Body:      "@channel 本番環境で障害が発生しています https://example.com/incident",
		Source:    model.EventSourceSlack,
		Category:  model.EventCategoryIncident,
		Timestamp: time.Now(),
		Comments: []model.Comment{
			{Author: "user1", Text: "確認中"},
		},
	}

	features := calc.ExtractFeatures(event)

	// メンションあり
	if features.HasMention != 1.0 {
		t.Errorf("HasMention = %f; want 1.0", features.HasMention)
	}

	// URLあり
	if features.HasURL != 1.0 {
		t.Errorf("HasURL = %f; want 1.0", features.HasURL)
	}

	// Incidentカテゴリ
	if features.IsIncident != 1.0 {
		t.Errorf("IsIncident = %f; want 1.0", features.IsIncident)
	}

	// Slackソース
	if features.IsSlack != 1.0 {
		t.Errorf("IsSlack = %f; want 1.0", features.IsSlack)
	}

	// コメント数 > 0
	if features.CommentCount <= 0 {
		t.Errorf("CommentCount = %f; want > 0", features.CommentCount)
	}
}

func TestFeatureBasedCalculator_GetFeatureVector(t *testing.T) {
	calc := NewFeatureBasedCalculator(nil)
	event := &model.Event{
		Title:     "テスト",
		Body:      "テストメッセージ",
		Source:    model.EventSourceNotion,
		Timestamp: time.Now(),
	}

	vector := calc.GetFeatureVector(event)

	// 特徴量数は20個
	if len(vector) != 20 {
		t.Errorf("Feature vector length = %d; want 20", len(vector))
	}
}

func TestGetFeatureNames(t *testing.T) {
	names := GetFeatureNames()

	// 特徴量名は20個
	if len(names) != 20 {
		t.Errorf("Feature names count = %d; want 20", len(names))
	}
}

// TopK Tests

func TestTopK_ReturnsTopKEvents(t *testing.T) {
	events := model.Events{
		{ID: "1", Title: "Low", Importance: 10},
		{ID: "2", Title: "High", Importance: 90},
		{ID: "3", Title: "Medium", Importance: 50},
		{ID: "4", Title: "Very High", Importance: 95},
		{ID: "5", Title: "Medium High", Importance: 70},
	}

	topK := TopK(events, 3)

	if len(topK) != 3 {
		t.Errorf("TopK count = %d; want 3", len(topK))
	}

	// 上位3件は importance 95, 90, 70 の順
	if topK[0].Importance != 95 {
		t.Errorf("TopK[0].Importance = %d; want 95", topK[0].Importance)
	}
	if topK[1].Importance != 90 {
		t.Errorf("TopK[1].Importance = %d; want 90", topK[1].Importance)
	}
	if topK[2].Importance != 70 {
		t.Errorf("TopK[2].Importance = %d; want 70", topK[2].Importance)
	}
}

func TestTopK_ReturnsAllWhenKIsZero(t *testing.T) {
	events := model.Events{
		{ID: "1", Title: "Event 1", Importance: 50},
		{ID: "2", Title: "Event 2", Importance: 60},
	}

	topK := TopK(events, 0)

	if len(topK) != len(events) {
		t.Errorf("TopK(0) count = %d; want %d (all events)", len(topK), len(events))
	}
}

func TestTopK_ReturnsAllWhenKIsNegative(t *testing.T) {
	events := model.Events{
		{ID: "1", Title: "Event 1", Importance: 50},
		{ID: "2", Title: "Event 2", Importance: 60},
	}

	topK := TopK(events, -1)

	if len(topK) != len(events) {
		t.Errorf("TopK(-1) count = %d; want %d (all events)", len(topK), len(events))
	}
}

func TestTopK_ReturnsAllWhenKGreaterThanLength(t *testing.T) {
	events := model.Events{
		{ID: "1", Title: "Event 1", Importance: 50},
		{ID: "2", Title: "Event 2", Importance: 60},
	}

	topK := TopK(events, 10)

	if len(topK) != len(events) {
		t.Errorf("TopK(10) count = %d; want %d (all events)", len(topK), len(events))
	}
}

func TestTopK_DoesNotModifyOriginal(t *testing.T) {
	events := model.Events{
		{ID: "1", Title: "Low", Importance: 10},
		{ID: "2", Title: "High", Importance: 90},
		{ID: "3", Title: "Medium", Importance: 50},
	}

	_ = TopK(events, 2)

	// 元のスライスの順序が変わっていないことを確認
	if events[0].ID != "1" || events[1].ID != "2" || events[2].ID != "3" {
		t.Error("TopK modified the original slice")
	}
}
