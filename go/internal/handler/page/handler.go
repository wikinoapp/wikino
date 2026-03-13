// Package page はページ関連のHTTPハンドラーを提供します
package page

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/sidebar"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// Handler はページハンドラー
type Handler struct {
	cfg               *config.Config
	flashMgr          *session.FlashManager
	getPageDetailUC   *usecase.GetPageDetailUsecase
	getEditLinkDataUC *usecase.GetEditLinkDataUsecase
	publishPageUC     *usecase.PublishPageUsecase
	sidebarHelper     *sidebar.Helper
	updateValidator   *validator.PageUpdateValidator
}

// NewHandler は新しいページハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	flashMgr *session.FlashManager,
	getPageDetailUC *usecase.GetPageDetailUsecase,
	getEditLinkDataUC *usecase.GetEditLinkDataUsecase,
	publishPageUC *usecase.PublishPageUsecase,
	sidebarHelper *sidebar.Helper,
	updateValidator *validator.PageUpdateValidator,
) *Handler {
	return &Handler{
		cfg:               cfg,
		flashMgr:          flashMgr,
		getPageDetailUC:   getPageDetailUC,
		getEditLinkDataUC: getEditLinkDataUC,
		publishPageUC:     publishPageUC,
		sidebarHelper:     sidebarHelper,
		updateValidator:   updateValidator,
	}
}
