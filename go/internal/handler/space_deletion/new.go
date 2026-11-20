package space_deletion

import (
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	spacepages "github.com/wikinoapp/wikino/go/internal/templates/pages/space"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Newはスペースの削除の確認画面を表示します (GET /s/{space_identifier}/settings/deletion/new)。
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	spaceIdentifier, user, ok := requestTarget(w, r)
	if !ok {
		return
	}

	output, err := h.getSpaceDeletionNewUC.Execute(ctx, usecase.GetSpaceDeletionNewInput{
		SpaceIdentifier: spaceIdentifier,
		UserID:          user.ID,
	})
	if err != nil {
		h.handleError(w, r, err, "スペースの削除の確認画面のデータの取得に失敗")
		return
	}

	h.render(w, r, user, http.StatusOK, spacepages.DeletionNewData{
		CSRFToken: middleware.GetCSRFTokenFromContext(ctx),
		Space:     viewmodel.NewSpace(output.Space),
	})
}
