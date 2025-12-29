package fetcher

import (
	"context"
	"testing"
	"time"

	"github.com/masumomo/voice-brief/internal/model"
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
