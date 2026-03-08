package page_move

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Create はページ移動を実行します (POST /s/{space_identifier}/pages/{page_number}/move)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
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

	// フォームデータを取得
	destTopicNumber := r.FormValue("dest_topic")

	// ページタイトルを取得
	var pageTitle string
	if pg.Title != nil {
		pageTitle = *pg.Title
	}

	// バリデーション
	validator := NewCreateValidator(h.pageRepo, h.topicRepo, h.topicMemberRepo)
	validationResult := validator.Validate(ctx, CreateValidatorInput{
		DestTopicNumber: destTopicNumber,
		PageID:          pg.ID,
		PageTitle:       pageTitle,
		CurrentTopicID:  pg.TopicID,
		SpaceID:         space.ID,
		SpaceMember:     spaceMember,
	})

	if validationResult.Err != nil {
		slog.ErrorContext(ctx, "バリデーション中にシステムエラーが発生", "error", validationResult.Err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if validationResult.FormErrors.HasErrors() {
		// 現在のトピックを取得（フォーム再表示用）
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

		w.WriteHeader(http.StatusUnprocessableEntity)
		h.renderMoveForm(w, r, user, spaceIdentifier, space, pg, currentTopic, availableTopics, validationResult.FormErrors)
		return
	}

	// ページ移動ユースケースを実行
	_, err = h.movePageUC.Execute(ctx, usecase.MovePageInput{
		PageID:      pg.ID,
		SpaceID:     space.ID,
		DestTopicID: validationResult.DestTopic.ID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "ページの移動に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// フラッシュメッセージを設定してリダイレクト
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "page_move_success"))
	pagePath := fmt.Sprintf("/s/%s/pages/%d", string(spaceIdentifier), pg.Number)
	http.Redirect(w, r, pagePath, http.StatusSeeOther)
}
