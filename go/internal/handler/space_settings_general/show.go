package space_settings_general

import (
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	spacepages "github.com/wikinoapp/wikino/go/internal/templates/pages/space"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Showはスペースの一般設定を表示します (GET /s/{space_identifier}/settings/general)。
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	spaceIdentifier, user, ok := requestTarget(w, r)
	if !ok {
		return
	}

	output, err := h.getSpaceSettingsGeneralUC.Execute(ctx, usecase.GetSpaceSettingsGeneralInput{
		SpaceIdentifier: spaceIdentifier,
		UserID:          user.ID,
	})
	if err != nil {
		h.handleError(w, r, err, "スペースの一般設定のデータの取得に失敗")
		return
	}

	h.render(w, r, user, spacepages.SettingsGeneralData{
		CSRFToken: middleware.GetCSRFTokenFromContext(ctx),
		Space:     viewmodel.NewSpace(output.Space),
		Fields: spacepages.FormFieldsData{
			Identifier: string(output.Space.Identifier),
			Name:       output.Space.Name,
		},
	})
}
