package fetcher

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jomei/notionapi"
	"github.com/masumomo/voice-brief/internal/config"
	"github.com/masumomo/voice-brief/internal/model"
)

// NotionFetcher はNotionからイベントを取得します
type NotionFetcher struct {
	client *notionapi.Client
	config *config.NotionConfig
}

// NewNotionFetcher は新しいNotionFetcherを作成します
func NewNotionFetcher(cfg *config.NotionConfig) *NotionFetcher {
	return &NotionFetcher{
		client: notionapi.NewClient(notionapi.Token(cfg.Token)),
		config: cfg,
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

	// 新規/更新の判定
	createdTime := time.Time(page.CreatedTime)
	lastEditedTime := time.Time(page.LastEditedTime)
	isNewPage := lastEditedTime.Sub(createdTime) < 5*time.Minute // 作成から5分以内なら新規

	if isNewPage {
		event.Refs["change_type"] = "新規作成"
	} else {
		event.Refs["change_type"] = "更新"
	}
	event.Refs["created_at"] = createdTime.Format("2006-01-02 15:04:05")
	event.Refs["updated_at"] = lastEditedTime.Format("2006-01-02 15:04:05")

	// 全プロパティを詳細に取得
	allProperties := f.extractAllProperties(page)
	event.Refs["properties"] = allProperties

	// プロパティ情報を本文に含める（指定されたプロパティのみ）
	event.Body = f.extractProperties(page, database.Properties)

	// ページ本文取得（max_content_blocks > 0 の場合）
	maxBlocks := database.GetMaxContentBlocks()
	if maxBlocks > 0 {
		pageContent := f.fetchPageContent(ctx, notionapi.PageID(page.ID), maxBlocks)
		if pageContent != "" {
			event.Body = event.Body + "\n\n---\n\n" + pageContent
		}
	}

	// 最終編集者
	if page.LastEditedBy.ID != "" {
		event.Author = string(page.LastEditedBy.ID)
		// ユーザー名も取得試行
		if page.LastEditedBy.Name != "" {
			event.Author = page.LastEditedBy.Name
		}
	} else {
		event.Author = "Unknown"
	}

	// Refs に追加情報を格納
	event.Refs["database_id"] = database.ID
	event.Refs["database_name"] = database.Name
	event.Refs["page_id"] = string(page.ID)

	// カテゴリはMultiFetcherのCategorizerで判定される

	// プロジェクト情報の追加
	if database.ProjectProperty != "" {
		if project := f.getPropertyValue(page, database.ProjectProperty); project != "" {
			event.Refs["project"] = project
		}
	}

	return event
}

// extractAllProperties は全プロパティを文字列として抽出します
func (f *NotionFetcher) extractAllProperties(page *notionapi.Page) string {
	var parts []string

	for propName, prop := range page.Properties {
		value := f.formatPropertyValue(prop)
		if value != "" {
			parts = append(parts, fmt.Sprintf("%s: %s", propName, value))
		}
	}

	return strings.Join(parts, " | ")
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
func (f *NotionFetcher) fetchPageContent(ctx context.Context, pageID notionapi.PageID, maxBlocks int) string {
	// ページのブロック（本文）を取得
	blocks, err := f.client.Block.GetChildren(ctx, notionapi.BlockID(pageID), nil)
	if err != nil {
		// エラーの場合は空文字列を返す（Best Effort）
		return ""
	}

	// ブロックからテキストを抽出
	textParts := make([]string, 0)
	count := 0
	for _, block := range blocks.Results {
		if count >= maxBlocks {
			break
		}

		text := f.extractBlockText(block)
		if text != "" {
			textParts = append(textParts, text)
			count++
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
		return "# " + extractRichText(heading1Block.Heading1.RichText)
	case notionapi.BlockTypeHeading2:
		heading2Block := block.(*notionapi.Heading2Block)
		return "## " + extractRichText(heading2Block.Heading2.RichText)
	case notionapi.BlockTypeHeading3:
		heading3Block := block.(*notionapi.Heading3Block)
		return "### " + extractRichText(heading3Block.Heading3.RichText)
	case notionapi.BlockTypeBulletedListItem:
		bulletBlock := block.(*notionapi.BulletedListItemBlock)
		return "• " + extractRichText(bulletBlock.BulletedListItem.RichText)
	case notionapi.BlockTypeNumberedListItem:
		numberedBlock := block.(*notionapi.NumberedListItemBlock)
		return "1. " + extractRichText(numberedBlock.NumberedListItem.RichText)
	case notionapi.BlockTypeToggle:
		toggleBlock := block.(*notionapi.ToggleBlock)
		return "▶ " + extractRichText(toggleBlock.Toggle.RichText)
	case notionapi.BlockTypeQuote:
		quoteBlock := block.(*notionapi.QuoteBlock)
		return "> " + extractRichText(quoteBlock.Quote.RichText)
	case notionapi.BlockTypeCallout:
		calloutBlock := block.(*notionapi.CalloutBlock)
		return "💡 " + extractRichText(calloutBlock.Callout.RichText)
	case notionapi.BlockTypeCode:
		codeBlock := block.(*notionapi.CodeBlock)
		return "```\n" + extractRichText(codeBlock.Code.RichText) + "\n```"
	case notionapi.BlockTypeToDo:
		todoBlock := block.(*notionapi.ToDoBlock)
		checkbox := "☐"
		if todoBlock.ToDo.Checked {
			checkbox = "☑"
		}
		return checkbox + " " + extractRichText(todoBlock.ToDo.RichText)
	case notionapi.BlockTypeDivider:
		return "---"
	default:
		return ""
	}
}
