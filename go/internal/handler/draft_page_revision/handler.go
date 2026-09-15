// Package draft_page_revisionは下書きリビジョン関連のHTTPハンドラーを提供します
package draft_page_revision

import (
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerは下書きリビジョンハンドラー
type Handler struct {
	flashMgr                   *session.FlashManager
	manualSaveDraftPageUC      *usecase.ManualSaveDraftPageUsecase
	getDraftPageRevisionDiffUC *usecase.GetDraftPageRevisionDiffUsecase

	// getPageDetailUCは、ページ編集画面からの手動保存のOOBスワップレスポンス用に
	// 編集履歴カラム (リビジョン一覧と総件数) を再取得する。
	getPageDetailUC *usecase.GetPageDetailUsecase
}

// NewHandlerは新しい下書きリビジョンハンドラーを作成します
func NewHandler(
	flashMgr *session.FlashManager,
	manualSaveDraftPageUC *usecase.ManualSaveDraftPageUsecase,
	getDraftPageRevisionDiffUC *usecase.GetDraftPageRevisionDiffUsecase,
	getPageDetailUC *usecase.GetPageDetailUsecase,
) *Handler {
	return &Handler{
		flashMgr:                   flashMgr,
		manualSaveDraftPageUC:      manualSaveDraftPageUC,
		getDraftPageRevisionDiffUC: getDraftPageRevisionDiffUC,
		getPageDetailUC:            getPageDetailUC,
	}
}
