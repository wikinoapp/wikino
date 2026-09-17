// Package page_locationはページロケーション関連のHTTPハンドラーを提供します
package page_location

import (
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはページロケーションハンドラー
type Handler struct {
	getPageLocationsUC *usecase.GetPageLocationsUsecase
}

// NewHandlerは新しいページロケーションハンドラーを作成します
func NewHandler(
	getPageLocationsUC *usecase.GetPageLocationsUsecase,
) *Handler {
	return &Handler{
		getPageLocationsUC: getPageLocationsUC,
	}
}
