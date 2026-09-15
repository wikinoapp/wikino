// Package healthはヘルスチェックエンドポイントのハンドラーを提供します
package health

// Handlerはヘルスチェックエンドポイントのハンドラーです
type Handler struct{}

// NewHandlerは新しいHealthハンドラーを作成します
func NewHandler() *Handler {
	return &Handler{}
}
