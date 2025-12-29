package fetcher

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jomei/notionapi"
	"github.com/masumomo/voice-brief/internal/config"
	"github.com/masumomo/voice-brief/internal/filter"
	"github.com/masumomo/voice-brief/internal/model"
)

// NotionFetcher はNotionからイベントを取得します
type NotionFetcher struct {
	client     *notionapi.Client
	config     *config.NotionConfig
	calculator filter.ImportanceCalculator
}

// NewNotionFetcher は新しいNotionFetcherを作成します
func NewNotionFetcher(cfg *config.NotionConfig) *NotionFetcher {
	return &NotionFetcher{
		client:     notionapi.NewClient(notionapi.Token(cfg.Token)),
		config:     cfg,
		calculator: filter.NewRuleBasedCalculator(),
	}
}

// TestConnection はNotion APIへの接続をテストします
func (f *NotionFetcher) TestConnection(ctx context.Context) error {
	// 最初のデータベースでテスト
	if len(f.config.Databases) == 0 {
		return fmt.Errorf("監視対象データベースが設定されていません")
	}

	dbID := notionapi.DatabaseID(f.config.Databases[0].ID)
	_, err := f.client.Database.Get(ctx, dbID)
	if err != nil {
		return fmt.Errorf("Notion API接続テストに失敗: %w", err)
	}

	return nil
}

// Fetch は指定期間に更新されたページを取得します
func (f *NotionFetcher) Fetch(ctx context.Context, since time.Time) (model.Events, error) {
	allEvents := make(model.Events, 0)

	for _, database := range f.config.Databases {
		events, err := f.fetchDatabase(ctx, database, since)
		if err != nil {
			// エラーログを出力するが、他のDBの取得は継続（Best Effort）
			fmt.Printf("⚠️  警告: データベース %s (%s) の取得に失敗: %v\n", database.Name, database.ID, err)
			continue
		}
		allEvents = append(allEvents, events...)
	}

	// 重要度計算
	filter.CalculateAll(allEvents, f.calculator)

	return allEvents, nil
}

// fetchDatabase は単一データベースからページを取得します
func (f *NotionFetcher) fetchDatabase(ctx context.Context, database config.DatabaseConfig, since time.Time) (model.Events, error) {
	events := make(model.Events, 0)

	dbID := notionapi.DatabaseID(database.ID)

	// last_edited_time でフィルタ
	// Notion APIの制限により、タイムスタンプフィルタは現在未サポート
	// 代わりに全件取得後にアプリケーション側でフィルタリング
	filter := &notionapi.DatabaseQueryRequest{
		Sorts: []notionapi.SortObject{
			{
				Timestamp: "last_edited_time",
				Direction: notionapi.SortOrderDESC,
			},
		},
		PageSize: 100, // 最大100件
	}

	resp, err := f.client.Database.Query(ctx, dbID, filter)
	if err != nil {
		return nil, fmt.Errorf("データベースクエリに失敗: %w", err)
	}

	for _, page := range resp.Results {
		// アプリケーション側で期間フィルタリング
		pageEditedTime := time.Time(page.LastEditedTime)
		if pageEditedTime.Before(since) {
			continue // since より前のページは除外
		}

		// プロパティフィルタ適用
		if !f.matchesPropertyFilters(&page, database.PropertyFilters) {
			continue // フィルタ条件に合わないページは除外
		}

		event := f.pageToEvent(ctx, &page, database)
		events = append(events, event)
	}

	return events, nil
}

// pageToEvent はNotionページをEventに変換します
func (f *NotionFetcher) pageToEvent(ctx context.Context, page *notionapi.Page, database config.DatabaseConfig) *model.Event {
	event := model.NewEvent(model.EventSourceNotion)

	event.ID = string(page.ID)
	event.Timestamp = time.Time(page.LastEditedTime)
	event.Location = database.Name
	event.URL = page.URL

	// タイトル取得
	event.Title = f.extractTitle(page)

	// プロパティ情報を本文に含める
	event.Body = f.extractProperties(page, database.Properties)

	// ページ本文取得（オプション）
	if database.FetchPageContent {
		pageContent := f.fetchPageContent(ctx, notionapi.PageID(page.ID))
		if pageContent != "" {
			event.Body = event.Body + "\n\n" + pageContent
		}
	}

	// 最終編集者
	// User情報は簡易的にIDを使用（詳細取得はAPI制限を考慮して省略）
	if page.LastEditedBy.ID != "" {
		event.Author = string(page.LastEditedBy.ID)
	} else {
		event.Author = "Unknown"
	}

	// Refs に追加情報を格納
	event.Refs["database_id"] = database.ID
	event.Refs["database_name"] = database.Name
	event.Refs["page_id"] = string(page.ID)

	// カテゴリの判定（プロパティから）
	event.Category = f.detectCategoryFromProperties(page, database)

	// プロジェクト情報の追加
	if database.ProjectProperty != "" {
		if project := f.getPropertyValue(page, database.ProjectProperty); project != "" {
			event.Refs["project"] = project
		}
	}

	return event
}

