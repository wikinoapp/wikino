// Package attachmentは、ページの本文が参照する添付ファイルの表示 (`/attachments/{id}`) のハンドラーを提供します。
package attachment

import (
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerは添付ファイルの表示のHTTPハンドラーです
type Handler struct {
	getAttachmentUC *usecase.GetAttachmentUsecase
}

// NewHandlerは新しいHandlerを作成します
func NewHandler(getAttachmentUC *usecase.GetAttachmentUsecase) *Handler {
	return &Handler{getAttachmentUC: getAttachmentUC}
}
