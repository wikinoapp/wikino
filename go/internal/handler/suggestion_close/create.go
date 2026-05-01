package suggestion_close

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
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Create は編集提案をクローズします (POST /s/{space_identifier}/suggestions/{suggestion_number}/close)
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

	// UseCaseを実行
	_, err = h.closeSuggestionUsecase.Execute(ctx, usecase.CloseSuggestionInput{
		SpaceIdentifier:  spaceIdentifier,
		SuggestionNumber: suggestionNumber,
		UserID:           user.ID,
	})
	if err != nil {
		h.handleCreateError(w, r, err, spaceIdentifier, suggestionNumber)
		return
	}

	// フラッシュメッセージを設定してリダイレクト
	suggestionPath := string(templates.SuggestionShowPath(viewmodel.NewSpaceIdentifier(spaceIdentifier), int32(suggestionNumber)))
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "suggestion_close_success"))
	http.Redirect(w, r, suggestionPath, http.StatusSeeOther)
}

func (h *Handler) handleCreateError(w http.ResponseWriter, r *http.Request, err error, spaceIdentifier model.SpaceIdentifier, suggestionNumber model.SuggestionNumber) {
	ctx := r.Context()
	suggestionPath := string(templates.SuggestionShowPath(viewmodel.NewSpaceIdentifier(spaceIdentifier), int32(suggestionNumber)))

	if ae := model.AsAppError(err); ae != nil {
		switch ae.Code {
		case model.AppErrCodeResourceNotFound:
			handler.NotFound(w, r)
		case model.AppErrCodeForbidden:
			http.Error(w, "Forbidden", http.StatusForbidden)
		case model.AppErrCodeConflict:
			h.flashMgr.SetError(w, ae.UserMsg)
			http.Redirect(w, r, suggestionPath, http.StatusSeeOther)
		default:
			slog.ErrorContext(ctx, ae.LogString())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	slog.ErrorContext(ctx, "編集提案のクローズに失敗", "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
