// Package profileはプロフィールの画面のHTTPハンドラーを提供します
package profile

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはプロフィールの画面を提供します
type Handler struct {
	cfg              *config.Config
	getProfileShowUC *usecase.GetProfileShowUsecase
}

// NewHandlerはプロフィールの画面のハンドラーを生成します
func NewHandler(
	cfg *config.Config,
	getProfileShowUC *usecase.GetProfileShowUsecase,
) *Handler {
	return &Handler{
		cfg:              cfg,
		getProfileShowUC: getProfileShowUC,
	}
}
