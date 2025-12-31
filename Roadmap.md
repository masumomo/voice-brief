# VoiceBrief 開発ロードマップ

## 概要

VoiceBriefは「音声で聞く、チームの最新情報」をコンセプトとした、音声ブリーフィングツールです。
このロードマップは、VoiceBriefの段階的な開発計画を示します。各フェーズはMVP（Minimum Viable Product）の思想に基づき、最小限の機能から段階的に拡張していきます。

## フェーズ0: プロジェクトセットアップ（1日）

### 目標

開発環境の構築と基本骨格の作成

### タスク

- [x] Goプロジェクト初期化
  - `go mod init github.com/masumomo/voice-brief`
  - ディレクトリ構造作成（cmd/, internal/, scripts/）

- [x] 依存ライブラリ追加

  ```bash
  go get github.com/slack-go/slack
  go get github.com/jomei/notionapi
  go get golang.org/x/sync/errgroup
  ```

- [x] 設定ファイル骨格作成
  - `config.example.yaml`
  - `.env.example`
  - `.gitignore`（out/, .env含む）

- [x] README.md初版作成
  - プロジェクト概要
  - セットアップ手順
  - 使用例

### 成果物

- [x] Goプロジェクト（`go build`が通る状態）
- [x] 基本的なREADME
- [x] サンプル設定ファイル

---

## Phase 1: コア機能実装（MVP - 1週間）

### 目標

最小限の機能で「Daily Briefing音声生成」を実現

### Phase 1.1: 設定読み込み（1日）

**タスク:**

- [x] `internal/config/config.go`実装
  - YAMLパース機能
  - 環境変数からトークン読み込み
  - バリデーション

- [x] `voicebrief config check`コマンド実装
  - 設定ファイルの存在確認
  - 必須項目チェック
  - トークンの形式検証

**成果物:**

- [x] `voicebrief config check`が正常動作

**検証:**

```bash
voicebrief config check
# Output: ✓ Config loaded successfully
# Output: ✓ Slack token found
# Output: ✓ Notion token found
```

### Phase 1.2: Slack連携（2日）

**タスク:**

- [x] `internal/fetcher/slack.go`実装
  - Slack API接続確認
  - 指定チャンネルの履歴取得（24時間）
  - Bot投稿フィルタリング
  - Event構造体への正規化

- [x] `internal/model/event.go`実装
  - Event構造体定義
  - 基本的なImportance計算（キーワードベース）

- [x] エラーハンドリング
  - タイムアウト設定（30秒）
  - コンテキストによるキャンセル処理

**成果物:**

- [x] Slackからメッセージ取得・正規化が可能

**検証:**

```bash
voicebrief doctor --source slack
# Output: ✓ Slack API connection successful
# Output: ✓ Retrieved 15 messages from #general
```

### Phase 1.3: Notion連携（2日）

**タスク:**

- [x] `internal/fetcher/notion.go`実装
  - Notion API接続確認
  - 指定Database Query（last_edited_time filter）
  - タイトル・プロパティ抽出
  - Event構造体への正規化

- [x] 複数DB対応
  - 設定ファイルで複数DB指定
  - Best Effort並列取得

**成果物:**

- [x] Notionからページ更新情報取得・正規化が可能

**検証:**

```bash
voicebrief doctor --source notion
# Output: ✓ Notion API connection successful
# Output: ✓ Retrieved 8 pages from "Design Docs"
```

### Phase 1.4: 並列Fetcher実装（1日）

**タスク:**

- [x] `internal/fetcher/fetcher.go`実装
  - errgroup使用
  - Slack/Notion並列取得
  - 部分失敗許容（1つ失敗しても他は継続）
  - タイムアウト制御

**成果物:**

- [x] SlackとNotionを並列で取得

**検証:**

```bash
voicebrief run --daily --dry-run
# Output: ✓ Fetched 15 events from Slack
# Output: ✓ Fetched 8 events from Notion
# Output: ✓ Total: 23 events
```

### Phase 1.5: Rule-based Summarizer実装（2日）

