# VoiceBrief

育児中エンジニアのための完全音声キャッチアップツール

## 概要

VoiceBriefは、SlackとNotionの更新情報を自動的に収集し、音声ブリーフィングとして提供するCLIツールです。
ハンズフリーで業務の最新情報をキャッチアップできるため、育児や家事をしながらでも効率的に情報収集が可能です。

### 主な特徴

- **完全ローカル動作**: 生データをディスクに保存せず、プライバシーを保護
- **音声でキャッチアップ**: macOS標準TTSで音声ファイルを生成
- **並列高速取得**: SlackとNotionから同時並行でデータを取得
- **重要度フィルタリング**: ノイズを除外し、重要な情報のみを抽出
- **定期自動実行**: launchdで毎朝自動的にブリーフィング生成

## 必要環境

- macOS 12.0以降
- Go 1.21以降
- ffmpeg（オプション - 音声形式変換用）

```bash
brew install ffmpeg  # オプション
```

## セットアップ

### 1. リポジトリのクローン

```bash
git clone https://github.com/masumomo/voice-brief.git
cd voice-brief
```

### 2. 依存パッケージのインストール

```bash
go mod download
```

**または、Makefileを使って一括セットアップ:**

```bash
make dev
```

このコマンドで以下が自動実行されます:
- 設定ファイルのコピー（`config.yaml`, `.env`）
- 出力ディレクトリの作成
- 依存パッケージのダウンロード

### 3. 設定ファイルの作成

```bash
cp config.example.yaml config.yaml
cp .env.example .env
```

または `make dev` で自動作成されます。

### 4. トークンの取得と設定

#### Slack Bot Token

1. https://api.slack.com/apps にアクセス
2. 「Create New App」→「From scratch」を選択
3. アプリ名とワークスペースを指定して作成
4. 「OAuth & Permissions」で以下のBot Token Scopesを追加:
   - `channels:history`
   - `groups:history`（プライベートチャンネル監視時）
   - `chat:write`（Slack投稿機能使用時）
   - `files:write`（音声ファイル添付時）
5. 「Install to Workspace」でインストール
6. `Bot User OAuth Token`（`xoxb-`で始まる）をコピー
7. `.env`ファイルに設定:

```bash
VOICE_BRIEF_SLACK_TOKEN="xoxb-your-token-here"
```

#### Notion Integration Token

1. https://www.notion.so/my-integrations にアクセス
2. 「New integration」を作成
3. Integration名を設定し、関連ワークスペースを選択
4. 「Submit」で作成
5. 「Internal Integration Secret」をコピー
6. `.env`ファイルに設定:

```bash
VOICE_BRIEF_NOTION_TOKEN="secret_your-token-here"
```

7. 監視したいNotionデータベースで「・・・」→「Connections」→作成したIntegrationを追加

### 5. config.yamlの編集

```yaml
slack:
  channels:
    - id: "C01234567"  # 実際のChannel IDに変更
      name: "general"

notion:
  databases:
    - id: "db-uuid-xxxx"  # 実際のDatabase IDに変更
      name: "Task Board"
      properties: ["Status", "Assignee"]
```

**Channel IDの確認方法:**
- Slackでチャンネルを右クリック→「Copy link」
- URLの最後の部分（例: `C01234567`）がChannel ID

**Database IDの確認方法:**
- NotionでデータベースをFull pageで開く
- URLの`?v=`より前の部分（32文字のハイフン区切り文字列）

## ビルドと実行

### Makefile使用（推奨）

```bash
# ビルド
make build

# テスト実行
make test

# 設定確認
make config-check

# API接続確認
make doctor

# Daily Briefing生成
make run-daily

# Daily Briefing生成（Dry-run）
make run-daily-dry

# Weekly Briefing生成
make run-weekly

# 最新のDaily Briefingを表示
make cat-daily

# 最新のDaily Briefing音声を再生
make play-daily

# ヘルプ表示
make help
```

### 手動実行

#### ビルド

```bash
go build -o voicebrief cmd/voicebrief/main.go
```

#### 実行

```bash
# 設定確認
./voicebrief config check

# 接続テスト
./voicebrief doctor

# Daily Briefing生成（Dry-run）
./voicebrief run --daily --dry-run

# Daily Briefing生成（音声あり）
./voicebrief run --daily

# Weekly Briefing生成
./voicebrief run --weekly
```

### 生成されたファイルの確認

