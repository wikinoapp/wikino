package topic

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	topicpages "github.com/wikinoapp/wikino/go/internal/templates/pages/topic"
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

	// ログインユーザーのスペースメンバーを取得（未ログインならnil）
	user := middleware.UserFromContext(ctx)
	var spaceMember *model.SpaceMember
	if user != nil {
		spaceMember, err = h.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, user.ID)
		if err != nil {
			slog.ErrorContext(ctx, "スペースメンバーの取得に失敗", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	// トピックを取得
	topic, err := h.topicRepo.FindBySpaceAndNumber(ctx, space.ID, int32(topicNumber))
	if err != nil {
		slog.ErrorContext(ctx, "トピックの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if topic == nil {
		http.NotFound(w, r)
		return
	}

	// 権限チェック: 非公開トピックはトピックメンバーのみ閲覧可能
	var topicMember *model.TopicMember
	if spaceMember != nil {
		topicMember, err = h.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, topic.ID)
		if err != nil {
			slog.ErrorContext(ctx, "トピックメンバーの取得に失敗", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	if topic.Visibility == model.TopicVisibilityPrivate {
		// スペースオーナーまたはトピックメンバーのみ閲覧可能
		if spaceMember == nil || (spaceMember.Role != model.SpaceMemberRoleOwner && topicMember == nil) {
			http.NotFound(w, r)
			return
		}
	}

	// ページネーションパラメータを取得
	var currentPage int32 = 1
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.ParseInt(pageStr, 10, 32); err == nil && p > 0 {
			currentPage = int32(p)
		}
	}

	// ピン留めページを取得
	pinnedPages, err := h.pageRepo.FindPinnedByTopic(ctx, topic.ID, space.ID)
	if err != nil {
		slog.ErrorContext(ctx, "ピン留めページの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 通常ページをページネーションで取得
	paginatedResult, err := h.pageRepo.FindRegularByTopicPaginated(ctx, topic.ID, space.ID, currentPage, topicShowPageLimit)
	if err != nil {
		slog.ErrorContext(ctx, "通常ページの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 権限判定
	canUpdate := canUpdateTopic(spaceMember, topicMember)
	canCreatePage := canCreateTopicPage(spaceMember, topicMember)

	// ViewModelに変換
	// トピック詳細画面ではトピック情報をカードに表示しないため、topicMapにnilを渡す
	pinnedPageVMs := make([]viewmodel.CardLinkPage, len(pinnedPages))
	for i, pg := range pinnedPages {
		card := viewmodel.NewCardLinkPage(pg, nil)
		card.CanEdit = canCreatePage
		pinnedPageVMs[i] = card
	}

	pageVMs := make([]viewmodel.CardLinkPage, len(paginatedResult.Pages))
	for i, pg := range paginatedResult.Pages {
		card := viewmodel.NewCardLinkPage(pg, nil)
		card.CanEdit = canCreatePage
		pageVMs[i] = card
	}

	topicVM := viewmodel.NewTopicForShow(topic, canUpdate, canCreatePage)
	spaceVM := viewmodel.NewSpace(space)
	pagination := viewmodel.NewPagination(int(currentPage), paginatedResult.TotalCount, topicShowPageLimit)

	// ページメタ情報を設定
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.Title = i18n.T(ctx, "topic_show_title", map[string]any{
		"TopicName": topic.Name,
		"SpaceName": space.Name,
	}) + " | Wikino"

	// フラッシュメッセージを取得
	flash := h.flashMgr.GetFlash(w, r)

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

	layoutData := layouts.DefaultLayoutData{
		Meta:  meta,
		Flash: flash,
		Sidebar: components.SidebarData{
			CurrentPageName: templates.PageNameTopicShow,
			SignedIn:        signedIn,
			UserAtname:      userAtname,
			SpaceIdentifier: string(spaceIdentifier),
		},
		BottomNav: components.BottomNavData{
			CurrentPageName: templates.PageNameTopicShow,
			SignedIn:        signedIn,
			SpaceIdentifier: string(spaceIdentifier),
		},
	}

	// ログイン済みの場合はサイドバーコンテンツを取得
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

// canUpdateTopic はトピックの設定を更新できるかを判定する
// スペースオーナーまたはトピック管理者のみ可能
func canUpdateTopic(spaceMember *model.SpaceMember, topicMember *model.TopicMember) bool {
	if spaceMember == nil {
		return false
	}
	if spaceMember.Role == model.SpaceMemberRoleOwner {
		return true
	}
	if topicMember != nil && topicMember.Role == model.TopicMemberRoleAdmin {
		return true
	}
	return false
}

// canCreateTopicPage はトピックにページを作成できるかを判定する
// トピックメンバー（スペースオーナーを含む）のみ可能
func canCreateTopicPage(spaceMember *model.SpaceMember, topicMember *model.TopicMember) bool {
	if spaceMember == nil {
		return false
	}
	if spaceMember.Role == model.SpaceMemberRoleOwner {
		return true
	}
	return topicMember != nil
}
