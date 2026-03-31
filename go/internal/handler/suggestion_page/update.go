package suggestion_page

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

	// UseCaseを実行
	_, err = h.updateSuggestionPageUsecase.Execute(ctx, usecase.UpdateSuggestionPageInput{
		SpaceIdentifier:  spaceIdentifier,
		SuggestionNumber: suggestionNumber,
		SuggestionPageID: suggestionPageID,
		UserID:           user.ID,
	})
	if err != nil {
		h.handleUpdateError(w, r, err, spaceIdentifier, suggestionNumber)
		return
	}

	// フラッシュメッセージを設定してリダイレクト
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_suggestion_page_updated"))
	changesPath := string(templates.SuggestionChangesPath(string(spaceIdentifier), int32(suggestionNumber)))
	http.Redirect(w, r, changesPath, http.StatusSeeOther)
}

func (h *Handler) handleUpdateError(w http.ResponseWriter, r *http.Request, err error, spaceIdentifier model.SpaceIdentifier, suggestionNumber model.SuggestionNumber) {
	ctx := r.Context()

	if ae := model.AsAppError(err); ae != nil {
		switch ae.Code {
		case model.AppErrCodeResourceNotFound, model.AppErrCodeForbidden:
			handler.NotFound(w, r)
		case model.AppErrCodeConflict:
			slog.WarnContext(ctx, ae.LogString())
			h.flashMgr.SetError(w, ae.UserMsg)
			changesPath := string(templates.SuggestionChangesPath(string(spaceIdentifier), int32(suggestionNumber)))
			http.Redirect(w, r, changesPath, http.StatusSeeOther)
		default:
			slog.ErrorContext(ctx, ae.LogString())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	slog.ErrorContext(ctx, "編集提案ページの更新に失敗", "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
