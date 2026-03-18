// Package suggestion は編集提案関連のHTTPハンドラーを提供します
package suggestion

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/sidebar"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// Handler は編集提案ハンドラー
type Handler struct {
	cfg                        *config.Config
	flashMgr                   *session.FlashManager
	getSuggestionListUsecase   *usecase.GetSuggestionListUsecase
	getSuggestionDetailUsecase *usecase.GetSuggestionDetailUsecase
	getSuggestionDiffUsecase   *usecase.GetSuggestionDiffUsecase
	getSuggestionNewUsecase    *usecase.GetSuggestionNewUsecase
	createSuggestionUsecase    *usecase.CreateSuggestionUsecase
	sidebarHelper              *sidebar.Helper
	createValidator            *validator.SuggestionCreateValidator
}

// NewHandler は新しい編集提案ハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	flashMgr *session.FlashManager,
	getSuggestionListUsecase *usecase.GetSuggestionListUsecase,
	getSuggestionDetailUsecase *usecase.GetSuggestionDetailUsecase,
	getSuggestionDiffUsecase *usecase.GetSuggestionDiffUsecase,
	getSuggestionNewUsecase *usecase.GetSuggestionNewUsecase,
	createSuggestionUsecase *usecase.CreateSuggestionUsecase,
	sidebarHelper *sidebar.Helper,
	createValidator *validator.SuggestionCreateValidator,
) *Handler {
	return &Handler{
		cfg:                        cfg,
		flashMgr:                   flashMgr,
		getSuggestionListUsecase:   getSuggestionListUsecase,
		getSuggestionDetailUsecase: getSuggestionDetailUsecase,
		getSuggestionDiffUsecase:   getSuggestionDiffUsecase,
		getSuggestionNewUsecase:    getSuggestionNewUsecase,
		createSuggestionUsecase:    createSuggestionUsecase,
		sidebarHelper:              sidebarHelper,
		createValidator:            createValidator,
	}
}
