package space

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

	// Parse the pagination parameter (default 1; invalid values fall back to 1).
	// [Ja] ページネーションパラメータを取得 (デフォルト 1、不正値は 1 に丸める)。
	var currentPage int32 = 1
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.ParseInt(pageStr, 10, 32); err == nil && p > 0 {
			currentPage = int32(p)
		}
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
	spaceIdentVM := viewmodel.NewSpaceIdentifier(spaceIdentifier)
	pagination := viewmodel.NewPagination(int(currentPage), output.TotalCount, spaceShowPageLimit)

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitleWithoutSuffix(ctx, "space_show_title", map[string]any{
		"SpaceName": output.Space.Name,
	})
	meta.CurrentSpaceIdentifier = spaceIdentVM

	showData := spacepages.ShowData{
		Space:          spaceVM,
		PinnedPages:    pinnedPageVMs,
		Pages:          pageVMs,
		Pagination:     pagination,
		JoinedSpace:    output.JoinedSpace,
		SectionTopics:  viewmodel.NewTopicsForSpaceSection(output.SectionTopics, output.CanCreatePageByTopic),
		HasFirstTopic:  output.FirstJoinedTopic != nil,
		CanCreateTopic: output.CanCreateTopic,
	}
	content := spacepages.Show(showData)

	signedIn := user != nil
	var userAtname string
	if user != nil {
		userAtname = user.Atname
	}

	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		Sidebar: components.SidebarData{
			CurrentPageName: templates.PageNameSpaceShow,
			SignedIn:        signedIn,
			UserAtname:      userAtname,
			SpaceIdentifier: spaceIdentVM,
		},
		BottomNav: components.BottomNavData{
			CurrentPageName: templates.PageNameSpaceShow,
			SignedIn:        signedIn,
			SpaceIdentifier: spaceIdentVM,
		},
	}

	// Load sidebar content only for logged-in users.
	// [Ja] ログイン済みの場合のみサイドバーコンテンツを取得する。
	if user != nil {
		sidebarContent := h.sidebarHelper.Content(ctx, user.ID)
		layoutData.Sidebar.JoinedTopics = sidebarContent.JoinedTopics
		layoutData.Sidebar.DraftPages = sidebarContent.DraftPages
		layoutData.Sidebar.HasMoreDraftPages = sidebarContent.HasMoreDraftPages
	}

	err = layouts.Default(layoutData, content).Render(ctx, w)
	if err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
