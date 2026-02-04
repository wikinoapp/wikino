// Package password_reset はパスワードリセット申請機能のハンドラーを提供します
package password_reset

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/turnstile"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler はパスワードリセット申請機能のハンドラー
type Handler struct {
	cfg                *config.Config
	sessionMgr         *session.Manager
	flashMgr           *session.FlashManager
	userRepo           *repository.UserRepository
	limiter            *ratelimit.Limiter
	turnstileVerifier  turnstile.Verifier
	createTokenUsecase *usecase.CreatePasswordResetTokenUsecase
}

// NewHandler は新しいHandlerを作成します
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	flashMgr *session.FlashManager,
	userRepo *repository.UserRepository,
	limiter *ratelimit.Limiter,
	turnstileVerifier turnstile.Verifier,
	createTokenUsecase *usecase.CreatePasswordResetTokenUsecase,
) *Handler {
	return &Handler{
		cfg:                cfg,
		sessionMgr:         sessionMgr,
		flashMgr:           flashMgr,
		userRepo:           userRepo,
		limiter:            limiter,
		turnstileVerifier:  turnstileVerifier,
		createTokenUsecase: createTokenUsecase,
	}
}
