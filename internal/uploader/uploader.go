package uploader

import (
	"context"

	"github.com/masumomo/voice-brief/internal/model"
)

// Uploader はブリーフィングを外部サービスにアップロードするインターフェース
type Uploader interface {
	// Upload はブリーフィングをアップロードします
	Upload(ctx context.Context, brief *model.Brief) error

	// GetProvider はアップロード先のプロバイダー名を返します
	GetProvider() string
}
