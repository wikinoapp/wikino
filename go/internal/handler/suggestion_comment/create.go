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
	"github.com/wikinoapp/wikino/go/internal/validator"
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

	// バリデーション
	validationResult := h.createValidator.Validate(ctx, validator.SuggestionCommentCreateValidatorInput{
		Body: body,
	})

	// 編集提案のパスを生成（リダイレクト用）
	suggestionPath := string(templates.SuggestionShowPath(string(spaceIdentifier), int32(suggestionNumber)))

	if validationResult.FormErrors.HasErrors() {
		errs := validationResult.FormErrors.GetFieldErrors("body")
		if len(errs) > 0 {
			h.flashMgr.SetError(w, errs[0])
		}
		http.Redirect(w, r, suggestionPath, http.StatusSeeOther)
		return
	}

	// UseCaseでデータを取得（編集提案の存在確認と権限チェック）
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

	// スペースメンバーのみコメント可能
	if detailOutput.SpaceMember == nil {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// コメントを作成
	_, err = h.createSuggestionCommentUsecase.Execute(ctx, usecase.CreateSuggestionCommentInput{
		SpaceID:       detailOutput.Space.ID,
		SuggestionID:  detailOutput.Suggestion.ID,
		SpaceMemberID: detailOutput.SpaceMember.ID,
		Body:          body,
	})
	if err != nil {
		slog.ErrorContext(ctx, "編集提案コメントの作成に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// フラッシュメッセージを設定してリダイレクト
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "suggestion_comment_create_success"))
	http.Redirect(w, r, suggestionPath, http.StatusSeeOther)
}
