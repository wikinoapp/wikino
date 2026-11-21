// Package apple_touch_iconはルート直下で配信するApple Touch Iconのハンドラーを提供します
package apple_touch_icon

import "io/fs"

// HandlerはルートのApple Touch IconのHTTPハンドラーです
type Handler struct {
	staticFS fs.FS
}

// NewHandlerは新しいHandlerを作成します。staticFSは `/static` で配信するディレクトリです
func NewHandler(staticFS fs.FS) *Handler {
	return &Handler{
		staticFS: staticFS,
	}
}
