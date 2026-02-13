// Package user_session はユーザーセッション（ログアウト）のハンドラーを提供します
package user_session

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
)

// Handler はユーザーセッションハンドラー
type Handler struct {
	cfg             *config.Config
	sessionMgr      *session.Manager
	flashMgr        *session.FlashManager
	userSessionRepo *repository.UserSessionRepository
}

// NewHandler は新しいユーザーセッションハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	flashMgr *session.FlashManager,
	userSessionRepo *repository.UserSessionRepository,
) *Handler {
	return &Handler{
		cfg:             cfg,
		sessionMgr:      sessionMgr,
		flashMgr:        flashMgr,
		userSessionRepo: userSessionRepo,
	}
}
