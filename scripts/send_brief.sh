#!/bin/bash
# send_brief.sh - Send brief to Slack and generate audio from SCRIPT section
# Usage: ./scripts/send_brief.sh <markdown_file> [--dry-run]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check arguments
if [ $# -lt 1 ]; then
    echo -e "${RED}Usage: $0 <markdown_file> [--dry-run]${NC}"
    echo "Example: $0 out/weekly/2025-W52.md"
    exit 1
fi

MARKDOWN_FILE="$1"
DRY_RUN=false

if [ "$2" = "--dry-run" ]; then
    DRY_RUN=true
    echo -e "${YELLOW}[DRY-RUN MODE]${NC}"
fi

# Check if file exists
if [ ! -f "$MARKDOWN_FILE" ]; then
    echo -e "${RED}Error: File not found: $MARKDOWN_FILE${NC}"
    exit 1
fi

# Load environment variables from .env if exists
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

if [ -f "$PROJECT_ROOT/.env" ]; then
    export $(grep -v '^#' "$PROJECT_ROOT/.env" | xargs)
fi

# Check required environment variables
if [ -z "$SLACK_WEBHOOK_URL" ] && [ -z "$VOICE_BRIEF_SLACK_TOKEN" ]; then
    echo -e "${RED}Error: SLACK_WEBHOOK_URL or VOICE_BRIEF_SLACK_TOKEN must be set${NC}"
    exit 1
fi

if [ -z "$OPENAI_API_KEY" ]; then
    echo -e "${RED}Error: OPENAI_API_KEY must be set${NC}"
    exit 1
fi

# Parse the markdown file - split at "SCRIPT" line
echo -e "${GREEN}Parsing markdown file...${NC}"

# Find the line number where SCRIPT appears (standalone line)
SCRIPT_LINE=$(grep -n "^SCRIPT$" "$MARKDOWN_FILE" | head -1 | cut -d: -f1)

if [ -z "$SCRIPT_LINE" ]; then
    # Try alternative: "---SCRIPT---"
    SCRIPT_LINE=$(grep -n "^---SCRIPT---$" "$MARKDOWN_FILE" | head -1 | cut -d: -f1)
fi

if [ -z "$SCRIPT_LINE" ]; then
    echo -e "${RED}Error: Could not find SCRIPT delimiter in file${NC}"
    exit 1
fi

# Extract markdown content (before SCRIPT) - remove the "---" separator line before SCRIPT
MARKDOWN_CONTENT=$(head -n $((SCRIPT_LINE - 2)) "$MARKDOWN_FILE")

# Extract script content (after SCRIPT)
SCRIPT_CONTENT=$(tail -n +$((SCRIPT_LINE + 1)) "$MARKDOWN_FILE")

# Trim whitespace
SCRIPT_CONTENT=$(echo "$SCRIPT_CONTENT" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')

echo -e "${GREEN}Markdown content: $(echo "$MARKDOWN_CONTENT" | wc -c | tr -d ' ') bytes${NC}"
echo -e "${GREEN}Script content: $(echo "$SCRIPT_CONTENT" | wc -c | tr -d ' ') bytes${NC}"

# --- Send to Slack ---
echo -e "\n${GREEN}Sending to Slack...${NC}"

if [ "$DRY_RUN" = true ]; then
    echo -e "${YELLOW}[DRY-RUN] Would send markdown to Slack${NC}"
    echo "---"
    echo "$MARKDOWN_CONTENT" | head -20
    echo "..."
    echo "---"
else
    if [ -n "$SLACK_WEBHOOK_URL" ]; then
        # Use Webhook
        # Escape special characters for JSON
        ESCAPED_CONTENT=$(echo "$MARKDOWN_CONTENT" | python3 -c 'import json,sys; print(json.dumps(sys.stdin.read()))')

        RESPONSE=$(curl -s -X POST "$SLACK_WEBHOOK_URL" \
            -H "Content-Type: application/json" \
            -d "{\"text\": ${ESCAPED_CONTENT}}")

        if [ "$RESPONSE" = "ok" ]; then
            echo -e "${GREEN}Slack message sent successfully${NC}"
        else
            echo -e "${RED}Slack error: $RESPONSE${NC}"
        fi
    else
        # Use Slack API with token
        ESCAPED_CONTENT=$(echo "$MARKDOWN_CONTENT" | python3 -c 'import json,sys; print(json.dumps(sys.stdin.read()))')

        # Get default channel from config or use #general
        SLACK_CHANNEL="${SLACK_CHANNEL:-general}"

        RESPONSE=$(curl -s -X POST "https://slack.com/api/chat.postMessage" \
            -H "Authorization: Bearer $VOICE_BRIEF_SLACK_TOKEN" \
            -H "Content-Type: application/json" \
            -d "{\"channel\": \"$SLACK_CHANNEL\", \"text\": ${ESCAPED_CONTENT}}")

        if echo "$RESPONSE" | grep -q '"ok":true'; then
            echo -e "${GREEN}Slack message sent successfully${NC}"
        else
            echo -e "${RED}Slack error: $RESPONSE${NC}"
        fi
    fi
fi

# --- Generate audio with OpenAI TTS ---
echo -e "\n${GREEN}Generating audio with OpenAI TTS...${NC}"

# Determine output path
BASENAME=$(basename "$MARKDOWN_FILE" .md)
DIRNAME=$(dirname "$MARKDOWN_FILE")
OUTPUT_AUDIO="${DIRNAME}/${BASENAME}.mp3"

if [ "$DRY_RUN" = true ]; then
    echo -e "${YELLOW}[DRY-RUN] Would generate audio to: $OUTPUT_AUDIO${NC}"
    echo -e "${YELLOW}Script content (first 500 chars):${NC}"
    echo "$SCRIPT_CONTENT" | head -c 500
    echo "..."
else
    # Call OpenAI TTS API
    # Note: OpenAI TTS has a limit of 4096 characters per request
    CHAR_COUNT=$(echo -n "$SCRIPT_CONTENT" | wc -c | tr -d ' ')

    if [ "$CHAR_COUNT" -gt 4096 ]; then
        echo -e "${YELLOW}Warning: Script is $CHAR_COUNT chars (>4096), will be truncated${NC}"
        SCRIPT_CONTENT=$(echo "$SCRIPT_CONTENT" | head -c 4096)
    fi

    # Create request JSON
    REQUEST_JSON=$(python3 -c "
import json
import sys
content = '''$SCRIPT_CONTENT'''
print(json.dumps({
    'model': 'tts-1',
    'input': content,
    'voice': 'nova',
    'response_format': 'mp3'
}))
")

    echo "Calling OpenAI TTS API..."
    HTTP_CODE=$(curl -s -w "%{http_code}" -o "$OUTPUT_AUDIO" \
        -X POST "https://api.openai.com/v1/audio/speech" \
        -H "Authorization: Bearer $OPENAI_API_KEY" \
        -H "Content-Type: application/json" \
        -d "$REQUEST_JSON")

    if [ "$HTTP_CODE" = "200" ]; then
        FILE_SIZE=$(ls -lh "$OUTPUT_AUDIO" | awk '{print $5}')
        echo -e "${GREEN}Audio generated: $OUTPUT_AUDIO ($FILE_SIZE)${NC}"
    else
        echo -e "${RED}OpenAI TTS error (HTTP $HTTP_CODE)${NC}"
        if [ -f "$OUTPUT_AUDIO" ]; then
            cat "$OUTPUT_AUDIO"
            rm "$OUTPUT_AUDIO"
        fi
        exit 1
    fi
fi

echo -e "\n${GREEN}Done!${NC}"
