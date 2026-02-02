// Package email_confirmation はメール確認ハンドラーを提供します
package email_confirmation

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/turnstile"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler はメール確認ハンドラー
type Handler struct {
	cfg                       *config.Config
	sessionMgr                *session.Manager
	flashMgr                  *session.FlashManager
	userRepo                  *repository.UserRepository
	sendEmailConfirmationUC   *usecase.SendEmailConfirmationUsecase
	verifyEmailConfirmationUC *usecase.VerifyEmailConfirmationUsecase
	turnstileVerifier         turnstile.Verifier
}

// NewHandler は新しいメール確認ハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	flashMgr *session.FlashManager,
	userRepo *repository.UserRepository,
	sendEmailConfirmationUC *usecase.SendEmailConfirmationUsecase,
	verifyEmailConfirmationUC *usecase.VerifyEmailConfirmationUsecase,
	turnstileVerifier turnstile.Verifier,
) *Handler {
	return &Handler{
		cfg:                       cfg,
		sessionMgr:                sessionMgr,
		flashMgr:                  flashMgr,
		userRepo:                  userRepo,
		sendEmailConfirmationUC:   sendEmailConfirmationUC,
		verifyEmailConfirmationUC: verifyEmailConfirmationUC,
		turnstileVerifier:         turnstileVerifier,
	}
}
