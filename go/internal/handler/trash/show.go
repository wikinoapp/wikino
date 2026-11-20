package trash

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/httperror"
	"github.com/wikinoapp/wikino/go/internal/httppagination"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	trashpages "github.com/wikinoapp/wikino/go/internal/templates/pages/trash"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// trashShowPageLimitはゴミ箱の一覧の1ページあたりの表示件数です。
const trashShowPageLimit = 100

// Showはスペースのゴミ箱の一覧を表示します (GET /s/{space_identifier}/trash)。
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	// SQLのoffsetがクエリのint32パラメーターに収まらないページは、UseCaseを呼ぶ前に拒否する。
	currentPage, ok := httppagination.ParsePageParam(r, trashShowPageLimit)
	if !ok {
		httperror.NotFound(w, r)
		return
	}

	output, err := h.getTrashShowUC.Execute(ctx, usecase.GetTrashShowInput{
		SpaceIdentifier: model.SpaceIdentifier(chi.URLParam(r, "space_identifier")),
		UserID:          user.ID,
		Page:            currentPage,
		PageLimit:       trashShowPageLimit,
		Now:             time.Now(),
	})
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	pagination := viewmodel.NewPagination(int(currentPage), output.TotalCount, trashShowPageLimit)
	if pagination.Current > pagination.Total {
		httperror.NotFound(w, r)
		return
	}

	spaceVM := viewmodel.NewSpace(output.Space)

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	titleKey := "trash_show_title"
	if currentPage > 1 {
		titleKey = "trash_show_paginated_title"
	}
	meta.SetTitleWithoutSuffix(ctx, titleKey, map[string]any{
		"SpaceName":  spaceVM.Name,
		"PageNumber": currentPage,
	})
	meta.CurrentSpaceIdentifier = spaceVM.Identifier

	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		GlobalNav: components.GlobalNavData{
			CurrentPageName: templates.PageNameTrashShow,
			SignedIn:        true,
			UserAtname:      user.Atname,
			SpaceIdentifier: spaceVM.Identifier,
		},

		BreadcrumbHeader: components.BreadcrumbHeaderData{
			MaxWidthClass: "max-w-3xl",
			Items: append(components.HomeBreadcrumbItems(ctx, true),
				components.BreadcrumbItem{
					Label: spaceVM.Name,
					Path:  templates.SpacePath(spaceVM.Identifier),
				},
				// この画面が現在地のため、経路はaria-currentを持つリンク無しの項目で締める。
				components.BreadcrumbItem{
					Label:     i18n.T(ctx, "trash_show_breadcrumb"),
					IsCurrent: true,
				},
			),
		},
	}

	content := trashpages.Show(trashpages.ShowData{
		CSRFToken:  middleware.GetCSRFTokenFromContext(ctx),
		Space:      spaceVM,
		Pages:      viewmodel.NewTrashedPages(output.Pages, output.TopicMap),
		Pagination: pagination,
	})

	// ログインしたメンバーだけが開く画面のため、検索エンジンに載せない。
	w.Header().Set("X-Robots-Tag", "noindex")
	if err := layouts.Default(layoutData, content).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleErrorはユースケースから返ったエラーを、それに応じたレスポンスへ変えます。
// ゴミ箱を見られない閲覧者には「見つからない」として答え、ゴミ箱があることを伝えません。
// Rails版も同じく404を返していました。
func (h *Handler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	ctx := r.Context()

	if ae := model.AsAppError(err); ae != nil {
		switch ae.Code {
		case model.AppErrCodeResourceNotFound, model.AppErrCodeForbidden:
			httperror.NotFound(w, r)
		default:
			slog.ErrorContext(ctx, ae.LogString())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	slog.ErrorContext(ctx, "ゴミ箱の一覧のデータの取得に失敗", "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
