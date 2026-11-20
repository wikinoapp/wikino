// Package attachment_presignは、エディタにドロップしたファイルをオブジェクトストレージへ直接アップロード
// するための事前署名 (`POST /s/{space_identifier}/attachments/presign`) のハンドラーを提供します。
package attachment_presign

import (
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerは添付ファイルのアップロードの事前署名のHTTPハンドラーです
type Handler struct {
	createAttachmentPresignUC *usecase.CreateAttachmentPresignUsecase
}

// NewHandlerは新しいHandlerを作成します
func NewHandler(createAttachmentPresignUC *usecase.CreateAttachmentPresignUsecase) *Handler {
	return &Handler{createAttachmentPresignUC: createAttachmentPresignUC}
}
