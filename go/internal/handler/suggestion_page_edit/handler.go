// Package suggestion_page_edit は編集提案ページの編集開始関連のHTTPハンドラーを提供します
package suggestion_page_edit

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler は編集提案ページの編集開始ハンドラー
type Handler struct {
	cfg                            *config.Config
	flashMgr                       *session.FlashManager
	getSuggestionDetailUsecase     *usecase.GetSuggestionDetailUsecase
	startSuggestionPageEditUsecase *usecase.StartSuggestionPageEditUsecase
}

// NewHandler は新しい編集提案ページの編集開始ハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	flashMgr *session.FlashManager,
	getSuggestionDetailUsecase *usecase.GetSuggestionDetailUsecase,
	startSuggestionPageEditUsecase *usecase.StartSuggestionPageEditUsecase,
) *Handler {
	return &Handler{
		cfg:                            cfg,
		flashMgr:                       flashMgr,
		getSuggestionDetailUsecase:     getSuggestionDetailUsecase,
		startSuggestionPageEditUsecase: startSuggestionPageEditUsecase,
	}
}
