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

// showBreadcrumbMaxWidthClassはページ表示画面の本文幅にパンくずを揃える。
const showBreadcrumbMaxWidthClass = "max-w-3xl"

// Showはページ表示画面を表示します (GETまたはHEAD /s/{space_identifier}/pages/{page_number})。
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))

	pageNumber, err := strconv.ParseInt(chi.URLParam(r, "page_number"), 10, 32)
	if err != nil {
		handler.NotFound(w, r)
		return
	}

	// フルページリクエストでは3種類の関連ページ一覧を独立してページングする。
	linkState, ok := parseRelatedPageState(r, viewmodel.PageLinkContextShow)
	if !ok {
		handler.NotFound(w, r)
		return
	}

	// ページ表示画面は未ログインでも閲覧できる (公開トピックのページのみ)。
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
		// UseCaseは「この閲覧者に見せてはいけない」と「存在しない」を同じコードで返すため、
		// どちらも同じ404レスポンスになり両者は区別されない。
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

	// URLではなく保存済みの識別子からリンクを組み立て、リクエストの大文字小文字によって同じ
	// 表現が複数のcanonical URLに分かれないようにする。
	spaceIdentVM := spaceVM.Identifier

	pageTitle := pageVM.DisplayTitle(ctx)
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitleWithoutSuffix(ctx, "page_show_title", map[string]any{
		"PageTitle": pageTitle,
		"SpaceName": output.Space.Name,
	})
	// 本文にテキストが無いページはサイト共通の既定の説明文を保つ。
	if description := pageVM.MetaDescription(); description != "" {
		meta.Description = description
	}
	// 関連一覧のページネーションパラメータはページ自身の内容を変えない。本文・タイトル・説明文は
	// どの組み合わせでも同じで、動くのは本文の下の副次的な一覧だけである。これらを正規URLから外す
	// ことで、組み合わせの直積ぶんのURLを、内容を持つ1つのアドレスへ集約する。組み合わせごとに
	// インデックス対象のアドレスを宣言しない。
	meta.OGURL = h.cfg.AppURL() + string(templates.PagePath(spaceIdentVM, viewmodel.PageNumber(pageVM.Number)))
	// アイキャッチ画像を持つページはその画像をリンクプレビューとして出す。持たないページは
	// DefaultPageMetaが設定したサイト共通の既定OGP画像を保つ。
	if attachmentID := pageVM.OGImageAttachmentID(); attachmentID != "" {
		meta.OGImage = h.cfg.AppURL() + string(templates.AttachmentOGImagePath(attachmentID))
	}
	// ページ表示画面は唯一の本文ページのため、サイト共通のwebsiteではなくarticleを宣言する。
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
		// 本画面は公開のため、各カードの編集リンクは全員に出すのではなく閲覧者自身の権限に従う。
		CanEdit: output.CanUpdatePage,
	})

	// page:trashを持つ閲覧者のときだけトークンを取得する。既にゴミ箱にあるページでは
	// ShowDataにトークンを渡すが、ゴミ箱フォームとhidden inputは描画しない。ゲストを含む
	// page:trashを持たない閲覧者には空文字を渡すため、そのHTMLにはトークンが載らない。
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
