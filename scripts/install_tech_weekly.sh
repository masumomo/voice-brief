#!/bin/bash
# install_tech_weekly.sh - Tech Weekly Briefing用のlaunchdジョブをインストール
# 実行タイミング: 日曜日 10:00

set -e

echo "VoiceBrief Tech Weekly launchd インストール"
echo "=============================================="
echo ""

# スクリプトのディレクトリを取得
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
VOICEBRIEF_BIN="$PROJECT_DIR/voicebrief"
CONFIG_FILE="$PROJECT_DIR/config.tech.yaml"

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

# 設定ファイルの存在確認
if [ ! -f "$CONFIG_FILE" ]; then
    echo "❌ エラー: 設定ファイルが見つかりません: $CONFIG_FILE"
    echo ""
    echo "次のコマンドで設定ファイルを作成してください:"
    echo "  cp config.tech.example.yaml config.tech.yaml"
    echo "  # 設定を編集"
    echo "  vim config.tech.yaml"
    echo ""
    exit 1
fi

echo "✓ 設定ファイルを確認: $CONFIG_FILE"

# LaunchAgentsディレクトリ
LAUNCH_AGENTS_DIR="$HOME/Library/LaunchAgents"

# ディレクトリが存在しない場合は作成
if [ ! -d "$LAUNCH_AGENTS_DIR" ]; then
    echo "LaunchAgentsディレクトリを作成: $LAUNCH_AGENTS_DIR"
    mkdir -p "$LAUNCH_AGENTS_DIR"
fi

PLIST_NAME="com.voicebrief.tech-weekly.plist"

echo ""
echo "plistファイルを設定中..."

# plist - パスを置換
sed -e "s|REPLACE_WITH_VOICEBRIEF_PATH|$VOICEBRIEF_BIN|g" \
    -e "s|REPLACE_WITH_PROJECT_DIR|$PROJECT_DIR|g" \
    -e "s|REPLACE_WITH_CONFIG_PATH|$CONFIG_FILE|g" \
    "$SCRIPT_DIR/$PLIST_NAME" > "/tmp/$PLIST_NAME"

# LaunchAgentsにコピー
cp "/tmp/$PLIST_NAME" "$LAUNCH_AGENTS_DIR/$PLIST_NAME"

echo "✓ plistファイルをコピー完了"

# 既存のジョブをアンロード（存在する場合）
if launchctl list | grep -q "com.voicebrief.tech-weekly"; then
    echo ""
    echo "既存のTech Weekly jobをアンロード中..."
    launchctl unload "$LAUNCH_AGENTS_DIR/$PLIST_NAME" 2>/dev/null || true
fi

# 新しいジョブをロード
echo ""
echo "launchdジョブを登録中..."

launchctl load "$LAUNCH_AGENTS_DIR/$PLIST_NAME"
echo "✓ Tech Weekly job登録完了"

# 確認
echo ""
echo "=============================================="
echo "✅ Tech Weekly インストール完了！"
echo "=============================================="
echo ""
echo "スケジュール:"
echo "  日曜日 10:00 に実行"
echo "  （過去1週間のtechデータを取得）"
echo ""
echo "設定ファイル:"
echo "  $CONFIG_FILE"
echo ""
echo "ログファイル:"
echo "  /tmp/voicebrief-tech-weekly.log"
echo "  /tmp/voicebrief-tech-weekly-error.log"
echo ""
echo "手動実行（テスト）:"
echo "  $VOICEBRIEF_BIN run --weekly --config $CONFIG_FILE --dry-run"
echo ""
echo "アンインストール:"
echo "  launchctl unload ~/Library/LaunchAgents/$PLIST_NAME"
echo ""