// extractTitle はページタイトルを抽出します
func (f *NotionFetcher) extractTitle(page *notionapi.Page) string {
	// Notionのタイトルプロパティを探す
	for _, prop := range page.Properties {
		if prop.GetType() == notionapi.PropertyTypeTitle {
			titleProp := prop.(*notionapi.TitleProperty)
			if len(titleProp.Title) > 0 {
				return extractRichText(titleProp.Title)
			}
		}
	}

	return "Untitled"
}

// extractProperties は指定されたプロパティの情報を抽出します
func (f *NotionFetcher) extractProperties(page *notionapi.Page, targetProps []string) string {
	parts := make([]string, 0)

	for _, propName := range targetProps {
		if prop, exists := page.Properties[propName]; exists {
			value := f.formatPropertyValue(prop)
			if value != "" {
				parts = append(parts, fmt.Sprintf("%s: %s", propName, value))
			}
		}
	}

	return strings.Join(parts, " | ")
}

// formatPropertyValue はプロパティ値をフォーマットします
func (f *NotionFetcher) formatPropertyValue(prop notionapi.Property) string {
	switch prop.GetType() {
	case notionapi.PropertyTypeRichText:
		richTextProp := prop.(*notionapi.RichTextProperty)
		return extractRichText(richTextProp.RichText)

	case notionapi.PropertyTypeSelect:
		selectProp := prop.(*notionapi.SelectProperty)
		if selectProp.Select.Name != "" {
			return selectProp.Select.Name
		}

	case notionapi.PropertyTypeMultiSelect:
		multiSelectProp := prop.(*notionapi.MultiSelectProperty)
		names := make([]string, 0)
		for _, option := range multiSelectProp.MultiSelect {
			names = append(names, option.Name)
		}
		return strings.Join(names, ", ")

	case notionapi.PropertyTypeStatus:
		statusProp := prop.(*notionapi.StatusProperty)
		if statusProp.Status.Name != "" {
			return statusProp.Status.Name
		}

	case notionapi.PropertyTypeDate:
		dateProp := prop.(*notionapi.DateProperty)
		if dateProp.Date != nil && dateProp.Date.Start != nil {
			// notionapi.Date構造体からtime.Timeを取得
			t := time.Time(*dateProp.Date.Start)
			return t.Format("2006-01-02")
		}

	case notionapi.PropertyTypePeople:
		peopleProp := prop.(*notionapi.PeopleProperty)
		names := make([]string, 0)
		for _, person := range peopleProp.People {
			if person.Name != "" {
				names = append(names, person.Name)
			}
		}
		return strings.Join(names, ", ")

	case notionapi.PropertyTypeCheckbox:
		checkboxProp := prop.(*notionapi.CheckboxProperty)
		if checkboxProp.Checkbox {
			return "✓"
		}
		return "☐"

	case notionapi.PropertyTypeNumber:
		numberProp := prop.(*notionapi.NumberProperty)
		return fmt.Sprintf("%v", numberProp.Number)
	}

	return ""
}

// detectCategoryFromProperties はプロパティからカテゴリを判定します
func (f *NotionFetcher) detectCategoryFromProperties(page *notionapi.Page, database config.DatabaseConfig) string {
	// CategoryProperty が指定されている場合はそれを優先
	if database.CategoryProperty != "" {
		if value := f.getPropertyValue(page, database.CategoryProperty); value != "" {
			return f.mapToCategory(value)
		}
	}

	// Status や Tag などから判定
	for _, propName := range database.Properties {
		if prop, exists := page.Properties[propName]; exists {
			value := strings.ToLower(f.formatPropertyValue(prop))

			// Status ベースの判定
			if strings.Contains(propName, "Status") || strings.Contains(propName, "ステータス") {
				if containsAny(value, []string{"blocked", "ブロック", "問題"}) {
					return model.EventCategoryIncident
				}
				if containsAny(value, []string{"in progress", "進行中", "doing"}) {
					return model.EventCategoryDev
				}
			}

			// Tag ベースの判定
			if strings.Contains(propName, "Tag") || strings.Contains(propName, "タグ") {
				if containsAny(value, []string{"dev", "開発", "技術"}) {
					return model.EventCategoryDev
				}
				if containsAny(value, []string{"biz", "business", "ビジネス", "営業"}) {
					return model.EventCategoryBiz
				}
				if containsAny(value, []string{"ops", "運用", "インフラ"}) {
					return model.EventCategoryOps
				}
			}
		}
	}

	return model.EventCategoryOther
}