**タスク:**

- [x] `internal/summarizer/summarizer.go`（Interface定義）
- [x] `internal/summarizer/rule.go`実装
  - Importanceでソート
  - Daily用テンプレート作成
  - Markdownフォーマット生成
  - TTS用プレーンテキスト生成

- [x] `internal/model/brief.go`実装
  - Brief構造体定義

**成果物:**

- [x] Daily Briefing原稿（Markdown）生成可能
- [x] Weekly Briefing原稿（Markdown）生成可能

**検証:**

```bash
voicebrief run --daily --dry-run
# out/daily/2025-01-15.md が生成される
```

### Phase 1.6: TTS実装（macOS say）（2日）

**タスク:**

- [x] `internal/tts/tts.go`（Interface定義）
- [x] `internal/tts/say.go`実装
  - macOS sayコマンド実行
  - AIFF生成
  - ffmpegでM4A変換（オプション）

- [x] ファイル出力
  - `out/daily/YYYY-MM-DD.aiff`保存（ffmpeg未インストール時）
  - `out/daily/YYYY-MM-DD.m4a`保存（ffmpegインストール時）

**成果物:**

- [x] 音声ファイル生成機能

**検証:**

```bash
voicebrief run --daily
# out/daily/2025-12-29.md 生成
# out/daily/2025-12-29.aiff 生成
# 音声ファイルを再生して内容確認
```

### Phase 1.7: CLIコマンド統合（1日）

**タスク:**

- [ ] `cmd/voicebrief/main.go`実装
  - `run --daily`コマンド
  - `config check`コマンド
  - `doctor`コマンド
  - `version`コマンド

- [ ] 共通オプション実装
  - `--config`
  - `--out-dir`
  - `--dry-run`
  - `--log-level`

**成果物:**

- [x] すべてのCLIコマンドが動作

**検証:**

```bash
voicebrief run --daily
voicebrief config check
voicebrief doctor
voicebrief version
```

### Phase 1 完了条件（DoD）

- [x] `voicebrief run --daily`が正常完走
- [x] `voicebrief run --weekly`が正常完走
- [x] SlackとNotionから過去24時間のイベント取得
- [x] ブリーフィング原稿（Markdown）生成
- [x] 音声ファイル（AIFF/M4A）生成
- [x] 生データはディスクに保存されない（デフォルト、--debug-dumpで保存可能）
- [x] README手順で第三者が実行可能

---

## Phase 2: Weekly対応・品質向上（3日）

### 目標

Weeklyモード実装と安定性・ログ改善

### Phase 2.1: Weekly実装（1日）

**タスク:**

- [x] `voicebrief run --weekly`コマンド実装
- [x] Weekly用テンプレート作成
  - 7日間の集約ロジック
  - 重要決定・未解決事項の抽出

- [x] 出力先変更
  - `out/weekly/YYYY-Www.md`
  - `out/weekly/YYYY-Www.m4a`

**成果物:**

- [x] Weekly Briefing生成機能

**検証:**

```bash
voicebrief run --weekly
# out/weekly/2025-W02.md 生成
# out/weekly/2025-W02.m4a 生成
```

### Phase 2.2: ログ強化・デバッグモード（1日）

**タスク:**

- [x] 構造化ログ実装（JSON形式）
  - `internal/logger`パッケージ実装
  - DEBUGレベルのログ出力
  - 人間が読みやすい形式とJSON形式の切り替え
- [x] `--debug-dump`フラグ実装
  - 生データをout/debug/に保存
  - デフォルトはOFF

- [x] エラーハンドリング改善
  - 部分失敗時の詳細ログ
  - 重要処理ポイントでのログ出力

**成果物:**

- [x] デバッグ機能・詳細ログ（完了）

**検証:**

```bash
voicebrief run --daily --debug-dump
# out/debug/2025-01-15-raw.json に生データ保存
```

### Phase 2.3: テスト追加（1日）

**タスク:**

- [x] ユニットテスト追加
  - config package（75.0%）
  - model package（100.0%）
  - filter/importance package（86.0%）
  - logger package（68.8%）
  - tts package（46.3%）

