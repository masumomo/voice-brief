#!/bin/bash
# install.sh - 全てのlaunchdジョブをインストール
#
# 個別にインストールしたい場合:
#   ./scripts/install_today.sh   - Today のみ（月〜金曜日 21:00）
#   ./scripts/install_daily.sh   - Daily のみ（火〜土曜日 8:00）
#   ./scripts/install_weekly.sh  - Weekly のみ（土曜日 9:00）

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "VoiceBrief launchd インストール（全ジョブ）"
echo "============================================="
echo ""

# Today インストール
"$SCRIPT_DIR/install_today.sh"

echo ""

# Daily インストール
"$SCRIPT_DIR/install_daily.sh"

echo ""

# Weekly インストール
"$SCRIPT_DIR/install_weekly.sh"

echo ""
echo "============================================="
echo "✅ 全ジョブのインストール完了！"
echo "============================================="
echo ""
echo "登録されたジョブ:"
launchctl list | grep voicebrief || echo "（ジョブが見つかりませんでした）"
echo ""
echo "スケジュール:"
echo "  - Today:  月〜金曜日 21:00（当日分）"
echo "  - Daily:  火〜土曜日 8:00（前日分）"
echo "  - Weekly: 土曜日 9:00（過去1週間）"
echo ""
echo "個別にインストール/アンインストールする場合:"
echo "  ./scripts/install_today.sh"
echo "  ./scripts/install_daily.sh"
echo "  ./scripts/install_weekly.sh"
echo "  ./scripts/uninstall.sh"
echo ""
