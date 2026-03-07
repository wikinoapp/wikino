package page_move

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	pagemovepages "github.com/wikinoapp/wikino/go/internal/templates/pages/page_move"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// New はページ移動フォームを表示します (GET /s/{space_identifier}/pages/{page_number}/move)
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
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

	// 移動元トピックの権限チェック（CanUpdatePage）
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

	// 現在のトピックを取得
	currentTopic, err := h.topicRepo.FindBySpaceAndID(ctx, space.ID, pg.TopicID)
	if err != nil {
		slog.ErrorContext(ctx, "トピックの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if currentTopic == nil {
		slog.ErrorContext(ctx, "ページのトピックが見つかりません", "page_id", pg.ID, "topic_id", pg.TopicID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 移動先候補のトピック一覧を取得
	availableTopics, err := h.availableTopicsForMove(ctx, spaceMember, space, pg.TopicID)
	if err != nil {
		slog.ErrorContext(ctx, "移動先トピック一覧の取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.renderMoveForm(w, r, user, spaceIdentifier, space, pg, currentTopic, availableTopics, nil)
}

// renderMoveForm はページ移動フォームをレンダリングします
func (h *Handler) renderMoveForm(
	w http.ResponseWriter,
	r *http.Request,
	user *model.User,
	spaceIdentifier model.SpaceIdentifier,
	space *model.Space,
	pg *model.Page,
	currentTopic *model.Topic,
	availableTopics []*model.Topic,
	formErrors *session.FormErrors,
) {
	ctx := r.Context()

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// ページメタ情報を設定
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "page_move_title")

	// フラッシュメッセージを取得
	flash := h.flashMgr.GetFlash(w, r)

	pageVM := viewmodel.NewPageForMove(pg)
	spaceVM := viewmodel.NewSpace(space)
	currentTopicVM := viewmodel.NewTopic(currentTopic)

	// 移動先トピック一覧のViewModel
	topicsForSelect := make([]viewmodel.TopicForSelect, len(availableTopics))
	for i, t := range availableTopics {
		topicsForSelect[i] = viewmodel.NewTopicForSelect(t)
	}

	content := pagemovepages.New(pagemovepages.MovePageData{
		CSRFToken:       csrfToken,
		FormErrors:      formErrors,
		Page:            pageVM,
		Space:           spaceVM,
		CurrentTopic:    currentTopicVM,
		AvailableTopics: topicsForSelect,
	})

	// サイドバーコンテンツを取得
	sidebarContent := h.sidebarHelper.Content(ctx, user.ID)

	layoutData := layouts.DefaultLayoutData{
		Meta:  meta,
		Flash: flash,
		Sidebar: components.SidebarData{
			CurrentPageName:   templates.PageNamePageMove,
			SignedIn:          true,
			UserAtname:        user.Atname,
			SpaceIdentifier:   string(spaceIdentifier),
			JoinedTopics:      sidebarContent.JoinedTopics,
			DraftPages:        sidebarContent.DraftPages,
			HasMoreDraftPages: sidebarContent.HasMoreDraftPages,
		},
	}

	err := layouts.Default(layoutData, content).Render(ctx, w)
	if err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// availableTopicsForMove は移動先候補のトピック一覧を取得します。
// スペースオーナーは全アクティブトピック、それ以外は所属トピックのみ返します。
// 現在のトピックは除外します。
// スペースオーナーは同スペース内の全トピックにCanCreatePageが真であり、
// 非オーナーはListJoinedBySpaceMemberが所属トピックのみを返すため、
// いずれの場合もリスト取得の段階で権限が暗黙的に満たされています。
func (h *Handler) availableTopicsForMove(
	ctx context.Context,
	spaceMember *model.SpaceMember,
	space *model.Space,
	currentTopicID model.TopicID,
) ([]*model.Topic, error) {
	var topics []*model.Topic
	var err error

	if spaceMember.Role == model.SpaceMemberRoleOwner {
		topics, err = h.topicRepo.ListActiveBySpace(ctx, space.ID)
	} else {
		topics, err = h.topicRepo.ListJoinedBySpaceMember(ctx, spaceMember.ID, space.ID)
	}
	if err != nil {
		return nil, err
	}

	// 現在のトピックを除外
	var filtered []*model.Topic
	for _, t := range topics {
		if t.ID == currentTopicID {
			continue
		}
		filtered = append(filtered, t)
	}

	return filtered, nil
}
