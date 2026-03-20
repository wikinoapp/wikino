package suggestion_page_revision

import (
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
)

// Create は編集提案ページリビジョンを作成します (POST /s/{space_identifier}/suggestions/{suggestion_number}/page_revisions)
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
	suggestionNumberStr := chi.URLParam(r, "suggestion_number")

	suggestionNumberInt, err := strconv.ParseInt(suggestionNumberStr, 10, 32)
	if err != nil {
		handler.NotFound(w, r)
		return
	}
	suggestionNumber := model.SuggestionNumber(suggestionNumberInt)

	// フォームパラメータを取得
	pageNumberStr := r.FormValue("page_number")
	pageNumber, err := strconv.ParseInt(pageNumberStr, 10, 32)
	if err != nil {
		handler.NotFound(w, r)
		return
	}

	// 編集提案詳細を取得（認可チェック用）
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

	// オープンステータスでなければ更新不可
	if detailOutput.Suggestion.Status != model.SuggestionStatusOpen {
		suggestionPath := string(templates.SuggestionShowPath(string(spaceIdentifier), int32(suggestionNumber)))
		http.Redirect(w, r, suggestionPath, http.StatusSeeOther)
		return
	}

	// 編集提案ページを更新
	_, err = h.updateSuggestionPageUsecase.Execute(ctx, usecase.UpdateSuggestionPageInput{
		SpaceID:         detailOutput.Space.ID,
		SpaceMemberID:   detailOutput.SpaceMember.ID,
		SuggestionID:    detailOutput.Suggestion.ID,
		PageNumber:      int32(pageNumber),
		SuggestionPages: detailOutput.SuggestionPages,
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
