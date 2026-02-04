// Package password はパスワード更新機能のハンドラーを提供します
package password

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler はパスワード更新機能のハンドラー
type Handler struct {
	cfg                    *config.Config
	sessionMgr             *session.Manager
	flashMgr               *session.FlashManager
	passwordResetTokenRepo *repository.PasswordResetTokenRepository
	updatePasswordUsecase  *usecase.UpdatePasswordResetUsecase
}

// NewHandler は新しいHandlerを作成します
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	flashMgr *session.FlashManager,
	passwordResetTokenRepo *repository.PasswordResetTokenRepository,
	updatePasswordUsecase *usecase.UpdatePasswordResetUsecase,
) *Handler {
	return &Handler{
		cfg:                    cfg,
		sessionMgr:             sessionMgr,
		flashMgr:               flashMgr,
		passwordResetTokenRepo: passwordResetTokenRepo,
		updatePasswordUsecase:  updatePasswordUsecase,
	}
}
