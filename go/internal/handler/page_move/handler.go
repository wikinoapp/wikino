// Package page_move はページ移動関連のHTTPハンドラーを提供します
package page_move

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/sidebar"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// Handler はページ移動ハンドラー
type Handler struct {
	cfg               *config.Config
	flashMgr          *session.FlashManager
	getPageMoveDataUC *usecase.GetPageMoveDataUsecase
	movePageUC        *usecase.MovePageUsecase
	sidebarHelper     *sidebar.Helper
	createValidator   *validator.PageMoveCreateValidator
}

// NewHandler は新しいページ移動ハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	flashMgr *session.FlashManager,
	getPageMoveDataUC *usecase.GetPageMoveDataUsecase,
	movePageUC *usecase.MovePageUsecase,
	sidebarHelper *sidebar.Helper,
	createValidator *validator.PageMoveCreateValidator,
) *Handler {
	return &Handler{
		cfg:               cfg,
		flashMgr:          flashMgr,
		getPageMoveDataUC: getPageMoveDataUC,
		movePageUC:        movePageUC,
		sidebarHelper:     sidebarHelper,
		createValidator:   createValidator,
	}
}
