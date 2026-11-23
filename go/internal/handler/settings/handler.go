// Package settingsはユーザー設定のトップのHTTPハンドラーを提供します
package settings

import "github.com/wikinoapp/wikino/go/internal/config"

// Handlerはユーザー設定のトップを提供します
type Handler struct {
	cfg *config.Config
}

// NewHandlerはユーザー設定のトップのハンドラーを生成します
func NewHandler(cfg *config.Config) *Handler {
	return &Handler{cfg: cfg}
}
