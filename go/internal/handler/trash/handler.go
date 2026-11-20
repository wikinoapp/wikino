// Package trashはスペースのゴミ箱の一覧のHTTPハンドラーを提供します。
package trash

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはスペースのゴミ箱の一覧を提供します。
type Handler struct {
	cfg            *config.Config
	getTrashShowUC *usecase.GetTrashShowUsecase
}

// NewHandlerはゴミ箱の一覧のハンドラーを生成します
func NewHandler(
	cfg *config.Config,
	getTrashShowUC *usecase.GetTrashShowUsecase,
) *Handler {
	return &Handler{
		cfg:            cfg,
		getTrashShowUC: getTrashShowUC,
	}
}
