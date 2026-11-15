package settings

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	settingspages "github.com/wikinoapp/wikino/go/internal/templates/pages/settings"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Showはユーザー設定のトップを表示します (GET /settings)。
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "settings_show_title")

	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		GlobalNav: components.GlobalNavData{
			CurrentPageName: templates.PageNameSettings,
			SignedIn:        true,
			UserAtname:      user.Atname,
		},

		BreadcrumbHeader: components.BreadcrumbHeaderData{
			MaxWidthClass: "max-w-2xl",
		},
	}

	// 設定の画面はログインした本人のための画面のため、検索エンジンにインデックスさせない
	w.Header().Set("X-Robots-Tag", "noindex")

	content := settingspages.Show(settingspages.ShowData{
		CSRFToken: middleware.GetCSRFTokenFromContext(ctx),
	})
	if err := layouts.Default(layoutData, content).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
