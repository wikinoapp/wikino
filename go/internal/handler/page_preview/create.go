package page_preview

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	pagepages "github.com/wikinoapp/wikino/go/internal/templates/pages/page"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Create はフォームの現在値からプレビュー用 HTML フラグメントを生成して返します
// (POST /s/{space_identifier}/pages/{page_number}/preview)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
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
		handler.NotFound(w, r)
		return
	}

	// フォーム値を取得
	title := r.FormValue("title")
	body := r.FormValue("body")

	// UseCase 呼び出し (認可・レンダリングはすべて UseCase 内で実行)
	output, err := h.getPagePreviewUC.Execute(ctx, usecase.GetPagePreviewInput{
		SpaceIdentifier: spaceIdentifier,
		PageNumber:      int32(pageNumber),
		UserID:          user.ID,
		Title:           title,
		Body:            body,
	})
	if err != nil {
		if ae := model.AsAppError(err); ae != nil {
			switch ae.Code {
			case model.AppErrCodeResourceNotFound, model.AppErrCodeForbidden:
				// Treat non-members and missing pages as 404, same as the edit screen.
				// [Ja] 非メンバー・存在しないページは編集画面と同じく 404 として扱う。
				handler.NotFound(w, r)
			default:
				slog.ErrorContext(ctx, "プレビューの生成に失敗", "error", ae.LogString())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}
		slog.ErrorContext(ctx, "プレビューの生成に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// プレビュー用 HTML フラグメントをレンダリング
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pagepages.Preview(pagepages.PreviewData{
		Title:    output.Title,
		BodyHTML: output.BodyHTML,
	}).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "プレビューフラグメントのレンダリングに失敗", "error", err)
	}
}
