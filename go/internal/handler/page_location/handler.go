// Package page_location はページロケーション関連のHTTPハンドラーを提供します
package page_location

import (
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler はページロケーションハンドラー
type Handler struct {
	getPageLocationsUC *usecase.GetPageLocationsUsecase
}

// NewHandler は新しいページロケーションハンドラーを作成します
func NewHandler(
	getPageLocationsUC *usecase.GetPageLocationsUsecase,
) *Handler {
	return &Handler{
		getPageLocationsUC: getPageLocationsUC,
	}
}
