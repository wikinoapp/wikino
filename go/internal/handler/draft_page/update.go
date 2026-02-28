package draft_page

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// updateResponse は下書き自動保存のJSONレスポンス
type updateResponse struct {
	ModifiedAt   time.Time `json:"modified_at"`
	LinkListHTML string    `json:"link_list_html"`
}

// writeJSONError はJSON形式のエラーレスポンスを返す
func writeJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message}) //nolint:errcheck
}

// Update は下書きページを自動保存します (PATCH /go/s/{space_identifier}/pages/{page_number}/draft_page)
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 認証済みユーザーを取得
	user := middleware.UserFromContext(ctx)
	if user == nil {
		writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// URLパラメータを取得
	spaceIdentifier := chi.URLParam(r, "space_identifier")
	pageNumberStr := chi.URLParam(r, "page_number")

	pageNumber, err := strconv.ParseInt(pageNumberStr, 10, 32)
	if err != nil {
		writeJSONError(w, "Not Found", http.StatusNotFound)
		return
	}

	// スペースを取得
	space, err := h.spaceRepo.FindByIdentifier(ctx, spaceIdentifier)
	if err != nil {
		slog.ErrorContext(ctx, "スペースの取得に失敗", "error", err)
		writeJSONError(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if space == nil {
		writeJSONError(w, "Not Found", http.StatusNotFound)
		return
	}

	// スペースメンバーを取得
	spaceMember, err := h.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, user.ID)
	if err != nil {
		slog.ErrorContext(ctx, "スペースメンバーの取得に失敗", "error", err)
		writeJSONError(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if spaceMember == nil {
		writeJSONError(w, "Not Found", http.StatusNotFound)
		return
	}

	// ページを取得
	pg, err := h.pageRepo.FindBySpaceAndNumber(ctx, space.ID, int32(pageNumber))
	if err != nil {
		slog.ErrorContext(ctx, "ページの取得に失敗", "error", err)
		writeJSONError(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if pg == nil {
		writeJSONError(w, "Not Found", http.StatusNotFound)
		return
	}

	// トピックメンバーを取得してTopicPolicyを生成
	topicMember, err := h.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, pg.TopicID)
	if err != nil {
		slog.ErrorContext(ctx, "トピックメンバーの取得に失敗", "error", err)
		writeJSONError(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	topicPolicy := policy.NewTopicPolicy(spaceMember.Role, space.ID, topicMember, spaceMember.Active)
	if !topicPolicy.CanUpdatePage(pg) {
		writeJSONError(w, "Not Found", http.StatusNotFound)
		return
	}

	// フォームパラメータを取得
	topicNumberStr := r.FormValue("pages_edit_form[topic_number]")
	title := r.FormValue("pages_edit_form[title]")
	body := r.FormValue("pages_edit_form[body]")

	// トピックを取得（topic_numberからTopicIDを解決）
	topicNumber, err := strconv.ParseInt(topicNumberStr, 10, 32)
	if err != nil {
		writeJSONError(w, "Not Found", http.StatusNotFound)
		return
	}

	topic, err := h.topicRepo.FindBySpaceAndNumber(ctx, space.ID, int32(topicNumber))
	if err != nil {
		slog.ErrorContext(ctx, "トピックの取得に失敗", "error", err)
		writeJSONError(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if topic == nil {
		writeJSONError(w, "Not Found", http.StatusNotFound)
		return
	}

	// タイトルのポインタ変換（空文字列の場合もポインタとして渡す）
	var titlePtr *string
	if title != "" {
		titlePtr = &title
	}

	// ユースケースを実行
	output, err := h.autoSaveDraftPageUC.Execute(ctx, usecase.AutoSaveDraftPageInput{
		SpaceID:          space.ID,
		PageID:           pg.ID,
		SpaceMemberID:    spaceMember.ID,
		TopicID:          topic.ID,
		Title:            titlePtr,
		Body:             body,
		SpaceIdentifier:  spaceIdentifier,
		CurrentTopicName: topic.Name,
	})
	if err != nil {
		slog.ErrorContext(ctx, "下書きの自動保存に失敗", "error", err)
		writeJSONError(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// リンク一覧HTMLフラグメントを生成
	var linkListHTML string
	if output.DraftPage != nil && len(output.DraftPage.LinkedPageIDs) > 0 {
		linkedPages, err := h.pageRepo.FindByIDs(ctx, output.DraftPage.LinkedPageIDs, space.ID)
		if err != nil {
			slog.ErrorContext(ctx, "リンク先ページの取得に失敗", "error", err)
			writeJSONError(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		linkListVM := viewmodel.NewLinkList(linkedPages, spaceIdentifier)

		var buf bytes.Buffer
		if err := components.LinkList(linkListVM).Render(ctx, &buf); err != nil {
			slog.ErrorContext(ctx, "リンク一覧のレンダリングに失敗", "error", err)
			writeJSONError(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		linkListHTML = buf.String()
	}

	// JSONレスポンスを返す
	resp := updateResponse{
		ModifiedAt:   output.ModifiedAt,
		LinkListHTML: linkListHTML,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.ErrorContext(ctx, "JSONエンコードに失敗", "error", err)
	}
}
