// Package spaceはスペース詳細画面のHTTPハンドラーを提供します。
package space

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはスペース詳細画面のハンドラーです。
type Handler struct {
	cfg            *config.Config
	getSpaceShowUC *usecase.GetSpaceShowUsecase
}

// NewHandlerは新しいスペース詳細画面のハンドラーを作成します。
func NewHandler(
	cfg *config.Config,
	getSpaceShowUC *usecase.GetSpaceShowUsecase,
) *Handler {
	return &Handler{
		cfg:            cfg,
		getSpaceShowUC: getSpaceShowUC,
	}
}
