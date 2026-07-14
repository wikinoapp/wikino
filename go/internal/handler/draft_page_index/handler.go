// Package draft_page_index は下書き一覧画面のHTTPハンドラーを提供します
package draft_page_index

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler は下書き一覧ハンドラー
type Handler struct {
	cfg             *config.Config
	getDraftPagesUC *usecase.GetDraftPagesUsecase
}

// NewHandler は新しい下書き一覧ハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	getDraftPagesUC *usecase.GetDraftPagesUsecase,
) *Handler {
	return &Handler{
		cfg:             cfg,
		getDraftPagesUC: getDraftPagesUC,
	}
}
