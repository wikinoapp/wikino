// Package handler はHTTPハンドラーの共通関数を提供する。
package handler

import (
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/templates/components"
	errpages "github.com/wikinoapp/wikino/go/internal/templates/pages/errors"
)

// NotFound はスタイル付きの404ページをレンダリングする
func NotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	// テンプレートのレンダリングエラーはレスポンス書き込み後なので無視
	_ = errpages.NotFoundPage().Render(r.Context(), w)
}

// RelatedPageListNotFound returns a local 404 fragment to htmx and the complete 404 document to a
// normal browser request, preventing htmx 4 from nesting a document inside a pagination target.
//
// [Ja] RelatedPageListNotFound は htmx には局所的な 404 フラグメント、通常のブラウザリクエストには
// 完全な 404 文書を返し、htmx 4 がページネーション領域へ文書をネストすることを防ぐ。
func RelatedPageListNotFound(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("HX-Request") != "true" {
		NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_ = components.RelatedPageListNotFound().Render(r.Context(), w)
}
