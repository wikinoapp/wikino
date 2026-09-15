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

// spaceShowPageLimitは通常ページ (ピン留めなし) の1ページあたりの表示件数です。
const spaceShowPageLimit = 100

// Showはスペース詳細画面を表示します (GET /s/{space_identifier})。
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))

	// ページネーションパラメータを取得する。SQL offsetがクエリのint32パラメータに収まらない
	// ページはUseCase呼び出し前に拒否する。これより小さい範囲外値は後段の総ページ数チェックで
	// 処理する。
	currentPage, ok := httppagination.ParsePageParam(r, spaceShowPageLimit)
	if !ok {
		handler.NotFound(w, r)
		return
	}

	// スペース詳細は未ログインでも閲覧できる (公開トピックのページのみ)。
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

	// スペース横断のためページは複数トピックに跨るので、各カードはトピックラベル (TopicMap経由) と、
	// UseCaseが解決したトピックごとのページ編集権限に応じた編集導線を表示する。
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

	// URLではなく保存済みの識別子からリンクを組み立て、正規URLを1画面1アドレスに
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

	// スペースは現在地のため、スペース名をaria-currentを持つリンク無しの末尾パンくずとして
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

			// スペース詳細は公開・インデックス対象のため、同じ項目列から作るBreadcrumbList
			// JSON-LDを有効にする。未ログインの閲覧者には現在項目しか残らず、たどれる項目が無いため、
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
