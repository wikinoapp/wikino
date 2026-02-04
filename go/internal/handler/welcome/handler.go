// Package welcome はトップページ（ウェルカムページ）のハンドラーを提供します
package welcome

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/session"
)

// Handler はトップページ関連のHTTPハンドラーです
type Handler struct {
	cfg      *config.Config
	flashMgr *session.FlashManager
}

// NewHandler は新しいHandlerを作成します
func NewHandler(cfg *config.Config, flashMgr *session.FlashManager) *Handler {
	return &Handler{
		cfg:      cfg,
		flashMgr: flashMgr,
	}
}
