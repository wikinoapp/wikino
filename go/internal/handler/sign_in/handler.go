// Package sign_in はログインページのハンドラーを提供します
package sign_in

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/turnstile"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler はログインハンドラー
type Handler struct {
	cfg                   *config.Config
	sessionMgr            *session.Manager
	flashMgr              *session.FlashManager
	userSessionRepo       *repository.UserSessionRepository
	userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository
	createUserSessionUC   *usecase.CreateUserSessionUsecase
	turnstileVerifier     turnstile.Verifier
	validator             *CreateValidator
}

// NewHandler は新しいログインハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	flashMgr *session.FlashManager,
	userRepo *repository.UserRepository,
	userPasswordRepo *repository.UserPasswordRepository,
	userSessionRepo *repository.UserSessionRepository,
	userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository,
	createUserSessionUC *usecase.CreateUserSessionUsecase,
	turnstileVerifier turnstile.Verifier,
) *Handler {
	return &Handler{
		cfg:                   cfg,
		sessionMgr:            sessionMgr,
		flashMgr:              flashMgr,
		userSessionRepo:       userSessionRepo,
		userTwoFactorAuthRepo: userTwoFactorAuthRepo,
		createUserSessionUC:   createUserSessionUC,
		turnstileVerifier:     turnstileVerifier,
		validator:             NewCreateValidator(userRepo, userPasswordRepo),
	}
}
