package uploader

import (
	"context"
	"fmt"
	"os"

	"github.com/slack-go/slack"

	"github.com/masumomo/voice-brief/internal/config"
	"github.com/masumomo/voice-brief/internal/model"
)

// SlackUploader はSlackへブリーフィングを投稿します
type SlackUploader struct {
	client      *slack.Client
	channelID   string
	uploadAudio bool
}

// NewSlackUploader は新しいSlackUploaderを作成します
func NewSlackUploader(cfg *config.SlackConfig, uploadAudio bool) *SlackUploader {
	token := os.Getenv(cfg.TokenEnv)
	if token == "" {
		panic(fmt.Sprintf("環境変数 %s が設定されていません", cfg.TokenEnv))
	}

	return &SlackUploader{
		client:      slack.New(token),
		channelID:   cfg.PostChannel,
		uploadAudio: uploadAudio,
	}
}

// Upload はブリーフィングをSlackに投稿します
func (s *SlackUploader) Upload(ctx context.Context, brief *model.Brief) error {
	if s.channelID == "" {
		return fmt.Errorf("投稿先チャンネルが設定されていません")
	}

	// Markdownテキストを投稿
	_, _, err := s.client.PostMessageContext(
		ctx,
		s.channelID,
		slack.MsgOptionText(fmt.Sprintf("*%s Briefing*", brief.Type), false),
		slack.MsgOptionBlocks(
			slack.NewSectionBlock(
				&slack.TextBlockObject{
					Type: slack.MarkdownType,
					Text: brief.ScriptMarkdown,
				},
				nil,
				nil,
			),
		),
	)
	if err != nil {
		return fmt.Errorf("Slackへのメッセージ投稿に失敗: %w", err)
	}

	// 音声ファイルをアップロード（オプション）
	if s.uploadAudio && brief.AudioPath != "" {
		if err := s.uploadFile(ctx, brief); err != nil {
			// ファイルアップロードは失敗してもエラーにしない（Best Effort）
			fmt.Printf("⚠️  警告: 音声ファイルのアップロードに失敗: %v\n", err)
		}
	}

	return nil
}

// uploadFile は音声ファイルをSlackにアップロードします
func (s *SlackUploader) uploadFile(ctx context.Context, brief *model.Brief) error {
	// ファイルが存在するか確認
	if _, err := os.Stat(brief.AudioPath); os.IsNotExist(err) {
		return fmt.Errorf("音声ファイルが見つかりません: %s", brief.AudioPath)
	}

	// ファイルをアップロード
	params := slack.FileUploadParameters{
		Channels: []string{s.channelID},
		File:     brief.AudioPath,
		Title:    fmt.Sprintf("%s Briefing Audio", brief.Type),
		Filetype: "audio",
	}

	_, err := s.client.UploadFileContext(ctx, params)
	if err != nil {
		return fmt.Errorf("ファイルアップロードに失敗: %w", err)
	}

	return nil
}

// GetProvider はプロバイダー名を返します
func (s *SlackUploader) GetProvider() string {
	return "slack"
}
