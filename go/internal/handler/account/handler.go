// Package account はアカウント関連のハンドラーを提供します
package account

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// Handler はアカウントハンドラー
type Handler struct {
	cfg                 *config.Config
	sessionMgr          *session.Manager
	flashMgr            *session.FlashManager
	getAccountNewDataUC *usecase.GetAccountNewDataUsecase
	createValidator     *validator.AccountCreateValidator
	createAccountUC     *usecase.CreateAccountUsecase
	createUserSessionUC *usecase.CreateUserSessionUsecase
}

// NewHandler は新しいアカウントハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	flashMgr *session.FlashManager,
	getAccountNewDataUC *usecase.GetAccountNewDataUsecase,
	createValidator *validator.AccountCreateValidator,
	createAccountUC *usecase.CreateAccountUsecase,
	createUserSessionUC *usecase.CreateUserSessionUsecase,
) *Handler {
	return &Handler{
		cfg:                 cfg,
		sessionMgr:          sessionMgr,
		flashMgr:            flashMgr,
		getAccountNewDataUC: getAccountNewDataUC,
		createValidator:     createValidator,
		createAccountUC:     createAccountUC,
		createUserSessionUC: createUserSessionUC,
	}
}
