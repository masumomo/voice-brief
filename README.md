# VoiceBrief

音声で聞く、チームの最新情報

## 概要

VoiceBriefは、SlackとNotionの更新情報を自動的に収集し、音声ブリーフィングとして提供するCLIツールです。
ハンズフリーで業務の最新情報をキャッチアップできるため、通勤中や作業中など、画面を見られない状況でも効率的に情報収集が可能です。

### 主な特徴

- **完全ローカル動作**: 生データをディスクに保存せず、プライバシーを保護
- **AI要約対応**: Gemini 2.0 Flash（無料枠）による自然な要約生成
- **高品質音声合成**: macOS/Windows標準TTS or Google Cloud TTS（WaveNet音声）
- **並列高速取得**: SlackとNotionから同時並行でデータを取得
- **重要度フィルタリング**: ノイズを除外し、重要な情報のみを抽出
- **定期自動実行**: launchdで毎朝自動的にブリーフィング生成
- **クロスプラットフォーム**: macOS/Windows両対応

## 必要環境

- macOS 12.0以降 または Windows 10/11
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

#### Gemini API Key（オプション）

AI要約機能を使用する場合は、Gemini API Keyが必要です。

1. https://aistudio.google.com/app/apikey にアクセス
2. 「Create API Key」でAPIキーを作成
3. `.env`ファイルに設定:

```bash
GEMINI_API_KEY="your-gemini-api-key-here"
```

**無料枠:** 毎分15リクエスト、毎日1500リクエストまで無料で利用可能

#### Google Cloud TTS認証情報（オプション）

高品質な日本語音声合成（WaveNet）を使用する場合は、Google Cloud認証情報が必要です。

1. https://console.cloud.google.com/ にアクセス
2. プロジェクトを作成（または既存のプロジェクトを選択）
3. Cloud Text-to-Speech APIを有効化
4. サービスアカウントキー（JSON）を作成してダウンロード
5. JSONファイルの内容を`.env`ファイルに設定:

```bash
GOOGLE_APPLICATION_CREDENTIALS_JSON='{"type":"service_account",...}'
```

**または、gcloud CLIで認証:**
```bash
gcloud auth application-default login
```

**無料枠:** 毎月100万文字まで無料（WaveNetとStandard合計）

#### OpenAI API Key（オプション）

高品質なAI要約または音声合成を使用する場合は、OpenAI API Keyが必要です。

1. https://platform.openai.com/api-keys にアクセス
2. 「Create new secret key」でAPIキーを作成
3. `.env`ファイルに追加：

```bash
OPENAI_API_KEY="sk-your-openai-key-here"
```

**コスト目安（従量課金）:**

| サービス | モデル | Daily実行 | Weekly実行 |
|---------|--------|----------|-----------|
| Summarizer | gpt-4o-mini | $0.001-0.003/回 | $0.003-0.008/回 |
| Summarizer | gpt-4o | $0.02-0.05/回 | $0.05-0.15/回 |
| TTS | tts-1 | $0.03-0.06/回 | $0.08-0.15/回 |
| TTS | tts-1-hd | $0.06-0.12/回 | $0.15-0.30/回 |

**月次コスト試算:**
- Daily実行（毎日1回、gpt-4o-mini + tts-1）: 約 $1.2-2.7/月
- Daily + Weekly実行（gpt-4o-mini + tts-1）: 約 $1.5-3.5/月

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

# 要約エンジン設定（オプション）
summarizer:
  provider: "rule"  # "rule" | "gemini" (無料) | "openai" (従量課金)
  # gemini_model: "gemini-2.0-flash-exp"  # Gemini使用時のモデル
  # gemini_api_key_env: "GEMINI_API_KEY"  # Gemini使用時
  # openai_model: "gpt-4o-mini"  # OpenAI使用時（gpt-4o-mini, gpt-4o等）
  # openai_api_key_env: "OPENAI_API_KEY"  # OpenAI使用時

# 音声合成設定（オプション）
tts:
  provider: "say"  # "say" (macOS) | "sapi" (Windows) | "google_tts" (無料枠) | "openai_tts" (従量課金)
  voice: "Kyoko"   # say/sapi: Kyoko, Otoya / google_tts: ja-JP-Neural2-B / openai_tts: alloy, nova, shimmer等
  rate: 1.1        # 読み上げ速度（倍速）
  # google_credentials_json_env: "GOOGLE_APPLICATION_CREDENTIALS_JSON"  # Google TTS使用時
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

- ✅ **v1.0**: Daily/Weekly実行、Rule-based要約、macOS say TTS
- ✅ **v1.1**: Slackスレッド対応、Notionプロパティフィルタ強化
- ✅ **v1.2**: Gemini AI要約統合、Google Cloud TTS（無料枠）
- ✅ **v2.0**: Windows対応（SAPI TTS）
- ✅ **v2.1**: OpenAI統合（GPT-4o Summarizer + OpenAI TTS）
- ✅ **v2.2** (Current): GitHub統合（3ソース並列取得）

## ライセンス

MIT License

## コントリビューション

Issue・Pull Requestを歓迎します！

詳細な仕様は[Spec.md](./Spec.md)を参照してください。

## サポート

- GitHub Issues: https://github.com/masumomo/voice-brief/issues
