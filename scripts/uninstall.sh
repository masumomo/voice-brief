#!/bin/bash

set -e

echo "VoiceBrief launchd アンインストールスクリプト"
echo "=============================================="
echo ""

LAUNCH_AGENTS_DIR="$HOME/Library/LaunchAgents"
TODAY_PLIST="com.voicebrief.today.plist"
DAILY_PLIST="com.voicebrief.daily.plist"
WEEKLY_PLIST="com.voicebrief.weekly.plist"

# Today jobのアンロード
if [ -f "$LAUNCH_AGENTS_DIR/$TODAY_PLIST" ]; then
    echo "Today jobをアンロード中..."
    launchctl unload "$LAUNCH_AGENTS_DIR/$TODAY_PLIST" 2>/dev/null || true
    rm "$LAUNCH_AGENTS_DIR/$TODAY_PLIST"
    echo "✓ Today job削除完了"
else
    echo "ℹ️  Today jobは登録されていません"
fi

# Daily jobのアンロード
if [ -f "$LAUNCH_AGENTS_DIR/$DAILY_PLIST" ]; then
    echo "Daily jobをアンロード中..."
    launchctl unload "$LAUNCH_AGENTS_DIR/$DAILY_PLIST" 2>/dev/null || true
    rm "$LAUNCH_AGENTS_DIR/$DAILY_PLIST"
    echo "✓ Daily job削除完了"
else
    echo "ℹ️  Daily jobは登録されていません"
fi

# Weekly jobのアンロード
if [ -f "$LAUNCH_AGENTS_DIR/$WEEKLY_PLIST" ]; then
    echo "Weekly jobをアンロード中..."
    launchctl unload "$LAUNCH_AGENTS_DIR/$WEEKLY_PLIST" 2>/dev/null || true
    rm "$LAUNCH_AGENTS_DIR/$WEEKLY_PLIST"
    echo "✓ Weekly job削除完了"
else
    echo "ℹ️  Weekly jobは登録されていません"
fi

echo ""
echo "=========================================="
echo "✅ アンインストール完了！"
echo "=========================================="
echo ""
echo "残存ジョブ確認:"
launchctl list | grep voicebrief || echo "（voicebrief関連のジョブはありません）"
echo ""
