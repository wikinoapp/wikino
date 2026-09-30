// Package spaceはスペースの詳細画面と作成のHTTPハンドラーを提供します。
package space

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはスペースのハンドラーです。
type Handler struct {
	cfg            *config.Config
	flashMgr       *session.FlashManager
	getSpaceShowUC *usecase.GetSpaceShowUsecase
	createSpaceUC  *usecase.CreateSpaceUsecase
}

// NewHandlerは新しいスペースのハンドラーを作成します。
func NewHandler(
	cfg *config.Config,
	flashMgr *session.FlashManager,
	getSpaceShowUC *usecase.GetSpaceShowUsecase,
	createSpaceUC *usecase.CreateSpaceUsecase,
) *Handler {
	return &Handler{
		cfg:            cfg,
		flashMgr:       flashMgr,
		getSpaceShowUC: getSpaceShowUC,
		createSpaceUC:  createSpaceUC,
	}
}
