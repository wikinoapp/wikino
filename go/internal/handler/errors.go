// Package handler はHTTPハンドラーの共通関数を提供する。
package handler

import (
	"net/http"

	errpages "github.com/wikinoapp/wikino/go/internal/templates/pages/errors"
)

// NotFound はスタイル付きの404ページをレンダリングする
func NotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	// テンプレートのレンダリングエラーはレスポンス書き込み後なので無視
	_ = errpages.NotFoundPage().Render(r.Context(), w)
}
