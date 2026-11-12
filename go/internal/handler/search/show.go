package search

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	searchpages "github.com/wikinoapp/wikino/go/internal/templates/pages/search"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Showは検索の画面を表示します (GET /search)。
//
// キーワードに誤りがあっても、画面を200で返してエラーを欄の下に出します。検索の画面はキーワードを
// 送るたびに開き直すもので、送信の失敗ではないためです (Rails版も同じく200を返していました)。
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	data := searchpages.ShowData{Query: query}

	output, err := h.getSearchShowUC.Execute(ctx, usecase.GetSearchShowInput{
		UserID: user.ID,
		Query:  query,
	})
	if err != nil {
		ve := model.AsValidationError(err)
		if ve == nil {
			slog.ErrorContext(ctx, "ページの検索に失敗", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		data.FormErrors = ve
	} else {
		data.Searched = output.Searched
		data.HasSpaceFilter = output.HasSpaceFilter
		data.Pages = viewmodel.NewSearchResultPages(output.Pages, output.TopicMap)
	}

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "search_show_title")

	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		GlobalNav: components.GlobalNavData{
			CurrentPageName: templates.PageNameSearch,
			SignedIn:        true,
			UserAtname:      user.Atname,
		},

		BreadcrumbHeader: components.BreadcrumbHeaderData{
			MaxWidthClass: "max-w-3xl",
			Items: append(components.HomeBreadcrumbItems(ctx, true),
				// この画面が現在地のため、経路はaria-currentを持つリンク無しの項目で締める。
				components.BreadcrumbItem{
					Label:     i18n.T(ctx, "search_show_breadcrumb"),
					IsCurrent: true,
				},
			),
		},
	}

	// ログインした閲覧者だけが開く画面で、サイト内検索の結果でもあるため、検索エンジンに載せない。
	w.Header().Set("X-Robots-Tag", "noindex")
	if err := layouts.Default(layoutData, searchpages.Show(data)).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