// mapToCategory はプロパティ値をカテゴリにマッピングします
func (f *NotionFetcher) mapToCategory(value string) string {
	lower := strings.ToLower(value)

	if containsAny(lower, []string{"incident", "障害", "問題", "blocked"}) {
		return model.EventCategoryIncident
	}
	if containsAny(lower, []string{"dev", "開発", "技術", "development"}) {
		return model.EventCategoryDev
	}
	if containsAny(lower, []string{"biz", "business", "ビジネス", "営業"}) {
		return model.EventCategoryBiz
	}
	if containsAny(lower, []string{"ops", "運用", "インフラ", "operations"}) {
		return model.EventCategoryOps
	}

	return model.EventCategoryOther
}

// extractRichText はRichTextの配列からプレーンテキストを抽出します
func extractRichText(richTexts []notionapi.RichText) string {
	parts := make([]string, 0)
	for _, rt := range richTexts {
		parts = append(parts, rt.PlainText)
	}
	return strings.Join(parts, "")
}

// matchesPropertyFilters はページがプロパティフィルタに一致するかチェックします
func (f *NotionFetcher) matchesPropertyFilters(page *notionapi.Page, filters map[string]string) bool {
	if len(filters) == 0 {
		return true // フィルタなしの場合は全て通過
	}

	for propName, expectedValue := range filters {
		actualValue := f.getPropertyValue(page, propName)
		if actualValue != expectedValue {
			return false // フィルタ条件に合わない
		}
	}

	return true
}

// getPropertyValue はプロパティ値を文字列として取得します
func (f *NotionFetcher) getPropertyValue(page *notionapi.Page, propName string) string {
	prop, exists := page.Properties[propName]
	if !exists {
		return ""
	}

	return f.formatPropertyValue(prop)
}

// fetchPageContent はページの本文（ブロック）を取得します
func (f *NotionFetcher) fetchPageContent(ctx context.Context, pageID notionapi.PageID) string {
	// ページのブロック（本文）を取得
	blocks, err := f.client.Block.GetChildren(ctx, notionapi.BlockID(pageID), nil)
	if err != nil {
		// エラーの場合は空文字列を返す（Best Effort）
		return ""
	}

	// ブロックからテキストを抽出（簡易版：最初の3つのブロックのみ）
	textParts := make([]string, 0)
	for i, block := range blocks.Results {
		if i >= 3 {
			break // 最初の3ブロックのみ
		}

		text := f.extractBlockText(block)
		if text != "" {
			textParts = append(textParts, text)
		}
	}

	return strings.Join(textParts, "\n")
}

// extractBlockText はブロックからテキストを抽出します
func (f *NotionFetcher) extractBlockText(block notionapi.Block) string {
	switch block.GetType() {
	case notionapi.BlockTypeParagraph:
		paragraphBlock := block.(*notionapi.ParagraphBlock)
		return extractRichText(paragraphBlock.Paragraph.RichText)
	case notionapi.BlockTypeHeading1:
		heading1Block := block.(*notionapi.Heading1Block)
		return extractRichText(heading1Block.Heading1.RichText)
	case notionapi.BlockTypeHeading2:
		heading2Block := block.(*notionapi.Heading2Block)
		return extractRichText(heading2Block.Heading2.RichText)
	case notionapi.BlockTypeHeading3:
		heading3Block := block.(*notionapi.Heading3Block)
		return extractRichText(heading3Block.Heading3.RichText)
	case notionapi.BlockTypeBulletedListItem:
		bulletBlock := block.(*notionapi.BulletedListItemBlock)
		return "• " + extractRichText(bulletBlock.BulletedListItem.RichText)
	case notionapi.BlockTypeNumberedListItem:
		numberedBlock := block.(*notionapi.NumberedListItemBlock)
		return extractRichText(numberedBlock.NumberedListItem.RichText)
	default:
		return ""
	}
}