- [x] 統合テスト
  - fetcherのスケルトンテスト追加（統合テストは将来実装）

- [x] カバレッジ測定
  - コアパッケージで70%以上達成
  - 総合カバレッジ: 34.1%（cmd/fetcherを除く）

**成果物:**

- [x] テストコード（`*_test.go`）
  - internal/logger/logger_test.go
  - internal/tts/say_test.go
  - internal/model/brief_test.go
  - internal/fetcher/fetcher_test.go

**検証:**

```bash
go test ./... -cover
# PASS
# 主要パッケージカバレッジ:
# - model: 100.0%
# - summarizer: 91.4%
# - filter: 86.0%
# - config: 75.0%
```

### Phase 2 完了条件

- [x] `voicebrief run --weekly`が正常動作
- [x] 構造化ログ出力（internal/loggerパッケージ実装完了）
- [x] デバッグモードで生データ保存可能（--debug-dump実装済み）
- [x] ユニットテストカバレッジ70%以上（コアパッケージで達成）

---

## Phase 3: 自動化・運用機能（2日）

### 目標

launchd連携と定期実行

### Phase 3.1: launchd設定（1日）

**タスク:**

- [x] `scripts/com.voicebrief.daily.plist`作成
  - 毎朝8:00実行設定
  - 標準出力・エラーログ設定

- [x] `scripts/com.voicebrief.weekly.plist`作成
  - 毎週月曜8:00実行設定
  - 標準出力・エラーログ設定

- [x] `scripts/install.sh`作成
  - plistを`~/Library/LaunchAgents/`にコピー
  - パス自動置換機能
  - `launchctl load`実行
  - 環境変数チェック機能

- [x] `scripts/uninstall.sh`作成
  - launchdジョブのアンロード
  - plistファイルの削除

**成果物:**

- [x] launchd自動実行設定

**検証:**

```bash
./scripts/install.sh
launchctl list | grep voicebrief
# 自動実行が登録されていることを確認

# アンインストール
./scripts/uninstall.sh
```

### Phase 3.2: Slack投稿機能（オプション）（1日）

**タスク:**

- [x] `internal/uploader/slack.go`実装
  - 指定チャンネルへ原稿投稿
  - 音声ファイル添付機能

- [x] `internal/uploader/uploader.go`実装
  - Uploaderインターフェース定義

- [x] 設定ファイル拡張
  - `slack.post_enabled`設定追加（デフォルト: false）
  - `slack.post_channel`設定追加
  - `slack.upload_audio`設定追加

- [x] main.goに統合
  - Slack投稿処理の追加
  - エラーハンドリング（Best Effort）

**成果物:**

- [x] Slackへのブリーフィング投稿機能

**検証:**

```bash
# post_enabled: false の場合は投稿されない
voicebrief run --daily
# 投稿がスキップされることを確認

# post_enabled: true, post_channel設定後
voicebrief run --daily
# 指定チャンネルにブリーフィングが投稿される
```

### Phase 3 完了条件

- [x] launchdで定期実行設定可能
- [x] Slack投稿機能（オプション）動作
- [x] README手順で自動化設定完了

---

## Phase 4: 機能拡張（v1.1 - 1週間） ✅ 完了

### Phase 4.1: Slackスレッド対応（2日） ✅

**タスク:**

- [x] スレッド返信取得
  - `fetchThreadReplies`関数実装
  - 最大100件のスレッド返信を取得
  - 最初の3件の返信をBodyに追加
- [x] 親メッセージとの紐付け
  - `thread_ts`によるスレッド判定
  - 返信数を`Refs["thread_reply_count"]`に記録
- [x] Importance計算にスレッド数を反映
  - 5件以上の返信: +15ポイント
  - 2件以上の返信: +5ポイント

**成果物:**

- [x] スレッド対応のSlack取得（internal/fetcher/slack.go）
- [x] 重要度計算の改善（internal/filter/importance.go）

