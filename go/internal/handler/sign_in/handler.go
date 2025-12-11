// Package sign_in はログインページのハンドラーを提供します
package sign_in

import (
	"github.com/wikinoapp/wikino/go/internal/config"
)

// Handler はログインハンドラー
type Handler struct {
	cfg *config.Config
}

// NewHandler は新しいログインハンドラーを作成します
func NewHandler(cfg *config.Config) *Handler {
	return &Handler{
		cfg: cfg,
	}
}
