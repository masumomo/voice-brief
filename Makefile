.PHONY: help build test clean run-daily run-weekly config-check doctor install dev lint coverage

# デフォルトターゲット
help:
	@echo "VoiceBrief Makefile"
	@echo ""
	@echo "利用可能なコマンド:"
	@echo "  make build         バイナリをビルド"
	@echo "  make test          テストを実行"
	@echo "  make coverage      カバレッジレポートを生成"
	@echo "  make lint          コードを静的解析"
	@echo "  make clean         ビルド成果物を削除"
	@echo "  make run-daily     Daily Briefingを生成"
	@echo "  make run-weekly    Weekly Briefingを生成"
	@echo "  make config-check  設定ファイルを検証"
	@echo "  make doctor        API接続を確認"
	@echo "  make install       launchdに登録（自動実行設定）"
	@echo "  make dev           開発環境セットアップ"

# バイナリ名
BINARY_NAME=voicebrief
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.1.0-dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# ビルド
build:
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) cmd/voicebrief/main.go
	@echo "Build complete: ./$(BINARY_NAME)"

# 高速ビルド（開発用）
build-fast:
	@go build -o $(BINARY_NAME) cmd/voicebrief/main.go

# テスト実行
test:
	@echo "Running tests..."
	go test ./... -v

# カバレッジレポート生成
coverage:
	@echo "Generating coverage report..."
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# 静的解析
lint:
	@echo "Running linters..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Install: brew install golangci-lint"; exit 1)
	golangci-lint run ./...

# 依存関係の整理
tidy:
	@echo "Tidying dependencies..."
	go mod tidy

# ビルド成果物の削除
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	rm -rf out/daily/*
	rm -rf out/weekly/*
	rm -rf out/debug/*
	@echo "Clean complete"

# Daily Briefing生成
run-daily: build-fast
	@echo "Running Daily Briefing..."
	./$(BINARY_NAME) run --daily

# Daily Briefing生成（Dry-run）
run-daily-dry: build-fast
	@echo "Running Daily Briefing (dry-run)..."
	./$(BINARY_NAME) run --daily --dry-run

# Weekly Briefing生成
run-weekly: build-fast
	@echo "Running Weekly Briefing..."
	./$(BINARY_NAME) run --weekly

# Weekly Briefing生成（Dry-run）
run-weekly-dry: build-fast
	@echo "Running Weekly Briefing (dry-run)..."
	./$(BINARY_NAME) run --weekly --dry-run

# 設定ファイル検証
config-check: build-fast
	@./$(BINARY_NAME) config check

# API接続確認
doctor: build-fast
	@./$(BINARY_NAME) doctor

# バージョン表示
version: build-fast
	@./$(BINARY_NAME) version

# 開発環境セットアップ
dev:
	@echo "Setting up development environment..."
	@if [ ! -f config.yaml ]; then \
		cp config.example.yaml config.yaml; \
		echo "✓ config.yaml created from example"; \
	else \
		echo "✓ config.yaml already exists"; \
	fi
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "✓ .env created from example"; \
		echo ""; \
		echo "⚠️  .env ファイルを編集して、実際のトークンを設定してください:"; \
		echo "   - VOICE_BRIEF_SLACK_TOKEN"; \
		echo "   - VOICE_BRIEF_NOTION_TOKEN"; \
	else \
		echo "✓ .env already exists"; \
	fi
	@mkdir -p out/daily out/weekly out/debug
	@echo "✓ output directories created"
	@go mod download
	@echo "✓ dependencies downloaded"
	@echo ""
	@echo "Development environment ready!"
	@echo "Next steps:"
	@echo "  1. Edit .env and config.yaml with your tokens"
	@echo "  2. Run 'make config-check' to verify"
	@echo "  3. Run 'make doctor' to test API connections"

# launchdに登録（自動実行設定）
install:
	@echo "Installing to launchd..."
	@if [ ! -f scripts/install.sh ]; then \
		echo "Error: scripts/install.sh not found (will be created in Phase 3)"; \
		exit 1; \
	fi
	@./scripts/install.sh

# 出力ファイルを確認
show-daily:
	@ls -lh out/daily/ 2>/dev/null || echo "No daily briefings found"

show-weekly:
	@ls -lh out/weekly/ 2>/dev/null || echo "No weekly briefings found"

# 最新のDaily Briefingを表示
cat-daily:
	@latest=$$(ls -t out/daily/*.md 2>/dev/null | head -1); \
	if [ -n "$$latest" ]; then \
		cat "$$latest"; \
	else \
		echo "No daily briefings found"; \
	fi

# 最新のDaily Briefing音声を再生
play-daily:
	@latest=$$(ls -t out/daily/*.m4a 2>/dev/null | head -1); \
	if [ -n "$$latest" ]; then \
		echo "Playing: $$latest"; \
		open "$$latest"; \
	else \
		echo "No daily briefings found"; \
	fi

# デバッグ用：環境変数チェック
check-env:
	@echo "Checking environment variables..."
	@echo "VOICE_BRIEF_SLACK_TOKEN: $${VOICE_BRIEF_SLACK_TOKEN:+SET (hidden)} $${VOICE_BRIEF_SLACK_TOKEN:-NOT SET}"
	@echo "VOICE_BRIEF_NOTION_TOKEN: $${VOICE_BRIEF_NOTION_TOKEN:+SET (hidden)} $${VOICE_BRIEF_NOTION_TOKEN:-NOT SET}"
	@echo "VOICE_BRIEF_GITHUB_TOKEN: $${VOICE_BRIEF_GITHUB_TOKEN:+SET (hidden)} $${VOICE_BRIEF_GITHUB_TOKEN:-NOT SET}"
	@echo "VOICE_BRIEF_OPENAI_API_KEY: $${VOICE_BRIEF_OPENAI_API_KEY:+SET (hidden)} $${VOICE_BRIEF_OPENAI_API_KEY:-NOT SET}"

# CI用：全チェック実行
ci: lint test build
	@echo "All CI checks passed!"

# リリース用ビルド
release:
	@echo "Building release binaries..."
	@mkdir -p dist
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 cmd/voicebrief/main.go
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 cmd/voicebrief/main.go
	@echo "Release binaries created in dist/"
	@ls -lh dist/
