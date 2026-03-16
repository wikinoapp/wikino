// Package password はパスワード更新機能のハンドラーを提供します
package password

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// Handler はパスワード更新機能のハンドラー
type Handler struct {
	cfg                   *config.Config
	sessionMgr            *session.Manager
	flashMgr              *session.FlashManager
	getTokenDataUC        *usecase.GetPasswordResetTokenDataUsecase
	updatePasswordUsecase *usecase.UpdatePasswordResetUsecase
	updateValidator       *validator.PasswordUpdateValidator
}

// NewHandler は新しいHandlerを作成します
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	flashMgr *session.FlashManager,
	getTokenDataUC *usecase.GetPasswordResetTokenDataUsecase,
	updatePasswordUsecase *usecase.UpdatePasswordResetUsecase,
	updateValidator *validator.PasswordUpdateValidator,
) *Handler {
	return &Handler{
		cfg:                   cfg,
		sessionMgr:            sessionMgr,
		flashMgr:              flashMgr,
		getTokenDataUC:        getTokenDataUC,
		updatePasswordUsecase: updatePasswordUsecase,
		updateValidator:       updateValidator,
	}
}
