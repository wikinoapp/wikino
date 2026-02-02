// Package sign_up はサインアップページのハンドラーを提供します
package sign_up

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/session"
)

// Handler はサインアップハンドラー
type Handler struct {
	cfg        *config.Config
	sessionMgr *session.Manager
}

// NewHandler は新しいサインアップハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
) *Handler {
	return &Handler{
		cfg:        cfg,
		sessionMgr: sessionMgr,
	}
}
