package attachment_signed_url

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// createRequestは署名付きURLの一括取得のリクエスト本文
type createRequest struct {
	AttachmentIDs []string `json:"attachment_ids"`
}

// createResponseは署名付きURLの一括取得の応答。signed_urlsは添付ファイルのIDをキーにした署名付きURL
type createResponse struct {
	SignedURLs map[string]string `json:"signed_urls"`
}

// CreateはPOST /attachments/signed_urls - 指定したIDのうち、閲覧者が見られる添付ファイルの署名付きURLを
// まとめて返します。ログインしていなくても、参照するページがすべて公開トピックにある添付ファイルは返します。
// 見られないIDと存在しないIDは、区別せずに応答から省きます。
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req createRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Bad Request"})
		return
	}
	// 本文全体を1つのJSONとして扱い、後ろに別の値や不正な文字列があれば拒否する。
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Bad Request"})
		return
	}

	var userID *model.UserID
	if user := middleware.UserFromContext(ctx); user != nil {
		userID = &user.ID
	}

	attachmentIDs := make([]model.AttachmentID, len(req.AttachmentIDs))
	for i, id := range req.AttachmentIDs {
		attachmentIDs[i] = model.AttachmentID(id)
	}

	output, err := h.getAttachmentSignedURLsUC.Execute(ctx, usecase.GetAttachmentSignedURLsInput{
		AttachmentIDs: attachmentIDs,
		UserID:        userID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "添付ファイルの署名付きURLの一括取得に失敗", "error", err, "attachment_count", len(attachmentIDs))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal Server Error"})
		return
	}

	signedURLs := make(map[string]string, len(output.SignedURLs))
	for id, url := range output.SignedURLs {
		signedURLs[string(id)] = url
	}
	// 閲覧者ごとの判定と期限付きの署名付きURLを返すため、ブラウザにも途中のキャッシュにも保存させない。
	// 署名付きURLは資格情報の代わりになるため、ログに書かない
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, createResponse{SignedURLs: signedURLs})
}

// writeJSONは本文をJSONにして返す
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
