package draft_page

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Update は下書きページを自動保存します (PATCH /s/{space_identifier}/pages/{page_number}/draft_page)
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
	topicNumberStr := r.FormValue("pages_edit_form[topic_number]")
	title := r.FormValue("pages_edit_form[title]")
	body := r.FormValue("pages_edit_form[body]")

	topicNumber, err := strconv.ParseInt(topicNumberStr, 10, 32)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// UseCaseでデータ取得
	output, err := h.getSaveDraftPageDataUC.Execute(ctx, usecase.GetSaveDraftPageDataInput{
		SpaceIdentifier: spaceIdentifier,
		PageNumber:      int32(pageNumber),
		UserID:          user.ID,
		TopicNumber:     int32(topicNumber),
	})
	if err != nil {
		slog.ErrorContext(ctx, "下書き保存データの取得に失敗", "error", err)
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

	// タイトルのポインタ変換（空文字列の場合もポインタとして渡す）
	var titlePtr *string
	if title != "" {
		titlePtr = &title
	}

	// ユースケースを実行
	_, err = h.autoSaveDraftPageUC.Execute(ctx, usecase.AutoSaveDraftPageInput{
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
		slog.ErrorContext(ctx, "下書きの自動保存に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
