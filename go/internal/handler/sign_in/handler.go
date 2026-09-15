// Package sign_inはログインページのハンドラーを提供します
package sign_in

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/turnstile"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはログインハンドラー
type Handler struct {
	cfg               *config.Config
	sessionMgr        *session.Manager
	flashMgr          *session.FlashManager
	signInUC          *usecase.CreateSignInUsecase
	turnstileVerifier turnstile.Verifier
}

// NewHandlerは新しいログインハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	flashMgr *session.FlashManager,
	signInUC *usecase.CreateSignInUsecase,
	turnstileVerifier turnstile.Verifier,
) *Handler {
	return &Handler{
		cfg:               cfg,
		sessionMgr:        sessionMgr,
		flashMgr:          flashMgr,
		signInUC:          signInUC,
		turnstileVerifier: turnstileVerifier,
	}
}
