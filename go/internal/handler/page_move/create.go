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
	"github.com/wikinoapp/wikino/go/internal/validator"
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

	// UseCaseでデータを取得
	output, err := h.getPageMoveDataUC.Execute(ctx, usecase.GetPageMoveDataInput{
		SpaceIdentifier: spaceIdentifier,
		PageNumber:      int32(pageNumber),
		UserID:          user.ID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "ページ移動データの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if output == nil {
		http.NotFound(w, r)
		return
	}

	// 認可チェック
	topicPolicy := policy.NewTopicPolicy(output.SpaceMember, output.TopicMember)
	if !topicPolicy.CanUpdatePage(output.Page) {
		http.NotFound(w, r)
		return
	}

	// フォームデータを取得
	destTopicNumber := r.FormValue("dest_topic")

	// ページタイトルを取得
	var pageTitle string
	if output.Page.Title != nil {
		pageTitle = *output.Page.Title
	}

	// バリデーション
	validationResult := h.createValidator.Validate(ctx, validator.PageMoveCreateValidatorInput{
		DestTopicNumber: destTopicNumber,
		PageID:          output.Page.ID,
		PageTitle:       pageTitle,
		CurrentTopicID:  output.Page.TopicID,
		SpaceID:         output.Space.ID,
		SpaceMember:     output.SpaceMember,
	})

	if validationResult.Err != nil {
		slog.ErrorContext(ctx, "バリデーション中にシステムエラーが発生", "error", validationResult.Err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if validationResult.FormErrors.HasErrors() {
		w.WriteHeader(http.StatusUnprocessableEntity)
		h.renderMoveForm(w, r, user, spaceIdentifier, output, validationResult.FormErrors)
		return
	}

	// ページ移動ユースケースを実行
	_, err = h.movePageUC.Execute(ctx, usecase.MovePageInput{
		PageID:      output.Page.ID,
		SpaceID:     output.Space.ID,
		DestTopicID: validationResult.DestTopic.ID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "ページの移動に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// フラッシュメッセージを設定してリダイレクト
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "page_move_success"))
	pagePath := fmt.Sprintf("/s/%s/pages/%d", string(spaceIdentifier), output.Page.Number)
	http.Redirect(w, r, pagePath, http.StatusSeeOther)
}
