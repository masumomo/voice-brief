#!/bin/bash
# install_daily.sh - Daily Briefing用のlaunchdジョブをインストール
# 実行タイミング: 火〜土曜日 8:00（平日分のデータを翌日に取得）

set -e

echo "VoiceBrief Daily launchd インストール"
echo "========================================"
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

# LaunchAgentsディレクトリ
LAUNCH_AGENTS_DIR="$HOME/Library/LaunchAgents"

# ディレクトリが存在しない場合は作成
if [ ! -d "$LAUNCH_AGENTS_DIR" ]; then
    echo "LaunchAgentsディレクトリを作成: $LAUNCH_AGENTS_DIR"
    mkdir -p "$LAUNCH_AGENTS_DIR"
fi

DAILY_PLIST="com.voicebrief.daily.plist"

echo ""
echo "plistファイルを設定中..."

# Daily plist - パスを置換
sed -e "s|REPLACE_WITH_VOICEBRIEF_PATH|$VOICEBRIEF_BIN|g" \
    -e "s|REPLACE_WITH_PROJECT_DIR|$PROJECT_DIR|g" \
    "$SCRIPT_DIR/$DAILY_PLIST" > "/tmp/$DAILY_PLIST"

# LaunchAgentsにコピー
cp "/tmp/$DAILY_PLIST" "$LAUNCH_AGENTS_DIR/$DAILY_PLIST"

echo "✓ plistファイルをコピー完了"

# 既存のジョブをアンロード（存在する場合）
if launchctl list | grep -q "com.voicebrief.daily"; then
    echo ""
    echo "既存のDaily jobをアンロード中..."
    launchctl unload "$LAUNCH_AGENTS_DIR/$DAILY_PLIST" 2>/dev/null || true
fi

# 新しいジョブをロード
echo ""
echo "launchdジョブを登録中..."

launchctl load "$LAUNCH_AGENTS_DIR/$DAILY_PLIST"
echo "✓ Daily job登録完了"

# 確認
echo ""
echo "========================================"
echo "✅ Daily インストール完了！"
echo "========================================"
echo ""
echo "スケジュール:"
echo "  火〜土曜日 8:00 に実行"
echo "  （月〜金曜日のデータを翌朝取得）"
echo ""
echo "ログファイル:"
echo "  /tmp/voicebrief-daily.log"
echo ""
echo "アンインストール:"
echo "  launchctl unload ~/Library/LaunchAgents/com.voicebrief.daily.plist"
echo ""
