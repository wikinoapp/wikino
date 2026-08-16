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

	// The three related-page listings paginate independently on a full-page request.
	//
	// [Ja] フルページリクエストでは 3 種類の関連ページ一覧を独立してページングする。
	linkState, ok := parseRelatedPageState(r, viewmodel.PageLinkContextShow)
	if !ok {
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
		SpaceIdentifier:        spaceIdentifier,
		PageNumber:             int32(pageNumber),
		UserID:                 userID,
		LinkLimit:              viewmodel.LinkLimit,
		BacklinkLimit:          viewmodel.BacklinkLimit,
		PageBacklinkLimit:      viewmodel.PageBacklinkLimit,
		LinkPage:               linkState.LinkPage,
		LinkedPageNumber:       linkState.LinkedPageNumber,
		LinkedPageBacklinkPage: linkState.LinkedBacklinkPage,
		PageBacklinkPage:       linkState.PageBacklinkPage,
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

	if !relatedPageStateInRange(linkState, relatedPageCounts{
		LinkedTotalCount:      output.LinkedTotalCount,
		PageBacklinkCount:     output.PageBacklinkCount,
		LinkedPages:           output.LinkedPages,
		BacklinkCountByPageID: linkedPageBacklinkCounts(output.BacklinksPerPage),
	}) {
		handler.NotFound(w, r)
		return
	}

	pageVM := viewmodel.NewPageForShow(output.Page, output.FeaturedImageAttachment)
	spaceVM := viewmodel.NewSpace(output.Space)
	topicVM := viewmodel.NewTopic(output.Topic)

	// Build links from the stored identifier, not from the URL, so request casing cannot split one
	// representation across multiple canonical addresses.
	//
	// [Ja] URL ではなく保存済みの識別子からリンクを組み立て、リクエストの大文字小文字によって同じ
	// 表現が複数の canonical URL に分かれないようにする。
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
	// The related-list pagination parameters leave the page's own content untouched: the body, the
	// title and the description are the same for every combination of them, and only the secondary
	// listings under the body move. Keeping them out of the canonical URL collapses that whole
	// product of combinations onto the one address that carries the content, instead of declaring a
	// separate indexable address per combination.
	//
	// [Ja] 関連一覧のページネーションパラメータはページ自身の内容を変えない。本文・タイトル・説明文は
	// どの組み合わせでも同じで、動くのは本文の下の副次的な一覧だけである。これらを正規 URL から外す
	// ことで、組み合わせの直積ぶんの URL を、内容を持つ 1 つのアドレスへ集約する。組み合わせごとに
	// インデックス対象のアドレスを宣言しない。
	meta.OGURL = h.cfg.AppURL() + string(templates.PagePath(spaceIdentVM, viewmodel.PageNumber(pageVM.Number)))
	// A page with a cover image advertises it as the link preview. Pages without one keep the
	// site-wide default OGP image set by DefaultPageMeta.
	//
	// [Ja] アイキャッチ画像を持つページはその画像をリンクプレビューとして出す。持たないページは
	// DefaultPageMeta が設定したサイト共通の既定 OGP 画像を保つ。
	if attachmentID := pageVM.OGImageAttachmentID(); attachmentID != "" {
		meta.OGImage = h.cfg.AppURL() + string(templates.AttachmentOGImagePath(attachmentID))
	}
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

	backlinksPerPage := make(map[model.PageID]*viewmodel.PageSliceWithCount, len(output.BacklinksPerPage))
	for pageID, backlinks := range output.BacklinksPerPage {
		backlinksPerPage[pageID] = &viewmodel.PageSliceWithCount{
			Pages:       backlinks.Pages,
			TotalCount:  backlinks.TotalCount,
			CurrentPage: linkState.BacklinkPageFor(output.LinkedPages, pageID),
		}
	}

	linkData := viewmodel.BuildPageLinkData(viewmodel.BuildPageLinkDataInput{
		LinkedPages:         output.LinkedPages,
		LinkedTotalCount:    output.LinkedTotalCount,
		BacklinksPerPage:    backlinksPerPage,
		PageBacklinks:       output.PageBacklinks,
		PageBacklinkCount:   output.PageBacklinkCount,
		Topics:              output.LinkTopics,
		SpaceIdentifier:     output.Space.Identifier,
		PageNumber:          pageVM.Number,
		LinkedPageFirstPage: linkState.LinkPage,
		State:               linkState,
		// The screen is public, so the per-card edit link follows the viewer's own permission
		// rather than being shown to everyone.
		//
		// [Ja] 本画面は公開のため、各カードの編集リンクは全員に出すのではなく閲覧者自身の権限に従う。
		CanEdit: output.CanUpdatePage,
	})

	// The token is fetched only for a viewer holding page:trash. An already-trashed page passes it to
	// ShowData but does not render the trash form or hidden input. Viewers without page:trash,
	// including guests, receive an empty value, so their HTML carries no token.
	//
	// [Ja] page:trash を持つ閲覧者のときだけトークンを取得する。既にゴミ箱にあるページでは
	// ShowData にトークンを渡すが、ゴミ箱フォームと hidden input は描画しない。ゲストを含む
	// page:trash を持たない閲覧者には空文字を渡すため、その HTML にはトークンが載らない。
	var csrfToken string
	if output.CanTrashPage {
		csrfToken = middleware.GetCSRFTokenFromContext(ctx)
	}

	content := pagepages.Show(pagepages.ShowData{
		Page:          pageVM,
		Space:         spaceVM,
		IsTrashed:     output.IsTrashed,
		CanUpdatePage: output.CanUpdatePage,
		CanTrashPage:  output.CanTrashPage,
		CSRFToken:     csrfToken,
		LinkList:      linkData.LinkList,
		BacklinkList:  linkData.BacklinkList,
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
