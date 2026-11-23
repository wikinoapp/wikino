// Package robots_txtはrobots.txt (/robots.txt) のハンドラーを提供します
package robots_txt

// Handlerはrobots.txtのHTTPハンドラーです
type Handler struct{}

// NewHandlerは新しいHandlerを作成します
func NewHandler() *Handler {
	return &Handler{}
}
