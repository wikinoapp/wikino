// Package faviconはルート直下のファビコン (/favicon.ico) のハンドラーを提供します
package favicon

import "io/fs"

// Handlerはルート直下のファビコンのHTTPハンドラーです
type Handler struct {
	staticFS fs.FS
}

// NewHandlerは新しいHandlerを作成します。staticFSは `/static` で配信するディレクトリです
func NewHandler(staticFS fs.FS) *Handler {
	return &Handler{
		staticFS: staticFS,
	}
}
