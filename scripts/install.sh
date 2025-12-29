#!/bin/bash

set -e

echo "VoiceBrief launchd インストールスクリプト"
echo "=========================================="
echo ""

# スクリプトのディレクトリを取得
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
VOICEBRIEF_BIN="$PROJECT_DIR/voicebrief"

# バイナリの存在確認
if [ ! -f "$VOICEBRIEF_BIN" ]; then
    echo "❌ エラー: voicebrief バイナリが見つかりません: $VOICEBRIEF_BIN"
    echo ""
    echo "次のコマンドでビルドしてください:"
    echo "  make build"
    echo ""
    exit 1
fi

echo "✓ voicebrief バイナリを確認: $VOICEBRIEF_BIN"

# 環境変数の確認
if [ -z "$VOICE_BRIEF_SLACK_TOKEN" ] || [ -z "$VOICE_BRIEF_NOTION_TOKEN" ]; then
    echo ""
    echo "⚠️  警告: 環境変数が設定されていません"
    echo ""
    echo "launchdから実行するには、以下の環境変数を ~/.zshrc または ~/.bashrc に設定してください:"
    echo ""
    echo '  export VOICE_BRIEF_SLACK_TOKEN="xoxb-your-slack-token"'
    echo '  export VOICE_BRIEF_NOTION_TOKEN="secret_your-notion-token"'
    echo ""
    echo "または、plistファイルのEnvironmentVariablesセクションに直接記述してください。"
    echo ""
    read -p "続行しますか？ (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "インストールを中止しました"
        exit 1
    fi
fi

# LaunchAgentsディレクトリ
LAUNCH_AGENTS_DIR="$HOME/Library/LaunchAgents"

# ディレクトリが存在しない場合は作成
if [ ! -d "$LAUNCH_AGENTS_DIR" ]; then
    echo "LaunchAgentsディレクトリを作成: $LAUNCH_AGENTS_DIR"
    mkdir -p "$LAUNCH_AGENTS_DIR"
fi

# 一時ファイルにplistをコピーして、パスを置換
DAILY_PLIST="com.voicebrief.daily.plist"
WEEKLY_PLIST="com.voicebrief.weekly.plist"

echo ""
echo "plistファイルを設定中..."

# Daily plist
sed -e "s|REPLACE_WITH_VOICEBRIEF_PATH|$VOICEBRIEF_BIN|g" \
    -e "s|REPLACE_WITH_PROJECT_DIR|$PROJECT_DIR|g" \
    "$SCRIPT_DIR/$DAILY_PLIST" > "/tmp/$DAILY_PLIST"

# Weekly plist
sed -e "s|REPLACE_WITH_VOICEBRIEF_PATH|$VOICEBRIEF_BIN|g" \
    -e "s|REPLACE_WITH_PROJECT_DIR|$PROJECT_DIR|g" \
    "$SCRIPT_DIR/$WEEKLY_PLIST" > "/tmp/$WEEKLY_PLIST"

# LaunchAgentsにコピー
cp "/tmp/$DAILY_PLIST" "$LAUNCH_AGENTS_DIR/$DAILY_PLIST"
cp "/tmp/$WEEKLY_PLIST" "$LAUNCH_AGENTS_DIR/$WEEKLY_PLIST"

echo "✓ plistファイルをコピー完了"

# 既存のジョブをアンロード（存在する場合）
if launchctl list | grep -q "com.voicebrief.daily"; then
    echo ""
    echo "既存のDaily jobをアンロード中..."
    launchctl unload "$LAUNCH_AGENTS_DIR/$DAILY_PLIST" 2>/dev/null || true
fi

if launchctl list | grep -q "com.voicebrief.weekly"; then
    echo ""
    echo "既存のWeekly jobをアンロード中..."
    launchctl unload "$LAUNCH_AGENTS_DIR/$WEEKLY_PLIST" 2>/dev/null || true
fi

# 新しいジョブをロード
echo ""
echo "launchdジョブを登録中..."

launchctl load "$LAUNCH_AGENTS_DIR/$DAILY_PLIST"
echo "✓ Daily job登録完了"

launchctl load "$LAUNCH_AGENTS_DIR/$WEEKLY_PLIST"
echo "✓ Weekly job登録完了"

# 確認
echo ""
echo "=========================================="
echo "✅ インストール完了！"
echo "=========================================="
echo ""
echo "登録されたジョブ:"
launchctl list | grep voicebrief || echo "（ジョブが見つかりませんでした）"
echo ""
echo "スケジュール:"
echo "  - Daily:  毎朝 8:00"
echo "  - Weekly: 毎週月曜 8:00"
echo ""
echo "ログファイル:"
echo "  - Daily:  /tmp/voicebrief-daily.log"
echo "  - Weekly: /tmp/voicebrief-weekly.log"
echo ""
echo "アンインストールする場合:"
echo "  ./scripts/uninstall.sh"
echo ""
