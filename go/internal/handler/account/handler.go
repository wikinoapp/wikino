// Package account はアカウント関連のハンドラーを提供します
package account

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler はアカウントハンドラー
type Handler struct {
	cfg                   *config.Config
	sessionMgr            *session.Manager
	flashMgr              *session.FlashManager
	emailConfirmationRepo *repository.EmailConfirmationRepository
	createValidator       *CreateValidator
	createAccountUC       *usecase.CreateAccountUsecase
	createUserSessionUC   *usecase.CreateUserSessionUsecase
}

// NewHandler は新しいアカウントハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	flashMgr *session.FlashManager,
	emailConfirmationRepo *repository.EmailConfirmationRepository,
	userRepo *repository.UserRepository,
	createAccountUC *usecase.CreateAccountUsecase,
	createUserSessionUC *usecase.CreateUserSessionUsecase,
) *Handler {
	return &Handler{
		cfg:                   cfg,
		sessionMgr:            sessionMgr,
		flashMgr:              flashMgr,
		emailConfirmationRepo: emailConfirmationRepo,
		createValidator:       NewCreateValidator(emailConfirmationRepo, userRepo),
		createAccountUC:       createAccountUC,
		createUserSessionUC:   createUserSessionUC,
	}
}
