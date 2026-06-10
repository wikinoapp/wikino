package draft_page_revision

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Show returns the revision diff as an HTML fragment rendered into the diff modal on the page
// editor (GET /s/{space_identifier}/pages/{page_number}/draft_page_revisions/{draft_page_revision_id}).
//
// [Ja] Show はページ編集画面の差分モーダルに表示する、リビジョン差分の HTML フラグメントを返します
// (GET /s/{space_identifier}/pages/{page_number}/draft_page_revisions/{draft_page_revision_id})。
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
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

	output, err := h.getDraftPageRevisionDiffUC.Execute(ctx, usecase.GetDraftPageRevisionDiffInput{
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

		slog.ErrorContext(ctx, "下書きリビジョン差分の取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// The diff is computed in the view-model layer from the two revisions the UseCase returned.
	// [Ja] 差分は UseCase が返した 2 つのリビジョンからビューモデル層で計算する。
	diff := viewmodel.NewDraftPageRevisionDiff(output.Revision, output.PreviousRevision)

	// The fragment includes the inline restore form, so pass the restore URL and the CSRF token.
	// [Ja] フラグメントはインラインの復元フォームを含むため、復元 URL と CSRF トークンを渡す。
	restoreURL := string(templates.PageDraftPageRevisionRestorePath(viewmodel.NewSpaceIdentifier(spaceIdentifier), int32(pageNumber), string(output.Revision.ID)))

	if err := components.DraftPageRevisionDiff(components.DraftPageRevisionDiffData{
		Diff:       diff,
		RestoreURL: restoreURL,
		CSRFToken:  middleware.GetCSRFTokenFromContext(ctx),
	}).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "下書きリビジョン差分のレンダリングに失敗", "error", err)
	}
}
