package suggestion

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// Update は編集提案を更新します (PATCH /s/{space_identifier}/suggestions/{suggestion_number})
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 認証済みユーザーを取得
	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	// URLパラメータを取得
	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))
	suggestionNumberStr := chi.URLParam(r, "suggestion_number")

	suggestionNumber, err := strconv.ParseInt(suggestionNumberStr, 10, 32)
	if err != nil {
		handler.NotFound(w, r)
		return
	}

	// フォームデータをパース
	if err := r.ParseForm(); err != nil {
		slog.ErrorContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	title := r.FormValue("title")
	body := r.FormValue("body")

	// UseCaseでデータを取得
	output, err := h.getSuggestionDetailUsecase.Execute(ctx, usecase.GetSuggestionDetailInput{
		SpaceIdentifier:  spaceIdentifier,
		SuggestionNumber: model.SuggestionNumber(suggestionNumber),
		UserID:           &user.ID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "編集提案詳細の取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if output == nil {
		handler.NotFound(w, r)
		return
	}

	// 権限チェック
	if output.SpaceMember == nil {
		handler.NotFound(w, r)
		return
	}
	topicPolicy := policy.NewTopicPolicy(output.SpaceMember, output.TopicMember)
	if !topicPolicy.CanUpdateSuggestion(output.Suggestion) {
		handler.NotFound(w, r)
		return
	}

	// バリデーション
	validationResult := h.updateValidator.Validate(ctx, validator.SuggestionUpdateValidatorInput{
		Title: title,
		Body:  body,
	})

	if validationResult.FormErrors.HasErrors() {
		w.WriteHeader(http.StatusUnprocessableEntity)
		h.renderEditForm(w, r, user, spaceIdentifier, output, validationResult.FormErrors, title, body)
		return
	}

	// 編集提案を更新
	_, err = h.updateSuggestionUsecase.Execute(ctx, usecase.UpdateSuggestionInput{
		SuggestionID:     output.Suggestion.ID,
		SpaceID:          output.Space.ID,
		SpaceIdentifier:  spaceIdentifier,
		CurrentTopicName: output.Topic.Name,
		Title:            title,
		Body:             body,
	})
	if err != nil {
		slog.ErrorContext(ctx, "編集提案の更新に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// フラッシュメッセージを設定してリダイレクト
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "suggestion_update_success"))
	suggestionPath := fmt.Sprintf("/s/%s/suggestions/%d",
		string(spaceIdentifier),
		suggestionNumber,
	)
	http.Redirect(w, r, suggestionPath, http.StatusSeeOther)
}
