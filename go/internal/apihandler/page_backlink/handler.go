// Package page_backlinkは公開Web APIのページのバックリンク一覧のハンドラーを提供する
package page_backlink

import (
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはページのバックリンク一覧に関するAPIを担当するハンドラー
type Handler struct {
	// appURLはWikinoのオリジン (例: `https://example.com`)。応答の `url` を組み立てるのに使う
	appURL                 string
	listAPIPageBacklinksUC *usecase.ListAPIPageBacklinksUsecase
}

// NewHandlerは新しいHandlerを作成する
func NewHandler(appURL string, listAPIPageBacklinksUC *usecase.ListAPIPageBacklinksUsecase) *Handler {
	return &Handler{
		appURL:                 appURL,
		listAPIPageBacklinksUC: listAPIPageBacklinksUC,
	}
}