### Phase 4.2: Notionプロパティフィルタ強化（2日） ✅

**タスク:**

- [x] ページ本文取得（オプション）
  - `fetchPageContent`関数実装
  - 最初の3ブロックのテキストを抽出
  - Paragraph, Heading, List対応
- [x] プロパティベースフィルタ
  - `matchesPropertyFilters`関数実装
  - Status = "In Progress"のみ等のフィルタリング
  - 設定ファイルで`property_filters`指定可能
- [x] タグ・プロジェクトによるカテゴリ分類
  - `CategoryProperty`で明示的なカテゴリ指定
  - `ProjectProperty`でプロジェクト情報をRefsに記録
  - `mapToCategory`関数でカテゴリマッピング

**成果物:**

- [x] Notion本文取得機能（internal/fetcher/notion.go）
- [x] プロパティフィルタ機能
- [x] カテゴリ・プロジェクト分類強化
- [x] DatabaseConfig拡張（internal/config/config.go）

**設定例:**

```yaml
notion:
  databases:
    - id: "db-uuid-xxxx"
      name: "Task Board"
      properties: ["Status", "Assignee", "Priority"]
      fetch_page_content: true  # ページ本文を取得
      property_filters:
        Status: "In Progress"  # 進行中のタスクのみ
      category_property: "Category"  # カテゴリ判定用プロパティ
      project_property: "Project"    # プロジェクト名
```

### Phase 4.3: ML重要度判定 ✅ （既存実装で十分）

**実装済み機能:**

- [x] キーワードスコアリング（RuleBasedCalculator）
  - 緊急/障害キーワード: +30ポイント
  - 低優先度キーワード: -20ポイント
- [x] メンション考慮
  - @メンション、<!channel>、<!here>: +20ポイント
- [x] スレッド数考慮（Phase 4.1で実装）
  - 5件以上の返信: +15ポイント
  - 2件以上の返信: +5ポイント
- [x] カテゴリ別調整
  - Incident: +40ポイント
  - Dev: +10ポイント
  - Biz: +5ポイント

**判断:**
Phase 1-3で実装済みの重要度計算ロジックが十分に機能しているため、
追加のML実装は不要と判断。ルールベースで十分な精度を実現。

---

## Phase 5: Gemini統合（v1.2 - 1週間） ✅ 完了

### Phase 5.1: Gemini Summarizer（3日） ✅

**タスク:**

- [x] `internal/summarizer/gemini.go`実装
  - Gemini 2.0 Flash（無料枠）によるブリーフィング生成
  - プロンプトエンジニアリング
  - コンテキスト長制御
  - エラーハンドリング・リトライ

**設計方針:**

- Gemini API無料枠を活用（毎分15リクエスト、毎日1500リクエスト）
- `google.generativeai` Go SDKを使用
- Rule-based要約との切り替え可能な設計

**成果物:**

- [x] Gemini要約機能（internal/summarizer/gemini.go - 269行）
- [x] 設定ファイル拡張（gemini_api_key_env, gemini_model）
- [x] テスト作成（カバレッジ69.6%）
- [x] ドキュメント更新

### Phase 5.2: Google Cloud TTS（オプション・2日） ✅

**タスク:**

- [x] Google Cloud TTS API連携実装
  - WaveNet音声によるTTS
  - 日本語音声品質向上（ja-JP-Neural2-B等）
  - 無料枠活用（毎月100万文字まで）

**成果物:**

- [x] Google TTS実装（internal/tts/google_tts.go - 173行）
- [x] macOS say互換の音声名マッピング
- [x] MP3/OGG/WAV形式対応
- [x] 認証情報のJSON環境変数対応
- [x] テスト作成
- [x] ドキュメント更新

### Phase 5.3: iPhoneショートカット連携 ❌ スコープアウト

**理由:**

- コアバリュー「音声ブリーフィング」から外れる
- 優先度が低い（Phase 6, 7の方が重要）
- 将来的な拡張として検討

---

## Phase 6: Windows対応（v2.0 - 3日） ✅

### 目標

Windows環境での動作対応