```bash
# Markdown確認
cat out/daily/$(date +%Y-%m-%d).md

# 音声再生
open out/daily/$(date +%Y-%m-%d).m4a
```

## CLI コマンド

### `voicebrief run`

ブリーフィングを生成します。

```bash
voicebrief run --daily   # Daily Briefing（過去24時間）
voicebrief run --weekly  # Weekly Briefing（過去7日間）
```

**オプション:**
- `--config PATH`: 設定ファイルパス（デフォルト: `./config.yaml`）
- `--out-dir PATH`: 出力ディレクトリ（デフォルト: `./out`）
- `--dry-run`: 音声生成・投稿をスキップ（原稿のみ生成）
- `--debug-dump`: 生データを`out/debug/`に保存（デバッグ用）
- `--log-level LEVEL`: ログレベル（`debug`, `info`, `warn`, `error`）

### `voicebrief config check`

設定ファイルと環境変数を検証します。

```bash
voicebrief config check
```

### `voicebrief doctor`

API接続と権限を確認します。

```bash
voicebrief doctor                # 全ソース確認
voicebrief doctor --source slack  # Slackのみ
voicebrief doctor --source notion # Notionのみ
```

### `voicebrief version`

バージョン情報を表示します。

```bash
voicebrief version
```

## 自動実行設定（launchd）

### インストール

```bash
./scripts/install.sh
```

このスクリプトは以下を実行します:
1. Daily実行用plistを`~/Library/LaunchAgents/`にコピー
2. Weekly実行用plistを`~/Library/LaunchAgents/`にコピー
3. launchdに登録（毎朝8:00実行）

### 確認

```bash
launchctl list | grep voicebrief
```

### 停止・削除

```bash
launchctl unload ~/Library/LaunchAgents/com.voicebrief.daily.plist
launchctl unload ~/Library/LaunchAgents/com.voicebrief.weekly.plist
rm ~/Library/LaunchAgents/com.voicebrief.*.plist
```

## プロジェクト構成

```
voice-brief/
├── cmd/
│   └── voicebrief/
│       └── main.go              # エントリーポイント
├── internal/
│   ├── config/                  # 設定読み込み
│   ├── model/                   # データモデル
│   ├── fetcher/                 # API Clients（Slack/Notion）
│   ├── filter/                  # フィルタリング・重要度計算
│   ├── summarizer/              # 要約・原稿生成
│   ├── tts/                     # 音声合成
│   └── uploader/                # Slack投稿
├── scripts/                     # セットアップスクリプト
├── out/                         # 出力先（.gitignore）
├── config.example.yaml
├── .env.example
└── README.md
```

## 開発

### テスト実行

```bash
go test ./... -v
```

### カバレッジ

```bash
go test ./... -cover
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### デバッグモード

```bash
# 生データを保存して確認
./voicebrief run --daily --debug-dump --log-level debug
cat out/debug/$(date +%Y-%m-%d)-raw.json | jq .
```

## トラブルシューティング

### Slack API接続エラー

**エラー:** `invalid_auth`

**対処:**
1. `.env`のトークンが正しいか確認
2. Slack Appが対象ワークスペースにインストールされているか確認
3. 必要なScopesが付与されているか確認

### Notion API接続エラー

**エラー:** `unauthorized`

**対処:**
1. `.env`のトークンが正しいか確認
2. 対象DatabaseにIntegrationが接続されているか確認
3. Database IDが正しいか確認（32文字のハイフン区切り）

### 音声が生成されない

**対処:**
1. `say`コマンドが利用可能か確認:

```bash
say "テスト"
```

2. ffmpegがインストールされているか確認（M4A出力時）:

```bash
ffmpeg -version
```

3. `--dry-run`で原稿が生成されるか確認:

```bash
./voicebrief run --daily --dry-run
cat out/daily/$(date +%Y-%m-%d).md
```

## ロードマップ

詳細は[Roadmap.md](./Roadmap.md)を参照してください。

- **v1.0** (Current): Daily/Weekly実行、Rule-based要約、macOS say TTS
- **v1.1**: Slackスレッド対応、Notionプロパティフィルタ強化
- **v1.2**: OpenAI要約・TTS統合、iPhoneショートカット連携
- **v2.0**: マルチプラットフォーム対応、Web UI

## ライセンス

MIT License

## コントリビューション

Issue・Pull Requestを歓迎します！

詳細な仕様は[Spec.md](./Spec.md)を参照してください。

## サポート

- GitHub Issues: https://github.com/masumomo/voice-brief/issues
