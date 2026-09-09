// Package sign_in_two_factor_recovery は2要素認証のリカバリーコードハンドラーを提供します
package sign_in_two_factor_recovery

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler はリカバリーコードハンドラー
type Handler struct {
	cfg                         *config.Config
	sessionMgr                  *session.Manager
	flashMgr                    *session.FlashManager
	createRecoveryCodeSessionUC *usecase.CreateRecoveryCodeSessionUsecase
}

// NewHandler は新しいリカバリーコードハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	flashMgr *session.FlashManager,
	createRecoveryCodeSessionUC *usecase.CreateRecoveryCodeSessionUsecase,
) *Handler {
	return &Handler{
		cfg:                         cfg,
		sessionMgr:                  sessionMgr,
		flashMgr:                    flashMgr,
		createRecoveryCodeSessionUC: createRecoveryCodeSessionUC,
	}
}
