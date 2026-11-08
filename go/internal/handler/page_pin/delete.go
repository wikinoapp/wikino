package page_pin

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/httperror"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Deleteはページのピンを外します (DELETE /s/{space_identifier}/pages/{page_number}/pin)。
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))

	pageNumber, err := strconv.ParseInt(chi.URLParam(r, "page_number"), 10, 32)
	if err != nil {
		httperror.NotFound(w, r)
		return
	}

	output, err := h.unpinPageUC.Execute(ctx, usecase.UnpinPageInput{
		SpaceIdentifier: spaceIdentifier,
		PageNumber:      int32(pageNumber),
		UserID:          user.ID,
	})
	if err != nil {
		writeError(w, r, err, "ページのピン留めの解除に失敗")
		return
	}

	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_page_unpinned"))
	redirectToPage(w, r, output.Space, int32(pageNumber))
}
