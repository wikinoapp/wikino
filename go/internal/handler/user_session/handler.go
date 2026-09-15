// Package user_sessionはユーザーセッション (ログアウト) のハンドラーを提供します
package user_session

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはユーザーセッションハンドラー
type Handler struct {
	cfg                 *config.Config
	sessionMgr          *session.Manager
	flashMgr            *session.FlashManager
	deleteUserSessionUC *usecase.DeleteUserSessionUsecase
}

// NewHandlerは新しいユーザーセッションハンドラーを作成します
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
