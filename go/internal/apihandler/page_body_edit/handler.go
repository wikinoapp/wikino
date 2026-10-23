// Package page_body_editは公開Web APIのページの本文の検索置換のハンドラーを提供する
package page_body_edit

import (
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはページの本文の検索置換に関するAPIを担当するハンドラー
type Handler struct {
	// appURLはWikinoのオリジン (例: `https://example.com`)。応答の `url` を組み立てるのに使う
	appURL                  string
	createAPIPageBodyEditUC *usecase.CreateAPIPageBodyEditUsecase
}

// NewHandlerは新しいHandlerを作成する
func NewHandler(appURL string, createAPIPageBodyEditUC *usecase.CreateAPIPageBodyEditUsecase) *Handler {
	return &Handler{
		appURL:                  appURL,
		createAPIPageBodyEditUC: createAPIPageBodyEditUC,
	}
}
