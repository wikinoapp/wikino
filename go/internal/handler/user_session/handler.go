// Package user_session はユーザーセッション（ログアウト）のハンドラーを提供します
package user_session

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler はユーザーセッションハンドラー
type Handler struct {
	cfg                 *config.Config
	sessionMgr          *session.Manager
	flashMgr            *session.FlashManager
	deleteUserSessionUC *usecase.DeleteUserSessionUsecase
}

// NewHandler は新しいユーザーセッションハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	flashMgr *session.FlashManager,
	deleteUserSessionUC *usecase.DeleteUserSessionUsecase,
) *Handler {
	return &Handler{
		cfg:                 cfg,
		sessionMgr:          sessionMgr,
		flashMgr:            flashMgr,
		deleteUserSessionUC: deleteUserSessionUC,
	}
}
