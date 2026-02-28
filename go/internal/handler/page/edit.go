package page

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	pagepages "github.com/wikinoapp/wikino/go/internal/templates/pages/page"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Edit はページ編集フォームを表示します (GET /go/s/{space_identifier}/pages/{page_number}/edit)
func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 認証済みユーザーを取得
	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	// URLパラメータを取得
	spaceIdentifier := chi.URLParam(r, "space_identifier")
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

	const linkLimit int32 = 15
	const backlinkLimit int32 = 14

	var linkListVM viewmodel.LinkList
	if len(linkedPageIDs) > 0 {
		paginatedLinks, err := h.pageRepo.FindLinkedPagesPaginated(ctx, linkedPageIDs, space.ID, 1, linkLimit)
		if err != nil {
			slog.ErrorContext(ctx, "リンク先ページの取得に失敗", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		backlinkMap := make(map[model.PageID]viewmodel.BacklinkList, len(paginatedLinks.Pages))
		for _, linkedPage := range paginatedLinks.Pages {
			paginatedBacklinks, err := h.pageRepo.FindBacklinkedPagesPaginated(ctx, linkedPage.ID, space.ID, 1, backlinkLimit)
			if err != nil {
				slog.ErrorContext(ctx, "バックリンクの取得に失敗", "error", err, "page_id", linkedPage.ID)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			backlinkMap[linkedPage.ID] = viewmodel.NewBacklinkList(viewmodel.NewBacklinkListInput{
				Pages:      paginatedBacklinks.Pages,
				Pagination: viewmodel.NewPagination(1, paginatedBacklinks.TotalCount, int(backlinkLimit)),
			})
		}

		linkListVM = viewmodel.NewLinkList(viewmodel.NewLinkListInput{
			Pages:           paginatedLinks.Pages,
			BacklinkMap:     backlinkMap,
			Pagination:      viewmodel.NewPagination(1, paginatedLinks.TotalCount, int(linkLimit)),
			SpaceIdentifier: spaceIdentifier,
			PageNumber:      int32(pg.Number),
		})
	}

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
		CSRFToken: csrfToken,
		Page:      pageVM,
		Space:     spaceVM,
		Topic:     topicVM,
		LinkList:  linkListVM,
	})

	layoutData := layouts.DefaultLayoutData{
		Meta:                 meta,
		Flash:                flash,
		HideFooter:           true,
		DefaultSidebarClosed: true,
	}

	err = layouts.Default(layoutData, content).Render(ctx, w)
	if err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
