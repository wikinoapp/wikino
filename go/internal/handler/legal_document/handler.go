// Package legal_documentは、法的文書の固定URLを本文のページへ転送するハンドラーを提供します
package legal_document

// Handlerは法的文書の固定URLのHTTPハンドラーです
type Handler struct{}

// NewHandlerは新しいHandlerを作成します
func NewHandler() *Handler {
	return &Handler{}
}