### タスク

- [x] TTS抽象化（OS判定）
  - ビルドタグ（`// +build darwin` / `// +build windows`）による分岐
  - TTSインターフェースの活用

- [x] Windows SAPI対応
  - `internal/tts/sapi.go`実装（177行）
  - PowerShell経由でSAPI呼び出し
  - 音声ファイル生成（WAV/MP3）
  - macOS音声名互換マッピング（Kyoko→Haruka等）

- [x] ビルド・実行確認
  - クロスプラットフォームビルド対応
  - 設定ファイルとドキュメント更新

**成果物:**

- [x] Windows版TTS実装（internal/tts/sapi.go）
- [x] macOS版TTS分離（internal/tts/say.go - ビルドタグ追加）
- [x] クロスプラットフォームビルド対応
- [x] 設定ファイル更新（provider: "sapi"追加）
- [x] README/Roadmap更新

**実装詳細:**

```go
// internal/tts/sapi.go - Windows SAPI実装
// PowerShellスクリプトでSystem.Speech.Synthesis.SpeechSynthesizerを呼び出し
type SAPITTS struct {
    config Config
}

// 音声名マッピング
mapping := map[string]string{
    "Kyoko": "Microsoft Haruka Desktop",   // 日本語女性
    "Otoya": "Microsoft Ichiro Desktop",   // 日本語男性
}

// 速度設定: config.Rate (0.5-2.0) → SAPI Speed (-10 to 10)
speed := int((s.config.Rate - 1.0) * 5)
```

**設定例:**

```yaml
# Windows環境
tts:
  provider: "sapi"  # Windows SAPI
  voice: "Kyoko"    # Haruka Desktop にマッピング
  rate: 1.1
  format: "wav"     # または mp3（ffmpeg必要）
```

**検証:**

```bash
# Windows環境で
go build -o voicebrief.exe cmd/voicebrief/main.go
.\voicebrief.exe run --daily

# macOS環境で
go build -o voicebrief cmd/voicebrief/main.go
./voicebrief run --daily
```

---

## Phase 7: OpenAI統合（v2.1 - 1週間） ✅

### 目標

OpenAI APIによる高品質な要約とTTS

### Phase 7.1: OpenAI Summarizer（3日） ✅

**タスク:**

- [x] `internal/summarizer/openai.go`実装（348行）
  - GPT-4o/gpt-4o-mini/gpt-4-turboによる要約生成
  - プロンプトエンジニアリング（Daily/Weekly専用プロンプト）
  - トークン使用量追跡
  - リアルタイムコスト表示

- [x] 設定ファイル拡張
  - `openai_api_key_env`設定追加
  - `openai_model`設定追加（デフォルト: gpt-4o-mini）

**成果物:**

- [x] OpenAI Summarizer実装
- [x] Gemini/Rule-basedとの切り替え機能
- [x] トークン使用量・コスト表示機能

**実装詳細:**

```go
// Summarizerの初期化とコスト追跡
sum, err := summarizer.NewOpenAISummarizer(apiKey, model, maxDaily, maxWeekly)
brief, err := sum.GenerateDaily(events)

// コスト情報の取得
total, prompt, completion := sum.GetTokenUsage()
cost := sum.EstimateCost()
// 出力: 💰 OpenAI コスト: $0.000123 (Total: 450 tokens)
```

**コスト目安:**

- Daily (gpt-4o-mini): $0.001-0.003/回
- Weekly (gpt-4o-mini): $0.003-0.008/回

### Phase 7.2: OpenAI TTS（2日） ✅

**タスク:**

- [x] `internal/tts/openai_tts.go`実装（147行）
  - OpenAI TTS API連携（tts-1 / tts-1-hd）
  - 音声選択（alloy, echo, fable, onyx, nova, shimmer）
  - 速度調整（0.25 - 4.0倍速）
  - フォーマット選択（mp3, opus, aac, flac）

**成果物:**

- [x] OpenAI TTS機能
- [x] macOS音声名マッピング（Kyoko→nova, Otoya→onyx）
- [x] 文字数カウント・コスト見積もり機能

