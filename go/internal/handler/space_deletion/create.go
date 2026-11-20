package space_deletion

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	spacepages "github.com/wikinoapp/wikino/go/internal/templates/pages/space"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Createはスペースを削除し、ホームへリダイレクトします (POST /s/{space_identifier}/settings/deletion)。
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	spaceIdentifier, user, ok := requestTarget(w, r)
	if !ok {
		return
	}

	if err := r.ParseForm(); err != nil {
		slog.ErrorContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	spaceName := r.FormValue("space_name")

	output, err := h.discardSpaceUC.Execute(ctx, usecase.DiscardSpaceInput{
		SpaceIdentifier: spaceIdentifier,
		UserID:          user.ID,
		SpaceName:       spaceName,
	})
	if err != nil {
		if ve := model.AsValidationError(err); ve != nil && output != nil {
			h.render(w, r, user, http.StatusUnprocessableEntity, spacepages.DeletionNewData{
				CSRFToken:  middleware.GetCSRFTokenFromContext(ctx),
				Space:      viewmodel.NewSpace(output.Space),
				SpaceName:  spaceName,
				FormErrors: ve,
			})
			return
		}
		h.handleError(w, r, err, "スペースの削除に失敗")
		return
	}

	// 削除したスペースの画面はどれも見つからなくなるため、Rails版と同じくホームへ戻す。
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_space_deleted"))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
