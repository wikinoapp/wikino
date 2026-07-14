// Package home はホーム画面のHTTPハンドラーを提供します
package home

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler はホーム画面ハンドラー
type Handler struct {
	cfg           *config.Config
	getHomeShowUC *usecase.GetHomeShowUsecase
}

// NewHandler は新しいホーム画面ハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	getHomeShowUC *usecase.GetHomeShowUsecase,
) *Handler {
	return &Handler{
		cfg:           cfg,
		getHomeShowUC: getHomeShowUC,
	}
}
