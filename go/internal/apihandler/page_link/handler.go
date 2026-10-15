// Package page_linkは公開Web APIのページのリンク一覧のハンドラーを提供する
package page_link

import (
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはページのリンク一覧に関するAPIを担当するハンドラー
type Handler struct {
	// appURLはWikinoのオリジン (例: `https://example.com`)。応答の `url` を組み立てるのに使う
	appURL             string
	listAPIPageLinksUC *usecase.ListAPIPageLinksUsecase
}

// NewHandlerは新しいHandlerを作成する
func NewHandler(appURL string, listAPIPageLinksUC *usecase.ListAPIPageLinksUsecase) *Handler {
	return &Handler{
		appURL:             appURL,
		listAPIPageLinksUC: listAPIPageLinksUC,
	}
}
