package page

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	pagepages "github.com/wikinoapp/wikino/go/internal/templates/pages/page"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// showBreadcrumbMaxWidthClass lines the page detail screen's breadcrumb up with its body.
//
// [Ja] showBreadcrumbMaxWidthClass はページ表示画面の本文幅にパンくずを揃える。
const showBreadcrumbMaxWidthClass = "max-w-3xl"

// Show renders the page detail screen (GET /s/{space_identifier}/pages/{page_number}).
//
// [Ja] Show はページ表示画面を表示します (GET /s/{space_identifier}/pages/{page_number})。
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))

	pageNumber, err := strconv.ParseInt(chi.URLParam(r, "page_number"), 10, 32)
	if err != nil {
		handler.NotFound(w, r)
		return
	}

	// The page detail is viewable without signing in (public-topic pages only).
	//
	// [Ja] ページ表示画面は未ログインでも閲覧できる (公開トピックのページのみ)。
	user := middleware.UserFromContext(ctx)
	signedIn := user != nil
	var userID *model.UserID
	if user != nil {
		userID = &user.ID
	}

	output, err := h.getPageShowUC.Execute(ctx, usecase.GetPageShowInput{
		SpaceIdentifier: spaceIdentifier,
		PageNumber:      int32(pageNumber),
		UserID:          userID,
	})
	if err != nil {
		// The usecase reports "must not be shown to this viewer" and "does not exist" with the same
		// code, so both end up as the same 404 response and the two stay indistinguishable.
		//
		// [Ja] UseCase は「この閲覧者に見せてはいけない」と「存在しない」を同じコードで返すため、
		// どちらも同じ 404 レスポンスになり両者は区別されない。
		if ae := model.AsAppError(err); ae != nil {
			switch ae.Code {
			case model.AppErrCodeResourceNotFound, model.AppErrCodeForbidden:
				handler.NotFound(w, r)
			default:
				slog.ErrorContext(ctx, ae.LogString())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}
		slog.ErrorContext(ctx, "ページ表示画面のデータ取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	pageVM := viewmodel.NewPageForShow(output.Page)
	spaceVM := viewmodel.NewSpace(output.Space)
	topicVM := viewmodel.NewTopic(output.Topic)

	// Build links from the stored identifier, not from the URL, so that the canonical URL collapses
	// to one address per page.
	//
	// [Ja] URL ではなく保存済みの識別子からリンクを組み立て、正規 URL を 1 ページ 1 アドレスに
	// 集約する。
	spaceIdentVM := spaceVM.Identifier

	pageTitle := pageVM.DisplayTitle(ctx)
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitleWithoutSuffix(ctx, "page_show_title", map[string]any{
		"PageTitle": pageTitle,
		"SpaceName": output.Space.Name,
	})
	// A page with no body text keeps the site-wide default description.
	//
	// [Ja] 本文にテキストが無いページはサイト共通の既定の説明文を保つ。
	if description := pageVM.MetaDescription(); description != "" {
		meta.Description = description
	}
	meta.OGURL = h.cfg.AppURL() + string(templates.PagePath(spaceIdentVM, viewmodel.PageNumber(pageVM.Number)))
	// The page detail is the one long-form content screen, so it declares itself as an article
	// instead of taking the site-wide website type.
	//
	// [Ja] ページ表示画面は唯一の本文ページのため、サイト共通の website ではなく article を宣言する。
	meta.OGType = "article"
	meta.CurrentSpaceIdentifier = spaceIdentVM

	breadcrumbHeader := pageBreadcrumbHeaderData(ctx, spaceVM, topicVM, showBreadcrumbMaxWidthClass, signedIn)
	breadcrumbHeader.Items = append(breadcrumbHeader.Items, components.BreadcrumbItem{
		Label:     pageTitle,
		IsCurrent: true,
	})
	breadcrumbHeader.StructuredDataBaseURL = h.cfg.AppURL()

	content := pagepages.Show(pagepages.ShowData{
		Page:      pageVM,
		Space:     spaceVM,
		IsTrashed: output.IsTrashed,
	})

	var userAtname string
	if user != nil {
		userAtname = user.Atname
	}

	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		GlobalNav: components.GlobalNavData{
			CurrentPageName: templates.PageNamePageShow,
			SignedIn:        signedIn,
			UserAtname:      userAtname,
			SpaceIdentifier: spaceIdentVM,
		},

		BreadcrumbHeader: breadcrumbHeader,
	}

	if err := layouts.Default(layoutData, content).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
