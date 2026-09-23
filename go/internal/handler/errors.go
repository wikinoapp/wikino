// Package handlerはHTTPハンドラーの共通関数を提供する。
package handler

import (
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/templates/components"
	errpages "github.com/wikinoapp/wikino/go/internal/templates/pages/errors"
)

// NotFoundはスタイル付きの404ページをレンダリングする
func NotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	// テンプレートのレンダリングエラーはレスポンス書き込み後なので無視
	_ = errpages.NotFoundPage().Render(r.Context(), w)
}

// RelatedPageListNotFoundはhtmxには局所的な404フラグメント、通常のブラウザリクエストには
// 完全な404文書を返し、htmx 4がページネーション領域へ文書をネストすることを防ぐ。
func RelatedPageListNotFound(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("HX-Request") != "true" {
		NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_ = components.RelatedPageListNotFound().Render(r.Context(), w)
}
