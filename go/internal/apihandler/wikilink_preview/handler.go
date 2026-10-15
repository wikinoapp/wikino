// Package wikilink_previewは公開Web APIの本文のWikiリンクのプレビューのハンドラーを提供する
package wikilink_preview

import (
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerは本文のWikiリンクのプレビューに関するAPIを担当するハンドラー
type Handler struct {
	// appURLはWikinoのオリジン (例: `https://example.com`)。応答の `url` を組み立てるのに使う
	appURL                  string
	getAPIWikilinkPreviewUC *usecase.GetAPIWikilinkPreviewUsecase
}

// NewHandlerは新しいHandlerを作成する
func NewHandler(appURL string, getAPIWikilinkPreviewUC *usecase.GetAPIWikilinkPreviewUsecase) *Handler {
	return &Handler{
		appURL:                  appURL,
		getAPIWikilinkPreviewUC: getAPIWikilinkPreviewUC,
	}
}
