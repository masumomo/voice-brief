package categorizer

import (
	"strings"

	"github.com/masumomo/voice-brief/internal/model"
)

// RuleCategorizer はルールベースのカテゴリ判定
type RuleCategorizer struct{}

// NewRuleCategorizer は新しいRuleCategorizerを作成します
func NewRuleCategorizer() *RuleCategorizer {
	return &RuleCategorizer{}
}

// Categorize はイベントのカテゴリを判定します
func (c *RuleCategorizer) Categorize(event *model.Event) string {
	switch event.Source {
	case model.EventSourceSlack:
		return c.categorizeSlack(event)
	case model.EventSourceNotion:
		return c.categorizeNotion(event)
	case model.EventSourceGitHub:
		return c.categorizeGitHub(event)
	default:
		return c.categorizeByText(event.Title + " " + event.Body)
	}
}

// CategorizeAll は複数イベントのカテゴリを一括判定します
func (c *RuleCategorizer) CategorizeAll(events model.Events) {
	for _, event := range events {
		if event.Category == "" || event.Category == model.EventCategoryOther {
			event.Category = c.Categorize(event)
		}
	}
}

// categorizeSlack はSlackメッセージのカテゴリを判定します
func (c *RuleCategorizer) categorizeSlack(event *model.Event) string {
	text := strings.ToLower(event.Title + " " + event.Body)

	// チャンネル名からの判定
	location := strings.ToLower(event.Location)
	if containsAny(location, []string{"incident", "alert", "障害", "緊急"}) {
		return model.EventCategoryIncident
	}
	if containsAny(location, []string{"dev", "engineer", "開発", "技術"}) {
		return model.EventCategoryDev
	}
	if containsAny(location, []string{"biz", "sales", "営業", "ビジネス"}) {
		return model.EventCategoryBiz
	}
	if containsAny(location, []string{"ops", "infra", "運用", "インフラ"}) {
		return model.EventCategoryOps
	}

	// 本文からの判定
	return c.categorizeByText(text)
}

// categorizeNotion はNotionページのカテゴリを判定します
func (c *RuleCategorizer) categorizeNotion(event *model.Event) string {
	// プロパティからの判定
	if properties, ok := event.Refs["properties"]; ok {
		lower := strings.ToLower(properties)

		// Statusベースの判定
		if containsAny(lower, []string{"blocked", "ブロック", "問題"}) {
			return model.EventCategoryIncident
		}

		// タグベースの判定
		if containsAny(lower, []string{"dev", "開発", "技術", "development"}) {
			return model.EventCategoryDev
		}
		if containsAny(lower, []string{"biz", "business", "ビジネス", "営業"}) {
			return model.EventCategoryBiz
		}
		if containsAny(lower, []string{"ops", "運用", "インフラ", "operations"}) {
			return model.EventCategoryOps
		}
		if containsAny(lower, []string{"incident", "障害", "緊急", "問題"}) {
			return model.EventCategoryIncident
		}
	}

	// DB名からの判定
	location := strings.ToLower(event.Location)
	if containsAny(location, []string{"incident", "障害"}) {
		return model.EventCategoryIncident
	}
	if containsAny(location, []string{"dev", "開発", "tech"}) {
		return model.EventCategoryDev
	}

	// 本文からの判定
	return c.categorizeByText(event.Title + " " + event.Body)
}

// categorizeGitHub はGitHubイベントのカテゴリを判定します
func (c *RuleCategorizer) categorizeGitHub(event *model.Event) string {
	text := strings.ToLower(event.Title + " " + event.Body)

	// ラベルからの判定
	if labels, ok := event.Refs["labels"]; ok {
		lower := strings.ToLower(labels)
		if containsAny(lower, []string{"bug", "incident", "critical", "urgent"}) {
			return model.EventCategoryIncident
		}
		if containsAny(lower, []string{"feature", "enhancement"}) {
			return model.EventCategoryDev
		}
		if containsAny(lower, []string{"ops", "infra", "ci", "deploy"}) {
			return model.EventCategoryOps
		}
	}

	// Conventional Commits形式の判定
	if strings.HasPrefix(text, "fix:") || strings.HasPrefix(text, "hotfix:") {
		return model.EventCategoryIncident
	}
	if strings.HasPrefix(text, "feat:") || strings.HasPrefix(text, "feature:") {
		return model.EventCategoryDev
	}
	if strings.HasPrefix(text, "refactor:") || strings.HasPrefix(text, "chore:") {
		return model.EventCategoryDev
	}
	if strings.HasPrefix(text, "ci:") || strings.HasPrefix(text, "ops:") {
		return model.EventCategoryOps
	}

	// キーワードベースの判定
	if containsAny(text, []string{"bug", "fix", "error", "issue", "problem"}) {
		return model.EventCategoryIncident
	}
	if containsAny(text, []string{"feat", "add", "implement", "create"}) {
		return model.EventCategoryDev
	}

	return model.EventCategoryOther
}

// categorizeByText はテキスト内容からカテゴリを判定します
func (c *RuleCategorizer) categorizeByText(text string) string {
	lower := strings.ToLower(text)

	// 緊急・障害系
	if containsAny(lower, []string{
		"障害", "緊急", "エラー", "バグ", "incident", "urgent", "critical",
		"error", "bug", "failure", "outage", "down", "broken",
	}) {
		return model.EventCategoryIncident
	}

	// 開発系
	if containsAny(lower, []string{
		"実装", "開発", "機能", "feature", "implement", "develop",
		"コード", "code", "pr", "pull request", "merge", "リリース", "release",
	}) {
		return model.EventCategoryDev
	}

	// ビジネス系
	if containsAny(lower, []string{
		"営業", "売上", "顧客", "契約", "sales", "customer", "revenue",
		"マーケ", "marketing", "提案", "proposal",
	}) {
		return model.EventCategoryBiz
	}

	// 運用系
	if containsAny(lower, []string{
		"運用", "デプロイ", "インフラ", "監視", "ops", "deploy", "infra",
		"monitoring", "ci/cd", "pipeline",
	}) {
		return model.EventCategoryOps
	}

	return model.EventCategoryOther
}

// containsAny は文字列にいずれかのキーワードが含まれるかチェックします
func containsAny(s string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(s, keyword) {
			return true
		}
	}
	return false
}
