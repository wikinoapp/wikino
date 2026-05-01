package draft_page_revision

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
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Update は下書きページを手動保存します (PATCH /s/{space_identifier}/pages/{page_number}/draft_page_revision)
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 認証済みユーザーを取得
	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// URLパラメータを取得
	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))
	pageNumberStr := chi.URLParam(r, "page_number")

	pageNumber, err := strconv.ParseInt(pageNumberStr, 10, 32)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// フォームパラメータを取得
	title := r.FormValue("title")
	body := r.FormValue("body")

	// タイトルのポインタ変換
	var titlePtr *string
	if title != "" {
		titlePtr = &title
	}

	// UseCase を実行
	saveOutput, err := h.manualSaveDraftPageUC.Execute(ctx, usecase.ManualSaveDraftPageInput{
		SpaceIdentifier: spaceIdentifier,
		PageNumber:      int32(pageNumber),
		UserID:          user.ID,
		Title:           titlePtr,
		Body:            body,
	})
	if err != nil {
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

		slog.ErrorContext(ctx, "下書きの手動保存に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// リダイレクト先を決定
	redirectTo := r.URL.Query().Get("redirect_to")
	if redirectTo == "suggestion_new" && saveOutput.DraftPage != nil {
		suggestionNewPath := fmt.Sprintf("%s?draft_page_ids=%s",
			string(templates.SuggestionNewPath(viewmodel.NewSpaceIdentifier(spaceIdentifier), saveOutput.TopicNumber)),
			string(saveOutput.DraftPage.ID),
		)
		http.Redirect(w, r, suggestionNewPath, http.StatusSeeOther)
		return
	}

	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_draft_page_saved"))
	http.Redirect(w, r, "/drafts", http.StatusSeeOther)
}
