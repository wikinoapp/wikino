// Package welcomeはトップページ (ウェルカムページ) のハンドラーを提供します
package welcome

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/session"
)

// Handlerはトップページ関連のHTTPハンドラーです
type Handler struct {
	cfg      *config.Config
	flashMgr *session.FlashManager
}

// NewHandlerは新しいHandlerを作成します
func NewHandler(cfg *config.Config, flashMgr *session.FlashManager) *Handler {
	return &Handler{
		cfg:      cfg,
		flashMgr: flashMgr,
	}
}
