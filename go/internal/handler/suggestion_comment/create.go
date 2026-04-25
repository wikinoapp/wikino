package suggestion_comment

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

// Create は編集提案コメントを作成します (POST /s/{space_identifier}/suggestions/{suggestion_number}/comments)
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

	body := r.FormValue("body")

	// 編集提案のパスを生成（リダイレクト用）
	suggestionPath := string(templates.SuggestionShowPath(string(spaceIdentifier), int32(suggestionNumber)))

	// UseCase を実行
	_, err = h.createSuggestionCommentUsecase.Execute(ctx, usecase.CreateSuggestionCommentInput{
		SpaceIdentifier:  spaceIdentifier,
		SuggestionNumber: suggestionNumber,
		UserID:           user.ID,
		Body:             body,
	})
	if err != nil {
		h.handleError(w, r, err, suggestionPath)
		return
	}

	// フラッシュメッセージを設定してリダイレクト
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "suggestion_comment_create_success"))
	http.Redirect(w, r, suggestionPath, http.StatusSeeOther)
}

func (h *Handler) handleError(w http.ResponseWriter, r *http.Request, err error, suggestionPath string) {
	ctx := r.Context()

	if ve := model.AsValidationError(err); ve != nil {
		errs := ve.GetFieldErrors("body")
		if len(errs) > 0 {
			h.flashMgr.SetError(w, errs[0])
		}
		http.Redirect(w, r, suggestionPath, http.StatusSeeOther)
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

	slog.ErrorContext(ctx, "編集提案コメントの作成に失敗", "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
