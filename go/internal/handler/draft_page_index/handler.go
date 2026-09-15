// Package draft_page_indexは下書き一覧画面のHTTPハンドラーを提供します
package draft_page_index

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerは下書き一覧ハンドラー
type Handler struct {
	cfg             *config.Config
	getDraftPagesUC *usecase.GetDraftPagesUsecase
}

// NewHandlerは新しい下書き一覧ハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	getDraftPagesUC *usecase.GetDraftPagesUsecase,
) *Handler {
	return &Handler{
		cfg:             cfg,
		getDraftPagesUC: getDraftPagesUC,
	}
}
