package space

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	spacepages "github.com/wikinoapp/wikino/go/internal/templates/pages/space"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Createはスペースを作成します (POST /spaces)
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

	input := usecase.CreateSpaceInput{
		UserID:     user.ID,
		Identifier: r.FormValue("identifier"),
		Name:       r.FormValue("name"),
	}

	output, err := h.createSpaceUC.Execute(ctx, input)
	if err != nil {
		if ve := model.AsValidationError(err); ve != nil {
			h.renderNewForm(w, r, user, http.StatusUnprocessableEntity, spacepages.NewData{
				CSRFToken:  middleware.GetCSRFTokenFromContext(ctx),
				FormErrors: ve,
				Identifier: input.Identifier,
				Name:       input.Name,
			})
			return
		}

		slog.ErrorContext(ctx, "スペースの作成に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_space_created"))
	http.Redirect(w, r, string(templates.SpacePath(viewmodel.NewSpaceIdentifier(output.Space.Identifier))), http.StatusSeeOther)
}
