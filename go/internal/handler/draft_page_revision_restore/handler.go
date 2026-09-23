// Package draft_page_revision_restoreは、下書きページを選択されたリビジョンの内容に
// 復元するHTTPハンドラーを提供します。
package draft_page_revision_restore

import (
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerは下書きページリビジョン復元ハンドラー。
type Handler struct {
	flashMgr                   *session.FlashManager
	restoreDraftPageRevisionUC *usecase.RestoreDraftPageRevisionUsecase
}

// NewHandlerは新しいHandlerを作成します。
func NewHandler(
	flashMgr *session.FlashManager,
	restoreDraftPageRevisionUC *usecase.RestoreDraftPageRevisionUsecase,
) *Handler {
	return &Handler{
		flashMgr:                   flashMgr,
		restoreDraftPageRevisionUC: restoreDraftPageRevisionUC,
	}
}
