package welcome

import (
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	"github.com/wikinoapp/wikino/go/internal/templates/pages/welcome"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Show はトップページを表示します (GET /)
// ログイン済みの場合は /home にリダイレクトします
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// ログイン済みの場合は /home にリダイレクト
	if user := middleware.UserFromContext(ctx); user != nil {
		http.Redirect(w, r, "/home", http.StatusSeeOther)
		return
	}

	// ページメタデータを作成
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitleWithoutSuffix(ctx, "welcome_title")
	meta.Description = i18n.T(ctx, "welcome_description")

	// フラッシュメッセージを取得
	flash := h.flashMgr.GetFlash(w, r)

	// テンプレートをレンダリング
	layoutData := layouts.DefaultLayoutData{
		Meta:  meta,
		Flash: flash,
		Sidebar: components.SidebarData{
			CurrentPageName: templates.PageNameWelcome,
		},
	}
	pageData := welcome.ShowPageData{}

	if err := layouts.Default(layoutData, welcome.Show(pageData)).Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
