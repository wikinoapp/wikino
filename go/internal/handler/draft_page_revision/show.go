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

// Showはページ編集画面の差分モーダルに表示する、リビジョン差分のHTMLフラグメントを返します
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

	// 差分はUseCaseが返した2つのリビジョンからビューモデル層で計算する。
	diff := viewmodel.NewDraftPageRevisionDiff(output.Revision, output.PreviousRevision)

	// フラグメントはインラインの復元フォームを含むため、復元URLとCSRFトークンを渡す。
	restoreURL := string(templates.PageDraftPageRevisionRestorePath(viewmodel.NewSpaceIdentifier(spaceIdentifier), int32(pageNumber), string(output.Revision.ID)))

	if err := components.DraftPageRevisionDiff(components.DraftPageRevisionDiffData{
		Diff:       diff,
		RestoreURL: restoreURL,
		CSRFToken:  middleware.GetCSRFTokenFromContext(ctx),
		IsCurrent:  output.IsCurrent,
	}).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "下書きリビジョン差分のレンダリングに失敗", "error", err)
	}
}
