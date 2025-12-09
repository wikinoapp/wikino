// Package health はヘルスチェックエンドポイントのハンドラーを提供します
package health

// Handler はヘルスチェックエンドポイントのハンドラーです
type Handler struct{}

// NewHandler は新しいHealthハンドラーを作成します
func NewHandler() *Handler {
	return &Handler{}
}