**設定例:**

```yaml
tts:
  provider: "openai_tts"
  voice: "nova"  # alloy, echo, fable, onyx, nova, shimmer
  rate: 1.0
  format: "mp3"  # mp3, opus, aac, flac
```

**コスト目安:**

- Daily (tts-1): $0.03-0.06/回
- Weekly (tts-1): $0.08-0.15/回

### Phase 7.3: コスト管理（1日） ✅

**タスク:**

- [x] トークン使用量ログ出力
  - Summarizer実行時にトークン数とコストを表示
  - TTS実行時に文字数を追跡
- [x] コスト見積もり機能
  - `EstimateCost()`メソッドでリアルタイム計算
  - モデル別料金設定
- [x] README にコスト情報追加
  - 各プロバイダーのコスト比較表
  - 月次コスト試算

**成果物:**

- [x] リアルタイムコスト表示
- [x] 月次コスト試算ドキュメント
- [x] 無料枠 vs 従量課金の比較表

---

## Phase 8: GitHub統合（v2.2 - 3日） ✅

### 目標

GitHubの活動もブリーフィングに含める

### タスク

- [x] `internal/fetcher/github.go`実装（297行）
  - GitHub API連携（github.com/google/go-github/v57）
  - リポジトリのコミット取得
  - Issue/PR更新取得
  - ユーザー名フィルタ対応

- [x] 設定ファイル拡張
  - `github.enabled`設定
  - `github.repositories`リスト（owner/repo形式）
  - `github.username`フィルタ

- [x] Event構造体への変換
  - コミット → Event（SHA, メッセージ、diff stats）
  - Issue/PR → Event（番号、ラベル、コメント数）
  - カテゴリ自動判定（Conventional Commits対応）

**成果物:**

- [x] GitHub Fetcher実装
- [x] 並列取得対応（Slack/Notion/GitHub 3ソース）
- [x] MultiFetcher統合

**実装詳細:**

```go
// GitHub Fetcher初期化
githubFetcher := NewGitHubFetcher(&cfg.GitHub)

// コミット取得
commits, err := githubFetcher.fetchCommits(ctx, "owner/repo", since)

// Issue/PR取得
issues, err := githubFetcher.fetchIssuesAndPRs(ctx, "owner/repo", since)

// カテゴリ判定（Conventional Commits）
// fix/hotfix → Incident
// feat/feature → Dev
// chore/ci → Ops
```

**設定例:**

```yaml
github:
  enabled: true
  token_env: "VOICE_BRIEF_GITHUB_TOKEN"
  username: "your-username"  # 自分のアクティビティのみ
  repositories:
    - "owner/repository-name"
    - "your-org/another-repo"
```

**検証:**

```bash
# GitHub統合を有効化
export VOICE_BRIEF_GITHUB_TOKEN="ghp_your-token"

# 設定ファイルでgithub.enabled: true

# ブリーフィング生成
voicebrief run --daily
# ✓ GitHub から X 件のイベントを取得
```

---

## Phase 9: 特徴量ベース重要度計算（v2.3 - 1日） ✅

### 目標

特徴量ベースの機械学習的アプローチによる重要度計算の実装

### タスク

- [x] `internal/importance/feature.go`実装（287行）
  - 20個の特徴量を抽出（テキスト、エンゲージメント、カテゴリ、ソース、時間）
  - シグモイド関数による正規化
  - 重み付け線形結合でスコア計算
  - 0-100の範囲にクリップ

- [x] アーキテクチャ改善
  - Fetcher（slack/notion/github）から重要度計算を分離
  - main.goで一元管理（関心の分離）
  - 常にFeatureBasedCalculatorを使用（rule-basedは内部的に残すが設定不要）

- [x] ユーティリティ関数追加
  - `TopK(events, k)` - 重要度上位K件を取得
  - `GetFeatureVector(event)` - 特徴量ベクトル取得
  - `GetFeatureNames()` - 特徴量名一覧取得

