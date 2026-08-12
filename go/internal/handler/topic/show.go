package topic

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/httppagination"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	topicpages "github.com/wikinoapp/wikino/go/internal/templates/pages/topic"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

const topicShowPageLimit = 100

// Show はトピック詳細画面を表示します (GET /s/{space_identifier}/topics/{topic_number})
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// URLパラメータを取得
	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))
	topicNumberStr := chi.URLParam(r, "topic_number")

	topicNumber, err := strconv.ParseInt(topicNumberStr, 10, 32)
	if err != nil {
		handler.NotFound(w, r)
		return
	}

	// Parse the pagination parameter. A page whose SQL offset cannot fit the query's int32
	// parameter is rejected before invoking the usecase; the total-page check below handles every
	// smaller out-of-range value.
	//
	// [Ja] ページネーションパラメータを取得する。SQL offset がクエリの int32 パラメータに収まらない
	// ページは UseCase 呼び出し前に拒否する。これより小さい範囲外値は後段の総ページ数チェックで
	// 処理する。
	currentPage, ok := httppagination.ParsePageParam(r, topicShowPageLimit)
	if !ok {
		handler.NotFound(w, r)
		return
	}

	// ログインユーザーを取得
	user := middleware.UserFromContext(ctx)
	var userID *model.UserID
	if user != nil {
		userID = &user.ID
	}

	// UseCaseでデータを取得
	output, err := h.getTopicDetailUsecase.Execute(ctx, usecase.GetTopicDetailInput{
		SpaceIdentifier: spaceIdentifier,
		TopicNumber:     int32(topicNumber),
		UserID:          userID,
		Page:            currentPage,
		PageLimit:       topicShowPageLimit,
	})
	if err != nil {
		slog.ErrorContext(ctx, "トピック詳細の取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if output == nil {
		handler.NotFound(w, r)
		return
	}
	pagination := viewmodel.NewPagination(int(currentPage), output.TotalCount, topicShowPageLimit)
	if pagination.Current > pagination.Total {
		handler.NotFound(w, r)
		return
	}

	// 権限判定（UseCase が Authorizer 経由で判定した結果を使用）
	canUpdate := output.CanUpdateTopic
	canCreatePage := output.CanCreatePage

	// ViewModelに変換
	// トピック詳細画面ではトピック情報をカードに表示しないため、topicMapにnilを渡す
	pinnedPageVMs := make([]viewmodel.CardLinkPage, len(output.PinnedPages))
	for i, pg := range output.PinnedPages {
		card := viewmodel.NewCardLinkPage(pg, nil)
		card.CanEdit = canCreatePage
		pinnedPageVMs[i] = card
	}

	pageVMs := make([]viewmodel.CardLinkPage, len(output.Pages))
	for i, pg := range output.Pages {
		card := viewmodel.NewCardLinkPage(pg, nil)
		card.CanEdit = canCreatePage
		pageVMs[i] = card
	}

	topicVM := viewmodel.NewTopicForShow(output.Topic, canUpdate, canCreatePage)
	spaceVM := viewmodel.NewSpace(output.Space)

	// Build links from the stored identifier, not from the URL, so that the canonical URL collapses
	// to one address per screen.
	//
	// [Ja] URL ではなく保存済みの識別子からリンクを組み立て、正規 URL を 1 画面 1 アドレスに
	// 集約する。
	spaceIdentVM := spaceVM.Identifier

	// ページメタ情報を設定
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	titleKey := "topic_show_title"
	if currentPage > 1 {
		titleKey = "topic_show_paginated_title"
	}
	meta.SetTitleWithoutSuffix(ctx, titleKey, map[string]any{
		"TopicName":  output.Topic.Name,
		"SpaceName":  output.Space.Name,
		"PageNumber": currentPage,
	})
	meta.OGURL = h.cfg.AppURL() + string(templates.PaginatedPath(templates.TopicPath(spaceIdentVM, topicVM.Number), currentPage))
	meta.CurrentSpaceIdentifier = spaceIdentVM

	// テンプレートをレンダリング
	content := topicpages.Show(topicpages.ShowData{
		Topic:       topicVM,
		Space:       spaceVM,
		PinnedPages: pinnedPageVMs,
		Pages:       pageVMs,
		Pagination:  pagination,
	})

	signedIn := user != nil
	var userAtname string
	if user != nil {
		userAtname = user.Atname
	}

	breadcrumbItems := append(components.HomeBreadcrumbItems(ctx, signedIn),
		components.BreadcrumbItem{
			Label: spaceVM.Name,
			Path:  templates.SpacePath(spaceIdentVM),
		},
		// The topic is the current page, so it ends the breadcrumb as a plain (unlinked) crumb
		// carrying aria-current.
		//
		// [Ja] トピックは現在地のため、aria-current を持つリンク無しの項目としてパンくずを締める。
		components.BreadcrumbItem{
			Label:     topicVM.Name,
			IsCurrent: true,
		},
	)

	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		GlobalNav: components.GlobalNavData{
			CurrentPageName: templates.PageNameTopicShow,
			SignedIn:        signedIn,
			UserAtname:      userAtname,
			SpaceIdentifier: spaceIdentVM,
		},

		BreadcrumbHeader: components.BreadcrumbHeaderData{
			MaxWidthClass: "max-w-3xl",

			// The topic detail is public and indexable, so it opts into BreadcrumbList JSON-LD built
			// from the same items. Signed-out viewers start at the public space; signed-in viewers
			// also get /home.
			//
			// [Ja] トピック詳細は公開・インデックス対象のため、同じ項目列から作る BreadcrumbList
			// JSON-LD を有効にする。未ログインの閲覧者は公開スペースから始め、ログイン済みの閲覧者には
			// /home も含める。
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
