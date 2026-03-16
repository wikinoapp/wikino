package draft_page_revision

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/usecase"
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

	// UseCaseでデータを取得
	output, err := h.getPageDetailUC.Execute(ctx, usecase.GetPageDetailInput{
		SpaceIdentifier: spaceIdentifier,
		PageNumber:      int32(pageNumber),
		UserID:          user.ID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "ページ詳細の取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if output == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// 認可チェック
	topicPolicy := policy.NewTopicPolicy(output.SpaceMember, output.TopicMember)
	if !topicPolicy.CanUpdatePage(output.Page) {
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

	// ユースケースを実行
	_, err = h.manualSaveDraftPageUC.Execute(ctx, usecase.ManualSaveDraftPageInput{
		SpaceID:          output.Space.ID,
		PageID:           output.Page.ID,
		SpaceMemberID:    output.SpaceMember.ID,
		TopicID:          output.Topic.ID,
		Title:            titlePtr,
		Body:             body,
		SpaceIdentifier:  spaceIdentifier,
		CurrentTopicName: output.Topic.Name,
	})
	if err != nil {
		slog.ErrorContext(ctx, "下書きの手動保存に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_draft_page_saved"))
	http.Redirect(w, r, "/drafts", http.StatusSeeOther)
}
