// Package homeはホーム画面のHTTPハンドラーを提供します
package home

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはホーム画面ハンドラー
type Handler struct {
	cfg           *config.Config
	getHomeShowUC *usecase.GetHomeShowUsecase
}

// NewHandlerは新しいホーム画面ハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	getHomeShowUC *usecase.GetHomeShowUsecase,
) *Handler {
	return &Handler{
		cfg:           cfg,
		getHomeShowUC: getHomeShowUC,
	}
}
