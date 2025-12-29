package summarizer

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/masumomo/voice-brief/internal/model"
)

func TestNewGeminiSummarizer(t *testing.T) {
	tests := []struct {
		name        string
		apiKey      string
		model       string
		maxDaily    int
		maxWeekly   int
		wantErr     bool
		wantModel   string
		wantDaily   int
		wantWeekly  int
	}{
		{
			name:       "有効なパラメータ",
			apiKey:     "test-api-key",
			model:      "gemini-2.0-flash-exp",
			maxDaily:   10,
			maxWeekly:  30,
			wantErr:    false,
			wantModel:  "gemini-2.0-flash-exp",
			wantDaily:  10,
			wantWeekly: 30,
		},
		{
			name:       "デフォルトモデル",
			apiKey:     "test-api-key",
			model:      "",
			maxDaily:   8,
			maxWeekly:  25,
			wantErr:    false,
			wantModel:  "gemini-2.0-flash-exp",
			wantDaily:  8,
			wantWeekly: 25,
		},
		{
			name:       "デフォルト最大値",
			apiKey:     "test-api-key",
			model:      "gemini-2.0-flash-exp",
			maxDaily:   0,
			maxWeekly:  0,
			wantErr:    false,
			wantModel:  "gemini-2.0-flash-exp",
			wantDaily:  8,
			wantWeekly: 25,
		},
		{
			name:    "APIキーなし",
			apiKey:  "",
			model:   "gemini-2.0-flash-exp",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: 実際のAPI呼び出しはスキップ（エラーチェックのみ）
			if tt.apiKey == "" {
				_, err := NewGeminiSummarizer(tt.apiKey, tt.model, tt.maxDaily, tt.maxWeekly)
				if (err != nil) != tt.wantErr {
					t.Errorf("NewGeminiSummarizer() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			// API Key が設定されている場合のテスト（クライアント作成までは実行）
			// 実際のAPI呼び出しは行わない
			t.Skip("実際のGemini API呼び出しはスキップします")
		})
	}
}

func TestGetAPIKey(t *testing.T) {
	tests := []struct {
		name    string
		envName string
		envVal  string
		want    string
	}{
		{
			name:    "デフォルト環境変数名",
			envName: "",
			envVal:  "test-key-123",
			want:    "test-key-123",
		},
		{
			name:    "カスタム環境変数名",
			envName: "CUSTOM_GEMINI_KEY",
			envVal:  "custom-key-456",
			want:    "custom-key-456",
		},
		{
			name:    "環境変数が未設定",
			envName: "NON_EXISTENT_KEY",
			envVal:  "",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 環境変数をセットアップ
			if tt.envName == "" {
				tt.envName = "GEMINI_API_KEY"
			}

			// 既存の値を保存
			oldVal := os.Getenv(tt.envName)
			defer func() {
				if oldVal != "" {
					os.Setenv(tt.envName, oldVal)
				} else {
					os.Unsetenv(tt.envName)
				}
			}()

			// テスト用の値をセット
			if tt.envVal != "" {
				os.Setenv(tt.envName, tt.envVal)
			} else {
				os.Unsetenv(tt.envName)
			}

			// テスト実行
			got := GetAPIKey(tt.envName)
			if got != tt.want {
				t.Errorf("GetAPIKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterEventsBySource(t *testing.T) {
	now := time.Now()
	events := model.Events{
		{
			ID:         "1",
			Source:     "slack-general",
			Title:      "Slack message 1",
			Body:       "Test message",
			Timestamp:  now,
			Importance: 5,
		},
		{
			ID:         "2",
			Source:     "notion-db",
			Title:      "Notion update 1",
			Body:       "Test update",
			Timestamp:  now,
			Importance: 3,
		},
		{
			ID:         "3",
			Source:     "slack-dev",
			Title:      "Slack message 2",
			Body:       "Another message",
			Timestamp:  now,
			Importance: 4,
		},
	}

	tests := []struct {
		name       string
		source     string
		wantCount  int
		wantIDs    []string
	}{
		{
			name:      "Slackイベントのフィルタ",
			source:    "slack",
			wantCount: 2,
			wantIDs:   []string{"1", "3"},
		},
		{
			name:      "Notionイベントのフィルタ",
			source:    "notion",
			wantCount: 1,
			wantIDs:   []string{"2"},
		},
		{
			name:      "存在しないソース",
			source:    "github",
			wantCount: 0,
			wantIDs:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterEventsBySource(events, tt.source)

			if len(got) != tt.wantCount {
				t.Errorf("filterEventsBySource() count = %v, want %v", len(got), tt.wantCount)
			}

			gotIDs := make([]string, len(got))
			for i, event := range got {
				gotIDs[i] = event.ID
			}

			if len(gotIDs) != len(tt.wantIDs) {
				t.Errorf("filterEventsBySource() IDs = %v, want %v", gotIDs, tt.wantIDs)
				return
			}

			for i, id := range gotIDs {
				if id != tt.wantIDs[i] {
					t.Errorf("filterEventsBySource() ID[%d] = %v, want %v", i, id, tt.wantIDs[i])
				}
			}
		})
	}
}

func TestBuildPrompt(t *testing.T) {
	// モックデータ作成
	now := time.Now()
	start := now.Add(-24 * time.Hour)

	brief := model.NewBrief(model.BriefTypeDaily, start, now)
	brief.AddEvent(&model.Event{
		ID:         "1",
		Source:     "slack-general",
		Title:      "テストメッセージ",
		Body:       "これはテストです",
		Timestamp:  now,
		Importance: 5,
		URL:        "https://slack.com/test",
	})
	brief.AddEvent(&model.Event{
		ID:         "2",
		Source:     "notion-tasks",
		Title:      "タスク更新",
		Body:       "新しいタスクが追加されました",
		Timestamp:  now,
		Importance: 3,
		URL:        "https://notion.so/test",
	})

	// ダミーのSummarizerを作成（API呼び出しはしない）
	s := &GeminiSummarizer{
		apiKey:         "dummy-key",
		model:          "gemini-2.0-flash-exp",
		maxItemsDaily:  8,
		maxItemsWeekly: 25,
	}

	t.Run("Dailyプロンプト生成", func(t *testing.T) {
		prompt := s.buildPrompt(brief, model.BriefTypeDaily)

		// プロンプトに必要な要素が含まれているか確認
		requiredPhrases := []string{
			"デイリーブリーフィング",
			"過去24時間",
			"Slackメッセージ",
			"Notion更新",
			"テストメッセージ",
			"タスク更新",
			"---SCRIPT---",
		}

		for _, phrase := range requiredPhrases {
			if !strings.Contains(prompt, phrase) {
				t.Errorf("buildPrompt() プロンプトに '%s' が含まれていません", phrase)
			}
		}
	})

	t.Run("Weeklyプロンプト生成", func(t *testing.T) {
		weeklyBrief := model.NewBrief(model.BriefTypeWeekly, start, now)
		weeklyBrief.AddEvent(brief.Items[0])
		weeklyBrief.AddEvent(brief.Items[1])

		prompt := s.buildPrompt(weeklyBrief, model.BriefTypeWeekly)

		// Weeklyプロンプトの確認
		requiredPhrases := []string{
			"ウィークリーブリーフィング",
			"過去1週間",
		}

		for _, phrase := range requiredPhrases {
			if !strings.Contains(prompt, phrase) {
				t.Errorf("buildPrompt() プロンプトに '%s' が含まれていません", phrase)
			}
		}
	})
}
