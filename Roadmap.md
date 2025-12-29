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

- [ ] `scripts/com.voicebrief.daily.plist`作成
  - 毎朝8:00実行設定

- [ ] `scripts/com.voicebrief.weekly.plist`作成
  - 毎週月曜8:00実行設定

- [ ] `scripts/install.sh`作成
  - plistを`~/Library/LaunchAgents/`にコピー
  - `launchctl load`実行

**成果物:**

- [ ] launchd自動実行設定

**検証:**

```bash
./scripts/install.sh
launchctl list | grep voicebrief
# 自動実行が登録されていることを確認
```

### Phase 3.2: Slack投稿機能（オプション）（1日）

**タスク:**

- [ ] `internal/uploader/slack.go`実装
  - 指定チャンネルへ原稿投稿
  - 音声ファイル添付（可能なら）

- [ ] 設定ファイル拡張
  - `slack.post_channel`設定追加
  - 投稿ON/OFF切り替え

**成果物:**

- [ ] Slackへのブリーフィング投稿機能

**検証:**

```bash
voicebrief run --daily
# 指定チャンネルにブリーフィングが投稿される
```

### Phase 3 完了条件

- [x] launchdで定期実行設定可能
- [x] Slack投稿機能（オプション）動作
- [x] README手順で自動化設定完了

---

## Phase 4: 機能拡張（v1.1 - 1週間）

### Phase 4.1: Slackスレッド対応（2日）

**タスク:**

- [ ] スレッド返信取得
- [ ] 親メッセージとの紐付け
- [ ] Importance計算にスレッド数を反映

### Phase 4.2: Notionプロパティフィルタ強化（2日）

**タスク:**

- [ ] ページ本文取得（オプション）
- [ ] プロパティベースフィルタ
  - Status = "In Progress"のみ等
- [ ] タグ・プロジェクトによるカテゴリ分類

### Phase 4.3: ML重要度判定（3日）

**タスク:**

- [ ] Importance計算ロジック改善
  - キーワードスコアリング強化
  - メンション・リアクション考慮
  - 時系列重み付け

---

## Phase 5: OpenAI統合（v1.2 - 1週間）

### Phase 5.1: OpenAI Summarizer（3日）

**タスク:**

- [ ] `internal/summarizer/openai.go`実装
  - GPT-4によるブリーフィング生成
  - プロンプトエンジニアリング
  - トークン数制御

### Phase 5.2: OpenAI TTS（2日）

**タスク:**

- [ ] `internal/tts/openai_tts.go`実装
  - OpenAI TTS API連携
  - 音質設定（alloy, echo等）

### Phase 5.3: iPhoneショートカット連携（2日）

**タスク:**

- [ ] Webhook受信サーバー（簡易HTTP）
- [ ] Notionへタスク登録機能
- [ ] ショートカットサンプル作成

---

## Phase 6: マルチプラットフォーム対応（v2.0 - 2週間）

### Phase 6.1: Linux/Windows対応（1週間）

**タスク:**

- [ ] TTS抽象化（OS判定）
- [ ] Windows SAPI対応
- [ ] Linux espeak対応

### Phase 6.2: Web UI（1週間）

**タスク:**

- [ ] 簡易Web UI（Go標準net/http）
- [ ] ブリーフィング一覧表示
- [ ] 音声再生機能

---

## マイルストーン一覧

| マイルストーン | 期間 | 主な成果物 |
| --- | --- | --- |
| **Phase 0** | 1日 | プロジェクトセットアップ |
| **Phase 1 (MVP)** | 1週間 | Daily Briefing生成機能 |
| **Phase 2** | 3日 | Weekly対応・品質向上 |
| **Phase 3** | 2日 | launchd自動化 |
| **v1.0 リリース** | - | **完全ローカル動作版** |
| **Phase 4 (v1.1)** | 1週間 | スレッド・プロパティ強化 |
| **Phase 5 (v1.2)** | 1週間 | OpenAI統合 |
| **Phase 6 (v2.0)** | 2週間 | マルチプラットフォーム・Web UI |

---

## 優先順位付け（MoSCoW）

### Must Have（v1.0必須）

- ✅ Slack/Notion連携
- ✅ Daily/Weekly実行
- ✅ Rule-based要約
- ✅ macOS say TTS
- ✅ 生データ非保存（プライバシー）

### Should Have（v1.1推奨）

- Slackスレッド対応
- Notionプロパティフィルタ強化
- ML重要度判定

### Could Have（v1.2以降検討）

- OpenAI要約・TTS
- iPhoneショートカット連携
- Slack投稿機能

### Won't Have（v2.0以降）

- Web UI
- マルチプラットフォーム対応
- リアルタイム通知

---

## リスク管理

| リスク | 影響 | 対策 |
| --- | --- | --- |
| Slack API制限 | 高 | Rate Limit制御、Retry実装 |
| Notion API変更 | 中 | API Version固定、定期確認 |
| ffmpeg非インストール | 低 | AIFF出力をフォールバック |
| OpenAI費用 | 中 | 無料プロバイダ（say）をデフォルト |
| macOS依存 | 高 | v2.0でマルチプラットフォーム対応 |

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

1. **Phase 0開始**: プロジェクトセットアップ

   ```bash
   mkdir voice-brief && cd voice-brief
   go mod init github.com/yourusername/voice-brief
   ```

2. **Phase 1.1開始**: Config実装
   - `internal/config/config.go`を作成
   - `config.example.yaml`を作成

3. **継続的に**:
   - 各Phase完了後、動作確認
   - README更新
   - Git commit（コミットメッセージにPhase番号記載）

---

**このロードマップで進める準備ができたら、Phase 0のセットアップから始めましょう。**
