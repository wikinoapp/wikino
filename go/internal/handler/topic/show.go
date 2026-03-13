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
		http.NotFound(w, r)
		return
	}

	// ページネーションパラメータを取得
	var currentPage int32 = 1
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.ParseInt(pageStr, 10, 32); err == nil && p > 0 {
			currentPage = int32(p)
		}
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
		http.NotFound(w, r)
		return
	}

	// 権限判定
	canUpdate := canUpdateTopic(output.SpaceMember, output.TopicMember)
	canCreatePage := canCreateTopicPage(output.SpaceMember, output.TopicMember)

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
	pagination := viewmodel.NewPagination(int(currentPage), output.TotalCount, topicShowPageLimit)

	// ページメタ情報を設定
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.Title = i18n.T(ctx, "topic_show_title", map[string]any{
		"TopicName": output.Topic.Name,
		"SpaceName": output.Space.Name,
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
