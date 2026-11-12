package bulk_restored_page

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/httperror"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Createはゴミ箱で選んだページをまとめて復元します (POST /s/{space_identifier}/bulk_restored_pages)。
//
// 復元しても、できなくても、ゴミ箱の一覧へ戻して結果をフラッシュで伝えます。
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		slog.ErrorContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))
	trashPath := string(templates.TrashPath(viewmodel.NewSpaceIdentifier(spaceIdentifier)))

	_, err := h.restorePagesUC.Execute(ctx, usecase.RestorePagesInput{
		SpaceIdentifier: spaceIdentifier,
		UserID:          user.ID,
		PageIDs:         r.PostForm["page_ids"],
	})
	if err != nil {
		if ve := model.AsValidationError(err); ve != nil {
			if errs := ve.GetFieldErrors("page_ids"); len(errs) > 0 {
				h.flashMgr.SetError(w, errs[0])
			}
			http.Redirect(w, r, trashPath, http.StatusSeeOther)
			return
		}
		if ae := model.AsAppError(err); ae != nil {
			switch ae.Code {
			case model.AppErrCodeConflict:
				// 検証のあとに別の操作でページが動いた。何も戻していないことを伝え、選び直してもらう。
				slog.WarnContext(ctx, ae.LogString())
				h.flashMgr.SetError(w, ae.UserMsg)
				http.Redirect(w, r, trashPath, http.StatusSeeOther)
			case model.AppErrCodeResourceNotFound, model.AppErrCodeForbidden:
				// ゴミ箱からページを戻せない閲覧者には、ゴミ箱の一覧と同じく「見つからない」と答える。
				httperror.NotFound(w, r)
			default:
				slog.ErrorContext(ctx, ae.LogString())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}
		slog.ErrorContext(ctx, "ゴミ箱のページの一括復元に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_pages_restored"))
	http.Redirect(w, r, trashPath, http.StatusSeeOther)
}
