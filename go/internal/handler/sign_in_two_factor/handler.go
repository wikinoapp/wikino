// Package sign_in_two_factor は2要素認証のハンドラーを提供します
package sign_in_two_factor

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// Handler は2要素認証ハンドラー
type Handler struct {
	cfg                 *config.Config
	sessionMgr          *session.Manager
	createValidator     *validator.SignInTwoFactorCreateValidator
	createUserSessionUC *usecase.CreateUserSessionUsecase
}

// NewHandler は新しい2要素認証ハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	createValidator *validator.SignInTwoFactorCreateValidator,
	createUserSessionUC *usecase.CreateUserSessionUsecase,
) *Handler {
	return &Handler{
		cfg:                 cfg,
		sessionMgr:          sessionMgr,
		createValidator:     createValidator,
		createUserSessionUC: createUserSessionUC,
	}
}
