// Package draft_page_index は下書き一覧画面のHTTPハンドラーを提供します
package draft_page_index

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/sidebar"
)

// Handler は下書き一覧ハンドラー
type Handler struct {
	cfg           *config.Config
	flashMgr      *session.FlashManager
	draftPageRepo *repository.DraftPageRepository
	sidebarHelper *sidebar.Helper
}

// NewHandler は新しい下書き一覧ハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	flashMgr *session.FlashManager,
	draftPageRepo *repository.DraftPageRepository,
	sidebarHelper *sidebar.Helper,
) *Handler {
	return &Handler{
		cfg:           cfg,
		flashMgr:      flashMgr,
		draftPageRepo: draftPageRepo,
		sidebarHelper: sidebarHelper,
	}
}
