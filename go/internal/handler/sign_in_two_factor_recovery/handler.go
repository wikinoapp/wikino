// Package sign_in_two_factor_recovery は2要素認証のリカバリーコードハンドラーを提供します
package sign_in_two_factor_recovery

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// Handler はリカバリーコードハンドラー
type Handler struct {
	cfg                   *config.Config
	sessionMgr            *session.Manager
	createValidator       *validator.SignInTwoFactorRecoveryCreateValidator
	consumeRecoveryCodeUC *usecase.ConsumeRecoveryCodeUsecase
	createUserSessionUC   *usecase.CreateUserSessionUsecase
}

// NewHandler は新しいリカバリーコードハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	createValidator *validator.SignInTwoFactorRecoveryCreateValidator,
	consumeRecoveryCodeUC *usecase.ConsumeRecoveryCodeUsecase,
	createUserSessionUC *usecase.CreateUserSessionUsecase,
) *Handler {
	return &Handler{
		cfg:                   cfg,
		sessionMgr:            sessionMgr,
		createValidator:       createValidator,
		consumeRecoveryCodeUC: consumeRecoveryCodeUC,
		createUserSessionUC:   createUserSessionUC,
	}
}
