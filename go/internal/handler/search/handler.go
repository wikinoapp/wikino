// Package searchは検索の画面のHTTPハンドラーを提供します。
package search

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerは検索の画面を提供します。
type Handler struct {
	cfg             *config.Config
	getSearchShowUC *usecase.GetSearchShowUsecase
}

// NewHandlerは検索の画面のハンドラーを生成します
func NewHandler(
	cfg *config.Config,
	getSearchShowUC *usecase.GetSearchShowUsecase,
) *Handler {
	return &Handler{
		cfg:             cfg,
		getSearchShowUC: getSearchShowUC,
	}
}
