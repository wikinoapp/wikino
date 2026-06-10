// Package draft_page_revision_restore provides the HTTP handler that restores a draft page to
// the content of a selected revision.
//
// [Ja] Package draft_page_revision_restore は、下書きページを選択されたリビジョンの内容に
// 復元する HTTP ハンドラーを提供します。
package draft_page_revision_restore

import (
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler is the draft page revision restore handler.
// [Ja] Handler は下書きページリビジョン復元ハンドラー。
type Handler struct {
	flashMgr                   *session.FlashManager
	restoreDraftPageRevisionUC *usecase.RestoreDraftPageRevisionUsecase
}

// NewHandler creates a new Handler.
// [Ja] NewHandler は新しい Handler を作成します。
func NewHandler(
	flashMgr *session.FlashManager,
	restoreDraftPageRevisionUC *usecase.RestoreDraftPageRevisionUsecase,
) *Handler {
	return &Handler{
		flashMgr:                   flashMgr,
		restoreDraftPageRevisionUC: restoreDraftPageRevisionUC,
	}
}