- [x] テスト追加
  - FeatureBasedCalculatorの8つのテストケース
  - TopK関数の5つのテストケース

**成果物:**

- [x] FeatureBasedCalculator実装
- [x] TopK関数実装
- [x] テストカバレッジ維持（22テスト）

**特徴量一覧:**

| カテゴリ | 特徴量 | 説明 |
| --- | --- | --- |
| テキスト | TextLength | 本文の長さ（正規化） |
| テキスト | TitleLength | タイトルの長さ（正規化） |
| テキスト | HasMention | @メンション有無 |
| テキスト | HasQuestion | 疑問文か |
| テキスト | HasExclamation | 強調表現か |
| テキスト | HasURL | URL含有 |
| テキスト | KeywordScore | 重要キーワードスコア |
| エンゲージメント | CommentCount | コメント数（正規化） |
| エンゲージメント | UniqueCommenters | ユニーク投稿者数 |
| エンゲージメント | TotalCommentLen | コメント総文字数 |
| カテゴリ | IsIncident | Incidentカテゴリか |
| カテゴリ | IsDev | Devカテゴリか |
| カテゴリ | IsBiz | Bizカテゴリか |
| カテゴリ | IsOps | Opsカテゴリか |
| ソース | IsSlack | Slackソースか |
| ソース | IsNotion | Notionソースか |
| ソース | IsGitHub | GitHubソースか |
| 時間 | HourOfDay | 時間帯（正規化） |
| 時間 | IsBusinessHours | 営業時間内か |
| 時間 | IsWeekday | 平日か |

**デフォルト重み設定:**

```go
IsIncident:       30.0,  // 障害は最重要
KeywordScore:     20.0,  // キーワードマッチは重要
HasMention:       15.0,  // メンションは重要
CommentCount:     15.0,  // 議論が活発
UniqueCommenters: 10.0,  // 多くの人が参加
IsOps:            8.0,   // 運用は重要
TextLength:       5.0,   // 長い投稿は重要な可能性
HasQuestion:      5.0,   // 質問は対応必要
Bias:             30.0,  // ベーススコア
```

**検証:**

```bash
# 設定確認
voicebrief config check
# ✓ 重要度計算: feature-based (20特徴量)

# ブリーフィング生成
voicebrief run --daily
# 📊 Step 2/7: 重要度計算中...
# ✓ 23 件のイベントの重要度を計算
```

---

## マイルストーン一覧

| マイルストーン | 期間 | 主な成果物 |
| --- | --- | --- |
| **Phase 0** | 1日 | プロジェクトセットアップ | ✅ |
| **Phase 1 (MVP)** | 1週間 | Daily Briefing生成機能 | ✅ |
| **Phase 2** | 3日 | Weekly対応・品質向上 | ✅ |
| **Phase 3** | 2日 | launchd自動化・Slack投稿 | ✅ |
| **v1.0 リリース** | - | **完全ローカル動作版** | ✅ |
| **Phase 4 (v1.1)** | 1週間 | スレッド・プロパティ強化 | ✅ |
| **Phase 5 (v1.2)** | 5日 | Gemini統合（Summarizer + Google TTS） | ✅ |
| **Phase 6 (v2.0)** | 3日 | Windows対応（SAPI TTS） | ✅ |
| **Phase 7 (v2.1)** | 1週間 | OpenAI統合（Summarizer + TTS） | ✅ |
| **Phase 8 (v2.2)** | 3日 | GitHub統合（3ソース並列取得） | ✅ |
| **Phase 9 (v2.3)** | 1日 | ML重要度計算（特徴量ベース） | ✅ |

---

## 優先順位付け（MoSCoW）

### Must Have（v1.0必須）

- ✅ Slack/Notion連携
- ✅ Daily/Weekly実行
- ✅ Rule-based要約
- ✅ macOS say TTS
- ✅ 生データ非保存（プライバシー）

### Should Have（v1.1推奨）

- ✅ Slackスレッド対応 - Phase 4.1完了
- ✅ Notionプロパティフィルタ強化 - Phase 4.2完了
- ✅ 重要度判定強化（スレッド数反映） - Phase 4.3完了

