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
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Create は編集提案にページを追加します (POST /s/{space_identifier}/suggestions/{suggestion_number}/suggestion_pages)
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

	// フォームデータをパース
	if err := r.ParseForm(); err != nil {
		slog.ErrorContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	draftPageIDStrs := r.Form["draft_page_ids"]

	// 下書きページIDをドメイン型に変換
	draftPageIDs := make([]model.DraftPageID, len(draftPageIDStrs))
	for i, id := range draftPageIDStrs {
		draftPageIDs[i] = model.DraftPageID(id)
	}

	// UseCase を実行
	_, err = h.addSuggestionPageUsecase.Execute(ctx, usecase.AddSuggestionPageInput{
		SpaceIdentifier:  spaceIdentifier,
		SuggestionNumber: suggestionNumber,
		UserID:           user.ID,
		DraftPageIDs:     draftPageIDs,
	})
	if err != nil {
		h.handleCreateError(w, r, err, user, spaceIdentifier, suggestionNumber)
		return
	}

	// フラッシュメッセージを設定してリダイレクト
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_suggestion_page_added"))
	changesPath := string(templates.SuggestionChangesPath(viewmodel.NewSpaceIdentifier(spaceIdentifier), int32(suggestionNumber)))
	http.Redirect(w, r, changesPath, http.StatusSeeOther)
}

func (h *Handler) handleCreateError(w http.ResponseWriter, r *http.Request, err error, user *model.User, spaceIdentifier model.SpaceIdentifier, suggestionNumber model.SuggestionNumber) {
	ctx := r.Context()

	if ve := model.AsValidationError(err); ve != nil {
		// バリデーションエラー → フォーム再描画
		output, getErr := h.getSuggestionPageNewUsecase.Execute(ctx, usecase.GetSuggestionPageNewInput{
			SpaceIdentifier:  spaceIdentifier,
			SuggestionNumber: suggestionNumber,
			UserID:           user.ID,
		})
		if getErr != nil {
			slog.ErrorContext(ctx, "フォーム再表示用データの取得に失敗", "error", getErr)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusUnprocessableEntity)
		h.renderNewForm(w, r, user, spaceIdentifier, output, ve)
		return
	}

	if ae := model.AsAppError(err); ae != nil {
		switch ae.Code {
		case model.AppErrCodeResourceNotFound, model.AppErrCodeForbidden:
			handler.NotFound(w, r)
		default:
			slog.ErrorContext(ctx, ae.LogString())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	slog.ErrorContext(ctx, "編集提案ページの追加に失敗", "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
