package space

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	spacepages "github.com/wikinoapp/wikino/go/internal/templates/pages/space"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Newはスペース作成フォームを表示します (GET /spaces/new)
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	h.renderNewForm(w, r, user, http.StatusOK, spacepages.NewData{
		CSRFToken: middleware.GetCSRFTokenFromContext(ctx),
	})
}

// renderNewFormはスペース作成フォームをstatusで描画します。ログインした利用者だけが使う画面の
// ため、検索エンジンにインデックスさせません。
func (h *Handler) renderNewForm(w http.ResponseWriter, r *http.Request, user *model.User, status int, data spacepages.NewData) {
	ctx := r.Context()

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "space_new_title")

	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		GlobalNav: components.GlobalNavData{
			CurrentPageName: templates.PageNameSpaceNew,
			SignedIn:        true,
			UserAtname:      user.Atname,
		},

		BreadcrumbHeader: components.BreadcrumbHeaderData{
			MaxWidthClass: "max-w-2xl",
			Items: append(components.HomeBreadcrumbItems(ctx, true),
				components.BreadcrumbItem{
					Label:     i18n.T(ctx, "space_new_breadcrumb"),
					IsCurrent: true,
				},
			),
		},
	}

	w.Header().Set("X-Robots-Tag", "noindex")
	w.WriteHeader(status)
	if err := layouts.Default(layoutData, spacepages.New(data)).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
