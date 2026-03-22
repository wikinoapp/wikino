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
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// Create は編集提案を作成します (POST /s/{space_identifier}/topics/{topic_number}/suggestions)
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
	topicNumberStr := chi.URLParam(r, "topic_number")

	topicNumber, err := strconv.ParseInt(topicNumberStr, 10, 32)
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
	draftPageIDStrs := r.Form["draft_page_ids"]

	// 下書きページIDをドメイン型に変換
	draftPageIDs := make([]model.DraftPageID, len(draftPageIDStrs))
	for i, id := range draftPageIDStrs {
		draftPageIDs[i] = model.DraftPageID(id)
	}

	// UseCaseでデータを取得（フォーム再表示用にも必要）
	output, err := h.getSuggestionNewUsecase.Execute(ctx, usecase.GetSuggestionNewInput{
		SpaceIdentifier: spaceIdentifier,
		TopicNumber:     int32(topicNumber),
		UserID:          user.ID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "編集提案作成データの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if output == nil {
		handler.NotFound(w, r)
		return
	}

	// バリデーション
	validationResult := h.createValidator.Validate(ctx, validator.SuggestionCreateValidatorInput{
		Title:         title,
		DraftPageIDs:  draftPageIDs,
		SpaceMemberID: output.SpaceMember.ID,
		TopicID:       output.Topic.ID,
		SpaceID:       output.Space.ID,
	})

	if validationResult.Err != nil {
		slog.ErrorContext(ctx, "バリデーション中にシステムエラーが発生", "error", validationResult.Err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if validationResult.FormErrors.HasErrors() {
		w.WriteHeader(http.StatusUnprocessableEntity)
		h.renderNewForm(w, r, user, spaceIdentifier, output, validationResult.FormErrors, title, body, draftPageIDStrs)
		return
	}

	// 編集提案を作成
	createOutput, err := h.createSuggestionUsecase.Execute(ctx, usecase.CreateSuggestionInput{
		SpaceID:          output.Space.ID,
		SpaceIdentifier:  spaceIdentifier,
		TopicID:          output.Topic.ID,
		SpaceMemberID:    output.SpaceMember.ID,
		Title:            title,
		Body:             body,
		CurrentTopicName: output.Topic.Name,
		DraftPages:       validationResult.DraftPages,
	})
	if err != nil {
		slog.ErrorContext(ctx, "編集提案の作成に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// フラッシュメッセージを設定してリダイレクト
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "suggestion_create_success"))
	suggestionPath := fmt.Sprintf("/s/%s/topics/%d/suggestions/%d",
		string(spaceIdentifier),
		topicNumber,
		createOutput.Suggestion.Number,
	)
	http.Redirect(w, r, suggestionPath, http.StatusSeeOther)
}
