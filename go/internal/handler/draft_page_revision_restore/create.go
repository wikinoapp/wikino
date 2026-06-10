package draft_page_revision_restore

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Create restores the draft to the selected revision's content and redirects back to the page
// editor with a flash message
// (POST /s/{space_identifier}/pages/{page_number}/draft_page_revisions/{draft_page_revision_id}/restore).
//
// [Ja] Create は下書きを選択されたリビジョンの内容に復元し、フラッシュメッセージ付きで
// ページ編集画面へリダイレクトします
// (POST /s/{space_identifier}/pages/{page_number}/draft_page_revisions/{draft_page_revision_id}/restore)。
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))
	pageNumberStr := chi.URLParam(r, "page_number")
	revisionID := model.DraftPageRevisionID(chi.URLParam(r, "draft_page_revision_id"))

	pageNumber, err := strconv.ParseInt(pageNumberStr, 10, 32)
	if err != nil {
		handler.NotFound(w, r)
		return
	}

	_, err = h.restoreDraftPageRevisionUC.Execute(ctx, usecase.RestoreDraftPageRevisionInput{
		SpaceIdentifier: spaceIdentifier,
		PageNumber:      int32(pageNumber),
		RevisionID:      revisionID,
		UserID:          user.ID,
	})
	if err != nil {
		if ae := model.AsAppError(err); ae != nil {
			switch ae.Code {
			case model.AppErrCodeResourceNotFound, model.AppErrCodeForbidden:
				handler.NotFound(w, r)
			default:
				slog.ErrorContext(ctx, ae.LogString())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		slog.ErrorContext(ctx, "下書きリビジョンの復元に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Redirect back to the page editor so the editor reloads with the restored content.
	// [Ja] 復元後の内容でエディタが再読み込みされるよう、ページ編集画面へリダイレクトする。
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_draft_page_revision_restored"))
	editPath := string(templates.PageEditPath(viewmodel.NewSpaceIdentifier(spaceIdentifier), int32(pageNumber)))
	http.Redirect(w, r, editPath, http.StatusSeeOther)
}
