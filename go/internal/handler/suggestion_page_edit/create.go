package suggestion_page_edit

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Create は編集提案ページの編集を開始します (POST /s/{space_identifier}/suggestions/{suggestion_number}/page_edits)
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
	suggestionPageID := model.SuggestionPageID(r.FormValue("suggestion_page_id"))
	if suggestionPageID == "" {
		handler.NotFound(w, r)
		return
	}
	force := r.FormValue("force") == "true"

	// UseCaseを実行
	output, err := h.startSuggestionPageEditUsecase.Execute(ctx, usecase.StartSuggestionPageEditInput{
		SpaceIdentifier:  spaceIdentifier,
		SuggestionNumber: suggestionNumber,
		SuggestionPageID: suggestionPageID,
		UserID:           user.ID,
		Force:            force,
	})
	if err != nil {
		h.handleCreateError(w, r, err, spaceIdentifier, suggestionNumber)
		return
	}

	switch output.Status {
	case usecase.StartSuggestionPageEditRedirect:
		// ページ編集画面にリダイレクト
		editPath := string(templates.PageEditPath(string(spaceIdentifier), int32(output.PageNumber)))
		http.Redirect(w, r, editPath, http.StatusSeeOther)

	case usecase.StartSuggestionPageEditConflict:
		// 確認画面にリダイレクト
		confirmPath := string(templates.SuggestionPageEditShowPath(string(spaceIdentifier), int32(suggestionNumber), string(suggestionPageID)))
		if output.ConflictDraftKind == usecase.ConflictDraftKindOtherSuggestion {
			confirmPath += "?draft_kind=other_suggestion"
		}
		http.Redirect(w, r, confirmPath, http.StatusSeeOther)

	default:
		slog.ErrorContext(ctx, "予期しないステータス", "status", output.Status)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *Handler) handleCreateError(w http.ResponseWriter, r *http.Request, err error, spaceIdentifier model.SpaceIdentifier, suggestionNumber model.SuggestionNumber) {
	ctx := r.Context()
	changesPath := string(templates.SuggestionChangesPath(string(spaceIdentifier), int32(suggestionNumber)))

	if ae := model.AsAppError(err); ae != nil {
		switch ae.Code {
		case model.AppErrCodeResourceNotFound:
			handler.NotFound(w, r)
		case model.AppErrCodeForbidden:
			http.Error(w, "Forbidden", http.StatusForbidden)
		case model.AppErrCodeConflict:
			// ステータスが不正な場合は変更差分画面にリダイレクト
			http.Redirect(w, r, changesPath, http.StatusSeeOther)
		default:
			slog.ErrorContext(ctx, ae.LogString())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	slog.ErrorContext(ctx, "編集提案ページの編集開始に失敗", "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
