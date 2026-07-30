package home

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	homepages "github.com/wikinoapp/wikino/go/internal/templates/pages/home"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Show はホーム画面を表示します (GET /home)
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)

	output, err := h.getHomeShowUC.Execute(ctx, usecase.GetHomeShowInput{
		UserID: user.ID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "ホーム画面の取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "home_show_title")

	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		GlobalNav: components.GlobalNavData{
			CurrentPageName: templates.PageNameHome,
			SignedIn:        true,
			UserAtname:      user.Atname,
		},

		BreadcrumbHeader: components.BreadcrumbHeaderData{
			MaxWidthClass: "max-w-3xl",
		},
	}

	content := homepages.Show(homepages.ShowPageData{
		ActiveSpaces: viewmodel.NewSpaces(output.ActiveSpaces),
		JoinedTopics: viewmodel.NewCardLinkTopics(output.JoinedTopics, output.CanCreatePageByTopic),
		DraftPages:   viewmodel.NewCardLinkDraftPages(output.DraftPages),
	})

	err = layouts.Default(layoutData, content).Render(ctx, w)
	if err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
