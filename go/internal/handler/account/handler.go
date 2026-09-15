// Package accountはアカウント関連のハンドラーを提供します
package account

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはアカウントハンドラー
type Handler struct {
	cfg                 *config.Config
	sessionMgr          *session.Manager
	flashMgr            *session.FlashManager
	getAccountNewDataUC *usecase.GetAccountNewDataUsecase
	createAccountUC     *usecase.CreateAccountUsecase
	createUserSessionUC *usecase.CreateUserSessionUsecase
}

// NewHandlerは新しいアカウントハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	flashMgr *session.FlashManager,
	getAccountNewDataUC *usecase.GetAccountNewDataUsecase,
	createAccountUC *usecase.CreateAccountUsecase,
	createUserSessionUC *usecase.CreateUserSessionUsecase,
) *Handler {
	return &Handler{
		cfg:                 cfg,
		sessionMgr:          sessionMgr,
		flashMgr:            flashMgr,
		getAccountNewDataUC: getAccountNewDataUC,
		createAccountUC:     createAccountUC,
		createUserSessionUC: createUserSessionUC,
	}
}
