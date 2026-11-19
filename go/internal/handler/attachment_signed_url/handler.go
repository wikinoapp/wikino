// Package attachment_signed_urlは、ページの本文に添付ファイルを表示するための署名付きURLの一括取得
// (`POST /attachments/signed_urls`) のハンドラーを提供します。
package attachment_signed_url

import (
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerは署名付きURLの一括取得のHTTPハンドラーです
type Handler struct {
	getAttachmentSignedURLsUC *usecase.GetAttachmentSignedURLsUsecase
}

// NewHandlerは新しいHandlerを作成します
func NewHandler(getAttachmentSignedURLsUC *usecase.GetAttachmentSignedURLsUsecase) *Handler {
	return &Handler{getAttachmentSignedURLsUC: getAttachmentSignedURLsUC}
}
