// Package api_referenceは、公開Web APIの人間向けのAPIリファレンスのハンドラーを提供する
package api_reference

import (
	"github.com/wikinoapp/wikino/go/internal/config"
)

// HandlerはAPIリファレンスのHTTPハンドラー
type Handler struct {
	cfg *config.Config
}

// NewHandlerは新しいHandlerを作成する
func NewHandler(cfg *config.Config) *Handler {
	return &Handler{cfg: cfg}
}
