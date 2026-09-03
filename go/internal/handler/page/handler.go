// Package page はページ関連のHTTPハンドラーを提供します
package page

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler はページハンドラー
type Handler struct {
	cfg               *config.Config
	flashMgr          *session.FlashManager
	getPageShowUC     *usecase.GetPageShowUsecase
	getPageDetailUC   *usecase.GetPageDetailUsecase
	getEditLinkDataUC *usecase.GetEditLinkDataUsecase
	publishPageUC     *usecase.PublishPageUsecase
	createPageUC      *usecase.CreatePageUsecase
}

// NewHandler は新しいページハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	flashMgr *session.FlashManager,
	getPageShowUC *usecase.GetPageShowUsecase,
	getPageDetailUC *usecase.GetPageDetailUsecase,
	getEditLinkDataUC *usecase.GetEditLinkDataUsecase,
	publishPageUC *usecase.PublishPageUsecase,
	createPageUC *usecase.CreatePageUsecase,
) *Handler {
	return &Handler{
		cfg:               cfg,
		flashMgr:          flashMgr,
		getPageShowUC:     getPageShowUC,
		getPageDetailUC:   getPageDetailUC,
		getEditLinkDataUC: getEditLinkDataUC,
		publishPageUC:     publishPageUC,
		createPageUC:      createPageUC,
	}
}
