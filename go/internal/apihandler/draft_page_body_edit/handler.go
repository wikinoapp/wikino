// Package draft_page_body_editは公開Web APIの下書きの本文の検索置換のハンドラーを提供する
package draft_page_body_edit

import (
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerは下書きの本文の検索置換に関するAPIを担当するハンドラー
type Handler struct {
	// appURLはWikinoのオリジン (例: `https://example.com`)。応答の `url` を組み立てるのに使う
	appURL                       string
	createAPIDraftPageBodyEditUC *usecase.CreateAPIDraftPageBodyEditUsecase
}

// NewHandlerは新しいHandlerを作成する
func NewHandler(appURL string, createAPIDraftPageBodyEditUC *usecase.CreateAPIDraftPageBodyEditUsecase) *Handler {
	return &Handler{
		appURL:                       appURL,
		createAPIDraftPageBodyEditUC: createAPIDraftPageBodyEditUC,
	}
}
