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
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/usecase"
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

	// 編集提案のパスを生成（リダイレクト用）
	suggestionPath := string(templates.SuggestionShowPath(string(spaceIdentifier), int32(suggestionNumber)))

	// UseCaseでデータを取得（編集提案の存在確認と権限チェック用データ）
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

	// クローズ権限チェック（作成者本人、スペースオーナー、またはトピック管理者）
	topicPolicy := policy.NewTopicPolicy(detailOutput.SpaceMember, detailOutput.TopicMember)
	if !topicPolicy.CanCloseSuggestion(detailOutput.Suggestion) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// べき等性: 既にクローズ済みの場合は何もせず成功リダイレクト
	if detailOutput.Suggestion.Status == model.SuggestionStatusClosed {
		h.flashMgr.SetSuccess(w, i18n.T(ctx, "suggestion_close_success"))
		http.Redirect(w, r, suggestionPath, http.StatusSeeOther)
		return
	}

	// オープンステータスでなければクローズ不可
	if detailOutput.Suggestion.Status != model.SuggestionStatusOpen {
		h.flashMgr.SetError(w, i18n.T(ctx, "suggestion_close_error"))
		http.Redirect(w, r, suggestionPath, http.StatusSeeOther)
		return
	}

	// 編集提案をクローズ
	_, err = h.closeSuggestionUsecase.Execute(ctx, usecase.CloseSuggestionInput{
		Suggestion: detailOutput.Suggestion,
	})
	if err != nil {
		slog.ErrorContext(ctx, "編集提案のクローズに失敗", "error", err)
		h.flashMgr.SetError(w, i18n.T(ctx, "suggestion_close_error"))
		http.Redirect(w, r, suggestionPath, http.StatusSeeOther)
		return
	}

	// フラッシュメッセージを設定してリダイレクト
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "suggestion_close_success"))
	http.Redirect(w, r, suggestionPath, http.StatusSeeOther)
}