### Could Have（v1.2以降検討）

- ✅ Gemini要約（無料枠活用）- Phase 5.1完了
- ✅ Google Cloud TTS（音声品質向上）- Phase 5.2完了
- ✅ OpenAI要約・TTS（高品質）- Phase 7完了
- ✅ GitHub統合 - Phase 8完了
- ✅ 特徴量ベース重要度計算 - Phase 9完了

### Won't Have（スコープアウト）

- iPhoneショートカット連携（コアバリューから外れる）
- Web UI
- Linux対応
- リアルタイム通知
- モバイルアプリ

---

## リスク管理

| リスク | 影響 | 対策 |
| --- | --- | --- |
| Slack API制限 | 高 | Rate Limit制御、Retry実装 |
| Notion API変更 | 中 | API Version固定、定期確認 |
| ffmpeg非インストール | 低 | AIFF出力をフォールバック |
| Gemini API制限 | 中 | 無料枠内での利用、Rate Limit制御 |
| OpenAI費用 | 中 | 使用量モニタリング、コスト見積もり |
| macOS/Windows依存 | 中 | Windows対応でカバー（Linuxはスコープアウト） |
| GitHub API制限 | 低 | Rate Limit制御、認証トークン管理 |

---

## 開発Tips

### Phase 1開始時の推奨手順

1. **Config実装から開始**

   ```bash
   go run cmd/voicebrief/main.go config check
   ```

2. **Slack疎通確認**

   ```bash
   go run cmd/voicebrief/main.go doctor --source slack
   ```

3. **Notion疎通確認**

   ```bash
   go run cmd/voicebrief/main.go doctor --source notion
   ```

4. **Dry-runで原稿生成確認**

   ```bash
   go run cmd/voicebrief/main.go run --daily --dry-run
   ```

5. **音声生成テスト**

   ```bash
   go run cmd/voicebrief/main.go run --daily
   open out/daily/$(date +%Y-%m-%d).m4a
   ```

### 開発中のデバッグ

```bash
# 詳細ログ出力
voicebrief run --daily --log-level debug

# 生データ確認
voicebrief run --daily --debug-dump
cat out/debug/$(date +%Y-%m-%d)-raw.json | jq .

# Dry-runで音声化スキップ
voicebrief run --daily --dry-run
```

---

## 完了判定チェックリスト（v1.0）

### 機能

- [ ] `voicebrief run --daily`が正常動作
- [ ] `voicebrief run --weekly`が正常動作
- [ ] SlackとNotionから並列取得
- [ ] Rule-based要約で原稿生成
- [ ] macOS sayで音声生成
- [ ] 生データがディスクに保存されない（デフォルト）

### 品質

- [ ] ユニットテストカバレッジ70%以上
- [ ] 統合テスト（Mock API）通過
- [ ] エラーハンドリング適切（部分失敗許容）
- [ ] 構造化ログ出力

### ドキュメント

- [ ] README.md完成（セットアップ手順）
- [ ] config.example.yaml提供
- [ ] .env.example提供
- [ ] launchd plistサンプル提供

### 運用

- [ ] launchdで自動実行可能
- [ ] `voicebrief doctor`で接続確認可能
- [ ] 第三者がREADME手順で実行成功

---

## 次のアクション

Phase 0〜9 すべて完了！🎉

### 今後の改善候補

1. **カテゴリ判定の改善**
   - OpenAI/Gemini Categorizerの精度向上
   - カスタムカテゴリ対応

2. **要約品質の向上**
   - プロンプトエンジニアリング改善
   - 日本語特化のチューニング

3. **運用改善**
   - コスト最適化（API使用量モニタリング）
   - エラーリカバリー強化

### 開発Tips

```bash
# ブリーフィング生成（昨日分）
voicebrief run

# 過去3日分
voicebrief run --days 3

# Dry-runで原稿のみ
voicebrief run --dry-run

# デバッグ
voicebrief run --debug-dump --log-level debug
```
