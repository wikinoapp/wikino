// Package account はアカウント関連のハンドラーを提供します
package account

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
)

// Handler はアカウントハンドラー
type Handler struct {
	cfg                   *config.Config
	sessionMgr            *session.Manager
	flashMgr              *session.FlashManager
	emailConfirmationRepo *repository.EmailConfirmationRepository
}

// NewHandler は新しいアカウントハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	flashMgr *session.FlashManager,
	emailConfirmationRepo *repository.EmailConfirmationRepository,
) *Handler {
	return &Handler{
		cfg:                   cfg,
		sessionMgr:            sessionMgr,
		flashMgr:              flashMgr,
		emailConfirmationRepo: emailConfirmationRepo,
	}
}
