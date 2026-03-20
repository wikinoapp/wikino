package suggestion_page

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// Update は編集提案ページを更新します (PATCH /s/{space_identifier}/suggestions/{suggestion_number}/suggestion_pages/{suggestion_page_id})
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
	suggestionPageID := model.SuggestionPageID(chi.URLParam(r, "suggestion_page_id"))

	suggestionNumberInt, err := strconv.ParseInt(suggestionNumberStr, 10, 32)
	if err != nil {
		handler.NotFound(w, r)
		return
	}
	suggestionNumber := model.SuggestionNumber(suggestionNumberInt)

	if suggestionPageID == "" {
		handler.NotFound(w, r)
		return
	}

	// 1. 読み取りUseCase: 編集提案詳細を取得
	detailOutput, err := h.getSuggestionDetailUsecase.Execute(ctx, usecase.GetSuggestionDetailInput{
		SpaceIdentifier:  spaceIdentifier,
		SuggestionNumber: suggestionNumber,
		UserID:           &user.ID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "編集提案詳細の取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if detailOutput == nil {
		handler.NotFound(w, r)
		return
	}

	// スペースメンバーでなければ403
	if detailOutput.SpaceMember == nil {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// 下書きまたはオープンステータスでなければ更新不可
	if detailOutput.Suggestion.Status != model.SuggestionStatusDraft &&
		detailOutput.Suggestion.Status != model.SuggestionStatusOpen {
		suggestionPath := string(templates.SuggestionShowPath(string(spaceIdentifier), int32(suggestionNumber)))
		http.Redirect(w, r, suggestionPath, http.StatusSeeOther)
		return
	}

	// SuggestionPageが現在の編集提案に属していることを検証
	var targetSP *model.SuggestionPage
	for _, sp := range detailOutput.SuggestionPages {
		if sp.ID == suggestionPageID {
			targetSP = sp
			break
		}
	}
	if targetSP == nil {
		handler.NotFound(w, r)
		return
	}

	// 2. Validator: DraftPageの取得・検証
	validationResult := h.updateValidator.Validate(ctx, validator.SuggestionPageUpdateValidatorInput{
		SuggestionPageID: suggestionPageID,
		PageID:           targetSP.PageID,
		SpaceMemberID:    detailOutput.SpaceMember.ID,
		SpaceID:          detailOutput.Space.ID,
	})
	if validationResult.Err != nil {
		if errors.Is(validationResult.Err, validator.ErrDraftPageNotFound) ||
			errors.Is(validationResult.Err, validator.ErrDraftPageNotLinked) {
			slog.WarnContext(ctx, "編集提案ページ更新の前提条件を満たしていない", "error", validationResult.Err)
			handler.NotFound(w, r)
			return
		}
		slog.ErrorContext(ctx, "編集提案ページ更新のバリデーションに失敗", "error", validationResult.Err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 3. 書き込みUseCase: 編集提案ページを更新
	_, err = h.updateSuggestionPageUsecase.Execute(ctx, usecase.UpdateSuggestionPageInput{
		SpaceID:          detailOutput.Space.ID,
		SpaceMemberID:    detailOutput.SpaceMember.ID,
		SuggestionPageID: suggestionPageID,
		DraftPage:        validationResult.DraftPage,
	})
	if err != nil {
		slog.ErrorContext(ctx, "編集提案ページの更新に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// フラッシュメッセージを設定してリダイレクト
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_suggestion_page_updated"))
	changesPath := string(templates.SuggestionChangesPath(string(spaceIdentifier), int32(suggestionNumber)))
	http.Redirect(w, r, changesPath, http.StatusSeeOther)
}
