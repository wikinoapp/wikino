package attachment_presign

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// createRequestは事前署名のリクエスト本文
type createRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	ByteSize    *int64 `json:"byte_size"`
	Checksum    string `json:"checksum"`
}

// createResponseは事前署名の応答。キーはエディタのJSが読む名前にする
type createResponse struct {
	DirectUploadURL     string            `json:"directUploadUrl"`
	DirectUploadHeaders map[string]string `json:"directUploadHeaders"`
	AttachmentID        string            `json:"attachmentId"`
}

// validationErrorResponseは検証に失敗したときの応答。メッセージの配列を返す
type validationErrorResponse struct {
	Errors []string `json:"errors"`
}

// presignFieldOrderは、検証エラーのメッセージを並べる順。項目の宣言の順にする
var presignFieldOrder = map[string]int{"filename": 0, "content_type": 1, "byte_size": 2, "checksum": 3}

// CreateはPOST /s/{space_identifier}/attachments/presign - スペースでattachment:writeを持つメンバーのために、
// 添付ファイルを処理待ちの状態で作り、ブラウザがオブジェクトストレージへ直接アップロードするための
// 署名付きURLとヘッダーを返します。
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return
	}

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

	output, err := h.createAttachmentPresignUC.Execute(ctx, usecase.CreateAttachmentPresignInput{
		SpaceIdentifier: model.SpaceIdentifier(chi.URLParam(r, "space_identifier")),
		UserID:          user.ID,
		Filename:        req.Filename,
		ContentType:     req.ContentType,
		ByteSize:        req.ByteSize,
		Checksum:        req.Checksum,
	})
	if err != nil {
		if ve := model.AsValidationError(err); ve != nil {
			writeJSON(w, http.StatusUnprocessableEntity, validationErrorResponse{Errors: validationMessages(ve)})
			return
		}
		if ae := model.AsAppError(err); ae != nil {
			switch ae.Code {
			case model.AppErrCodeResourceNotFound:
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "Not Found"})
				return
			case model.AppErrCodeForbidden:
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "Unauthorized"})
				return
			}
		}
		slog.ErrorContext(ctx, "添付ファイルのアップロードの事前署名に失敗", "error", err, "user_id", user.ID.String())
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal Server Error"})
		return
	}

	// 期限付きの署名付きURLを返すため、ブラウザにも途中のキャッシュにも保存させない。
	// 署名付きURLは資格情報の代わりになるため、ログに書かない
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, createResponse{
		DirectUploadURL:     output.DirectUploadURL,
		DirectUploadHeaders: output.DirectUploadHeaders,
		AttachmentID:        string(output.AttachmentID),
	})
}

// validationMessagesは、検証エラーのメッセージを項目の順に並べる
func validationMessages(ve *model.ValidationError) []string {
	fields := make([]string, 0, len(ve.Fields))
	for field := range ve.Fields {
		fields = append(fields, field)
	}
	sort.Slice(fields, func(i, j int) bool { return presignFieldOrder[fields[i]] < presignFieldOrder[fields[j]] })

	messages := append([]string{}, ve.Global...)
	for _, field := range fields {
		messages = append(messages, ve.Fields[field]...)
	}
	return messages
}

// writeJSONは本文をJSONにして返す
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
