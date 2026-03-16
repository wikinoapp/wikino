// Package email_confirmation はメール確認ハンドラーを提供します
package email_confirmation

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/turnstile"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// Handler はメール確認ハンドラー
type Handler struct {
	cfg                     *config.Config
	sessionMgr              *session.Manager
	flashMgr                *session.FlashManager
	sendEmailConfirmationUC *usecase.SendEmailConfirmationUsecase
	markEmailAsConfirmedUC  *usecase.MarkEmailAsConfirmedUsecase
	createValidator         *validator.EmailConfirmationCreateValidator
	updateValidator         *validator.EmailConfirmationUpdateValidator
	turnstileVerifier       turnstile.Verifier
	limiter                 *ratelimit.Limiter
}

// NewHandler は新しいメール確認ハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	flashMgr *session.FlashManager,
	sendEmailConfirmationUC *usecase.SendEmailConfirmationUsecase,
	markEmailAsConfirmedUC *usecase.MarkEmailAsConfirmedUsecase,
	createValidator *validator.EmailConfirmationCreateValidator,
	updateValidator *validator.EmailConfirmationUpdateValidator,
	turnstileVerifier turnstile.Verifier,
	limiter *ratelimit.Limiter,
) *Handler {
	return &Handler{
		cfg:                     cfg,
		sessionMgr:              sessionMgr,
		flashMgr:                flashMgr,
		sendEmailConfirmationUC: sendEmailConfirmationUC,
		markEmailAsConfirmedUC:  markEmailAsConfirmedUC,
		createValidator:         createValidator,
		updateValidator:         updateValidator,
		turnstileVerifier:       turnstileVerifier,
		limiter:                 limiter,
	}
}
