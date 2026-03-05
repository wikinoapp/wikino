package page

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	pagepages "github.com/wikinoapp/wikino/go/internal/templates/pages/page"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Edit はページ編集フォームを表示します (GET /s/{space_identifier}/pages/{page_number}/edit)
func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 認証済みユーザーを取得
	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	// URLパラメータを取得
	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))
	pageNumberStr := chi.URLParam(r, "page_number")

	pageNumber, err := strconv.ParseInt(pageNumberStr, 10, 32)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// スペースを取得
	space, err := h.spaceRepo.FindByIdentifier(ctx, spaceIdentifier)
	if err != nil {
		slog.ErrorContext(ctx, "スペースの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if space == nil {
		http.NotFound(w, r)
		return
	}

	// スペースメンバーを取得
	spaceMember, err := h.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, user.ID)
	if err != nil {
		slog.ErrorContext(ctx, "スペースメンバーの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if spaceMember == nil {
		http.NotFound(w, r)
		return
	}

	// ページを取得
	pg, err := h.pageRepo.FindBySpaceAndNumber(ctx, space.ID, model.PageNumber(pageNumber))
	if err != nil {
		slog.ErrorContext(ctx, "ページの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if pg == nil {
		http.NotFound(w, r)
		return
	}

	// トピックメンバーを取得してTopicPolicyを生成
	topicMember, err := h.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, pg.TopicID)
	if err != nil {
		slog.ErrorContext(ctx, "トピックメンバーの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	topicPolicy := policy.NewTopicPolicy(spaceMember, topicMember)
	if !topicPolicy.CanUpdatePage(pg) {
		http.NotFound(w, r)
		return
	}

	// トピックを取得（パンくずリスト用）
	topic, err := h.topicRepo.FindBySpaceAndID(ctx, space.ID, pg.TopicID)
	if err != nil {
		slog.ErrorContext(ctx, "トピックの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if topic == nil {
		slog.ErrorContext(ctx, "ページのトピックが見つかりません", "page_id", pg.ID, "topic_id", pg.TopicID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	topicVM := viewmodel.NewTopic(topic)

	// DraftPageを取得（存在すればその内容を表示）
	draftPage, err := h.draftPageRepo.FindByPageAndMember(ctx, pg.ID, spaceMember.ID, space.ID)
	if err != nil {
		slog.ErrorContext(ctx, "下書きの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 編集画面用のページViewModelを生成
	pageVM := viewmodel.NewPageForEdit(pg, draftPage)

	// リンク一覧を取得（DraftPage存在時はDraftPageのLinkedPageIDs、なければPageのLinkedPageIDs）
	var linkedPageIDs []model.PageID
	if draftPage != nil {
		linkedPageIDs = draftPage.LinkedPageIDs
	} else {
		linkedPageIDs = pg.LinkedPageIDs
	}

	// リンク先ページとバックリンクのデータを取得
	var paginatedLinks *repository.PaginatedPages
	var backlinkPaginatedMap map[model.PageID]*repository.PaginatedPages
	if len(linkedPageIDs) > 0 {
		paginatedLinks, err = h.pageRepo.FindLinkedPagesPaginated(ctx, linkedPageIDs, space.ID, 1, viewmodel.LinkLimit)
		if err != nil {
			slog.ErrorContext(ctx, "リンク先ページの取得に失敗", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		excludePageIDs := viewmodel.BuildExcludePageIDs(pg.ID, paginatedLinks.Pages)

		backlinkPaginatedMap, err = h.pageRepo.FindBacklinksForPages(ctx, paginatedLinks.Pages, space.ID, viewmodel.BacklinkLimit, excludePageIDs)
		if err != nil {
			slog.ErrorContext(ctx, "バックリンクの取得に失敗", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	// ページレベルのバックリンク一覧をページネーション付きで取得（公開済みページのLinkedPageIDsに基づく）
	paginatedBacklinks, err := h.pageRepo.FindBacklinkedPagesPaginated(ctx, pg.ID, space.ID, 1, viewmodel.PageBacklinkLimit, nil)
	if err != nil {
		slog.ErrorContext(ctx, "ページレベルのバックリンクの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// トピックIDを収集してトピックを一括取得
	var allPageSlices [][]*model.Page
	if paginatedLinks != nil {
		allPageSlices = append(allPageSlices, paginatedLinks.Pages)
	}
	for _, paginated := range backlinkPaginatedMap {
		allPageSlices = append(allPageSlices, paginated.Pages)
	}
	allPageSlices = append(allPageSlices, paginatedBacklinks.Pages)

	topicIDs := viewmodel.CollectTopicIDsFromPages(allPageSlices...)
	topics, err := h.topicRepo.FindByIDsAndSpace(ctx, topicIDs, space.ID)
	if err != nil {
		slog.ErrorContext(ctx, "トピックの一括取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// ViewModelを構築
	var linkedPages []*model.Page
	var linkedTotalCount int64
	if paginatedLinks != nil {
		linkedPages = paginatedLinks.Pages
		linkedTotalCount = paginatedLinks.TotalCount
	}

	backlinksPerPage := make(map[model.PageID]*viewmodel.PageSliceWithCount, len(backlinkPaginatedMap))
	for pageID, paginated := range backlinkPaginatedMap {
		backlinksPerPage[pageID] = &viewmodel.PageSliceWithCount{
			Pages:      paginated.Pages,
			TotalCount: paginated.TotalCount,
		}
	}

	editLinkData := viewmodel.BuildEditLinkData(viewmodel.BuildEditLinkDataInput{
		LinkedPages:       linkedPages,
		LinkedTotalCount:  linkedTotalCount,
		BacklinksPerPage:  backlinksPerPage,
		PageBacklinks:     paginatedBacklinks.Pages,
		PageBacklinkCount: paginatedBacklinks.TotalCount,
		Topics:            topics,
		SpaceIdentifier:   spaceIdentifier,
		PageNumber:        int32(pg.Number),
		CurrentPage:       1,
	})
	linkListVM := editLinkData.LinkList
	backlinkListVM := editLinkData.BacklinkList

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// ページメタ情報を設定
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "page_edit_title")

	// フラッシュメッセージを取得
	flash := h.flashMgr.GetFlash(w, r)

	// テンプレートをレンダリング
	spaceVM := viewmodel.NewSpace(space)

	content := pagepages.Edit(pagepages.EditPageData{
		CSRFToken:     csrfToken,
		Page:          pageVM,
		Space:         spaceVM,
		Topic:         topicVM,
		LinkList:      linkListVM,
		BacklinkList:  backlinkListVM,
		ManualSaveURL: string(templates.PageDraftPagePath(space.Identifier.String(), int32(pg.Number))),
	})

	// サイドバーコンテンツを取得
	joinedTopics, draftPages := h.sidebarHelper.Content(ctx, user.ID)

	layoutData := layouts.DefaultLayoutData{
		Meta:       meta,
		Flash:      flash,
		HideFooter: true,
		Sidebar: components.SidebarData{
			DefaultClosed:   layouts.SidebarDefaultClosed(r),
			CurrentPageName: templates.PageNamePageEdit,
			SignedIn:        true,
			UserAtname:      user.Atname,
			SpaceIdentifier: string(spaceIdentifier),
			JoinedTopics:    joinedTopics,
			DraftPages:      draftPages,
		},
	}

	err = layouts.Default(layoutData, content).Render(ctx, w)
	if err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
