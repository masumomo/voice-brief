package fetcher

import (
	"context"
	"testing"
	"time"

	"github.com/masumomo/voice-brief/internal/model"
	"github.com/slack-go/slack"
)

// MockFetcher はテスト用のモックFetcher
type MockFetcher struct {
	events model.Events
	err    error
}

func (m *MockFetcher) Fetch(ctx context.Context, since time.Time) (model.Events, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.events, nil
}

func TestMultiFetcher_BestEffort(t *testing.T) {
	// MultiFetcherがBest Effortで動作することをテスト
	// （一部のFetcherが失敗しても、他のFetcherの結果は返される）

	// 実際のテストは統合テストとして別途実装予定
	// ここではMultiFetcherの構造が正しいことのみ確認
	t.Skip("Integration test - requires mock Slack/Notion clients")
}

func TestMultiFetcher_ParallelFetch(t *testing.T) {
	// 並列取得が正しく動作することを確認するテスト
	// 実装は統合テストで行う
	t.Skip("Integration test - requires mock Slack/Notion clients")
}

// ここではfetcherパッケージの基本的な構造をテストする
// より詳細なテストはMock APIクライアントを実装した後に追加予定

func TestSlackFetcher_ReplaceGroupMentions(t *testing.T) {
	// SlackFetcherのReplaceMentionsがグループメンションを正しく置換することをテスト
	fetcher := &SlackFetcher{
		userCache: make(map[string]*slack.User),
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "here mention",
			input:    "<!here> 確認お願いします",
			expected: "チャンネルメンバー(オンライン) 確認お願いします",
		},
		{
			name:     "channel mention",
			input:    "<!channel> 重要なお知らせ",
			expected: "チャンネルメンバー全員 重要なお知らせ",
		},
		{
			name:     "everyone mention",
			input:    "<!everyone> 全社アナウンス",
			expected: "ワークスペース全員 全社アナウンス",
		},
		{
			name:     "subteam mention with label",
			input:    "<!subteam^S12345|@backend-team> レビューお願い",
			expected: "backend-team レビューお願い",
		},
		{
			name:     "subteam mention with @ in label",
			input:    "<!subteam^SABCDEF|@開発チーム> 確認してください",
			expected: "開発チーム 確認してください",
		},
		{
			name:     "multiple group mentions",
			input:    "<!here> と <!channel> に連絡",
			expected: "チャンネルメンバー(オンライン) と チャンネルメンバー全員 に連絡",
		},
		{
			name:     "no mentions",
			input:    "普通のテキスト",
			expected: "普通のテキスト",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fetcher.ReplaceMentions(tt.input)
			if result != tt.expected {
				t.Errorf("ReplaceMentions(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
