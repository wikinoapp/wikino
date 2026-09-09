// Package space provides HTTP handlers for the space detail page.
// [Ja] Package space はスペース詳細画面の HTTP ハンドラーを提供します。
package space

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler is the space detail page handler.
// [Ja] Handler はスペース詳細画面のハンドラーです。
type Handler struct {
	cfg            *config.Config
	getSpaceShowUC *usecase.GetSpaceShowUsecase
}

// NewHandler creates a space detail page handler.
// [Ja] NewHandler は新しいスペース詳細画面のハンドラーを作成します。
func NewHandler(
	cfg *config.Config,
	getSpaceShowUC *usecase.GetSpaceShowUsecase,
) *Handler {
	return &Handler{
		cfg:            cfg,
		getSpaceShowUC: getSpaceShowUC,
	}
}
