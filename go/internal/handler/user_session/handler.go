// Package user_session はユーザーセッション（ログイン・ログアウト）のハンドラーを提供します
package user_session

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/turnstile"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler はユーザーセッションハンドラー
type Handler struct {
	cfg                 *config.Config
	sessionMgr          *session.Manager
	userRepo            *repository.UserRepository
	userPasswordRepo    *repository.UserPasswordRepository
	createUserSessionUC *usecase.CreateUserSessionUsecase
	turnstileVerifier   turnstile.Verifier
}

// NewHandler は新しいユーザーセッションハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	userRepo *repository.UserRepository,
	userPasswordRepo *repository.UserPasswordRepository,
	createUserSessionUC *usecase.CreateUserSessionUsecase,
	turnstileVerifier turnstile.Verifier,
) *Handler {
	return &Handler{
		cfg:                 cfg,
		sessionMgr:          sessionMgr,
		userRepo:            userRepo,
		userPasswordRepo:    userPasswordRepo,
		createUserSessionUC: createUserSessionUC,
		turnstileVerifier:   turnstileVerifier,
	}
}
