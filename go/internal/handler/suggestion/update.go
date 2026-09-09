package suggestion

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

	// UseCase を実行
	_, err = h.updateSuggestionUsecase.Execute(ctx, usecase.UpdateSuggestionInput{
		SpaceIdentifier:  spaceIdentifier,
		SuggestionNumber: model.SuggestionNumber(suggestionNumber),
		UserID:           user.ID,
		Title:            title,
		Body:             body,
	})
	if err != nil {
		h.handleUpdateError(w, r, err, user, spaceIdentifier, model.SuggestionNumber(suggestionNumber), title, body)
		return
	}

	// フラッシュメッセージを設定してリダイレクト
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "suggestion_update_success"))
	suggestionPath := string(templates.SuggestionShowPath(viewmodel.NewSpaceIdentifier(spaceIdentifier), int32(suggestionNumber)))
	http.Redirect(w, r, suggestionPath, http.StatusSeeOther)
}

func (h *Handler) handleUpdateError(w http.ResponseWriter, r *http.Request, err error, user *model.User, spaceIdentifier model.SpaceIdentifier, suggestionNumber model.SuggestionNumber, title, body string) {
	ctx := r.Context()

	if ve := model.AsValidationError(err); ve != nil {
		// バリデーションエラー → フォーム再描画
		output, getErr := h.getSuggestionEditUsecase.Execute(ctx, usecase.GetSuggestionEditInput{
			SpaceIdentifier:  spaceIdentifier,
			SuggestionNumber: suggestionNumber,
			UserID:           user.ID,
		})
		if getErr != nil || output == nil {
			slog.ErrorContext(ctx, "フォーム再表示用データの取得に失敗", "error", getErr)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusUnprocessableEntity)
		h.renderEditForm(w, r, user, output, ve, title, body)
		return
	}

	if ae := model.AsAppError(err); ae != nil {
		switch ae.Code {
		case model.AppErrCodeResourceNotFound:
			handler.NotFound(w, r)
		case model.AppErrCodeForbidden:
			handler.NotFound(w, r)
		default:
			slog.ErrorContext(ctx, ae.LogString())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	slog.ErrorContext(ctx, "編集提案の更新に失敗", "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
