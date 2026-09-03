package space

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/httppagination"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	spacepages "github.com/wikinoapp/wikino/go/internal/templates/pages/space"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// spaceShowPageLimit is the number of regular (non-pinned) pages shown per page.
// [Ja] spaceShowPageLimit は通常ページ (ピン留めなし) の 1 ページあたりの表示件数です。
const spaceShowPageLimit = 100

// Show renders the space detail page (GET /s/{space_identifier}).
// [Ja] Show はスペース詳細画面を表示します (GET /s/{space_identifier})。
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))

	// Parse the pagination parameter. A page whose SQL offset cannot fit the query's int32
	// parameter is rejected before invoking the usecase; the total-page check below handles every
	// smaller out-of-range value.
	//
	// [Ja] ページネーションパラメータを取得する。SQL offset がクエリの int32 パラメータに収まらない
	// ページは UseCase 呼び出し前に拒否する。これより小さい範囲外値は後段の総ページ数チェックで
	// 処理する。
	currentPage, ok := httppagination.ParsePageParam(r, spaceShowPageLimit)
	if !ok {
		handler.NotFound(w, r)
		return
	}

	// The space detail is viewable without logging in (public-topic pages only).
	// [Ja] スペース詳細は未ログインでも閲覧できる (公開トピックのページのみ)。
	user := middleware.UserFromContext(ctx)
	var userID *model.UserID
	if user != nil {
		userID = &user.ID
	}

	output, err := h.getSpaceShowUC.Execute(ctx, usecase.GetSpaceShowInput{
		SpaceIdentifier: spaceIdentifier,
		UserID:          userID,
		Page:            currentPage,
		PageLimit:       spaceShowPageLimit,
	})
	if err != nil {
		slog.ErrorContext(ctx, "スペース詳細の取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if output == nil {
		handler.NotFound(w, r)
		return
	}
	pagination := viewmodel.NewPagination(int(currentPage), output.TotalCount, spaceShowPageLimit)
	if pagination.Current > pagination.Total {
		handler.NotFound(w, r)
		return
	}

	// Pages span multiple topics here, so each card shows its topic label (via TopicMap) and an
	// edit affordance gated on the per-topic page-edit permission resolved by the usecase.
	//
	// [Ja] スペース横断のためページは複数トピックに跨るので、各カードはトピックラベル (TopicMap 経由) と、
	// UseCase が解決したトピックごとのページ編集権限に応じた編集導線を表示する。
	pinnedPageVMs := make([]viewmodel.CardLinkPage, len(output.PinnedPages))
	for i, pg := range output.PinnedPages {
		card := viewmodel.NewCardLinkPage(pg, output.TopicMap)
		card.CanEdit = output.CanEditPageByTopic[pg.TopicID]
		pinnedPageVMs[i] = card
	}

	pageVMs := make([]viewmodel.CardLinkPage, len(output.Pages))
	for i, pg := range output.Pages {
		card := viewmodel.NewCardLinkPage(pg, output.TopicMap)
		card.CanEdit = output.CanEditPageByTopic[pg.TopicID]
		pageVMs[i] = card
	}

	spaceVM := viewmodel.NewSpace(output.Space)

	// Build links from the stored identifier, not from the URL, so that the canonical URL collapses
	// to one address per screen.
	//
	// [Ja] URL ではなく保存済みの識別子からリンクを組み立て、正規 URL を 1 画面 1 アドレスに
	// 集約する。
	spaceIdentVM := spaceVM.Identifier

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	titleKey := "space_show_title"
	if currentPage > 1 {
		titleKey = "space_show_paginated_title"
	}
	meta.SetTitleWithoutSuffix(ctx, titleKey, map[string]any{
		"SpaceName":  output.Space.Name,
		"PageNumber": currentPage,
	})
	meta.OGURL = h.cfg.AppURL() + string(templates.PaginatedPath(templates.SpacePath(spaceIdentVM), currentPage))
	meta.CurrentSpaceIdentifier = spaceIdentVM

	showData := spacepages.ShowData{
		Space:          spaceVM,
		PinnedPages:    pinnedPageVMs,
		Pages:          pageVMs,
		Pagination:     pagination,
		JoinedSpace:    output.JoinedSpace,
		SectionTopics:  viewmodel.NewCardLinkTopicsForSpace(output.SectionTopics, output.CanCreatePageByTopic, spaceIdentVM),
		CanCreateTopic: output.CanCreateTopic,
	}
	content := spacepages.Show(showData)

	signedIn := user != nil
	var userAtname string
	if user != nil {
		userAtname = user.Atname
	}

	// The space is the current page, so show its name as a plain (unlinked) trailing crumb carrying
	// aria-current. A signed-out viewer gets no home crumb, which leaves the trail with nothing to
	// navigate to; the header component drops the breadcrumb entirely in that case.
	//
	// [Ja] スペースは現在地のため、スペース名を aria-current を持つリンク無しの末尾パンくずとして
	// 表示する。未ログインの閲覧者にはホーム項目が付かず、経路にたどれる項目が無くなる。その場合は
	// ヘッダーコンポーネントがパンくずごと落とす。
	breadcrumbItems := append(components.HomeBreadcrumbItems(ctx, signedIn), components.BreadcrumbItem{
		Label:     spaceVM.Name,
		IsCurrent: true,
	})

	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		GlobalNav: components.GlobalNavData{
			CurrentPageName: templates.PageNameSpaceShow,
			SignedIn:        signedIn,
			UserAtname:      userAtname,
			SpaceIdentifier: spaceIdentVM,
		},

		BreadcrumbHeader: components.BreadcrumbHeaderData{
			MaxWidthClass: "max-w-3xl",

			// The space detail is public and indexable, so it opts into BreadcrumbList JSON-LD built
			// from the same items. A signed-out viewer is left with the current item alone, which is
			// nothing to navigate to, so no structured data is published for a crawler either.
			//
			// [Ja] スペース詳細は公開・インデックス対象のため、同じ項目列から作る BreadcrumbList
			// JSON-LD を有効にする。未ログインの閲覧者には現在項目しか残らず、たどれる項目が無いため、
			// クローラーに対して構造化データは出ない。
			StructuredDataBaseURL: h.cfg.AppURL(),

			Items: breadcrumbItems,
		},
	}

	err = layouts.Default(layoutData, content).Render(ctx, w)
	if err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
