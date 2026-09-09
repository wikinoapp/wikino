// Package suggestion は編集提案関連のHTTPハンドラーを提供します
package suggestion

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler は編集提案ハンドラー
type Handler struct {
	cfg                        *config.Config
	flashMgr                   *session.FlashManager
	getSuggestionListUsecase   *usecase.GetSuggestionListUsecase
	getSuggestionDetailUsecase *usecase.GetSuggestionDetailUsecase
	getSuggestionEditUsecase   *usecase.GetSuggestionEditUsecase
	getSuggestionNewUsecase    *usecase.GetSuggestionNewUsecase
	createSuggestionUsecase    *usecase.CreateSuggestionUsecase
	updateSuggestionUsecase    *usecase.UpdateSuggestionUsecase
}

// NewHandler は新しい編集提案ハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	flashMgr *session.FlashManager,
	getSuggestionListUsecase *usecase.GetSuggestionListUsecase,
	getSuggestionDetailUsecase *usecase.GetSuggestionDetailUsecase,
	getSuggestionEditUsecase *usecase.GetSuggestionEditUsecase,
	getSuggestionNewUsecase *usecase.GetSuggestionNewUsecase,
	createSuggestionUsecase *usecase.CreateSuggestionUsecase,
	updateSuggestionUsecase *usecase.UpdateSuggestionUsecase,
) *Handler {
	return &Handler{
		cfg:                        cfg,
		flashMgr:                   flashMgr,
		getSuggestionListUsecase:   getSuggestionListUsecase,
		getSuggestionDetailUsecase: getSuggestionDetailUsecase,
		getSuggestionEditUsecase:   getSuggestionEditUsecase,
		getSuggestionNewUsecase:    getSuggestionNewUsecase,
		createSuggestionUsecase:    createSuggestionUsecase,
		updateSuggestionUsecase:    updateSuggestionUsecase,
	}
}
