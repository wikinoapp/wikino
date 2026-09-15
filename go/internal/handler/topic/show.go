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

// Showはトピック詳細画面を表示します (GET /s/{space_identifier}/topics/{topic_number})
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

	// ページネーションパラメータを取得する。SQL offsetがクエリのint32パラメータに収まらない
	// ページはUseCase呼び出し前に拒否する。これより小さい範囲外値は後段の総ページ数チェックで
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

	// 権限判定 (UseCaseがAuthorizer経由で判定した結果を使用)
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

	// URLではなく保存済みの識別子からリンクを組み立て、正規URLを1画面1アドレスに
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
		// トピックは現在地のため、aria-currentを持つリンク無しの項目としてパンくずを締める。
		// アイコンは見出しが言葉で示す公開範囲を装飾として繰り返す。パンくずは閲覧者の現在地を
		// 読むためのもので、状態を名指すのは見出しのほうである。
		components.BreadcrumbItem{
			Label:     topicVM.Name,
			IconName:  topicVM.IconName,
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

			// トピック詳細は公開・インデックス対象のため、同じ項目列から作るBreadcrumbList
			// JSON-LDを有効にする。未ログインの閲覧者は公開スペースから始め、ログイン済みの閲覧者には
			// /homeも含める。
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
