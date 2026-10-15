// Package attachmentは公開Web APIの添付ファイルのハンドラーを提供する
package attachment

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerは添付ファイルに関するAPIを担当するハンドラー
type Handler struct {
	getAPIAttachmentUC *usecase.GetAPIAttachmentUsecase
}

// NewHandlerは新しいHandlerを作成する
func NewHandler(getAPIAttachmentUC *usecase.GetAPIAttachmentUsecase) *Handler {
	return &Handler{
		getAPIAttachmentUC: getAPIAttachmentUC,
	}
}

// toAPIAttachmentは添付ファイルの取得結果をAPIの応答の形に変換する。
// 種類が記録されていない添付ファイルは `content_type` を `null` で返す
func toAPIAttachment(output *usecase.GetAPIAttachmentOutput) (apigen.Attachment, error) {
	attachment := output.Attachment
	id, err := uuid.Parse(string(attachment.ID))
	if err != nil {
		return apigen.Attachment{}, fmt.Errorf("添付ファイルのID %q をUUIDとして読めません: %w", attachment.ID, err)
	}

	var contentType *string
	if attachment.ContentType != "" {
		contentType = &attachment.ContentType
	}
	return apigen.Attachment{
		Id:                   id,
		Filename:             attachment.Filename,
		ContentType:          contentType,
		ByteSize:             attachment.ByteSize,
		DownloadUrl:          output.DownloadURL,
		DownloadUrlExpiresAt: output.DownloadURLExpiresAt.UTC(),
	}, nil
}
