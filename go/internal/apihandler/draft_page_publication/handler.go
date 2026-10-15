// Package draft_page_publicationは公開Web APIの下書きの公開のハンドラーを提供する
package draft_page_publication

import (
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerは下書きの公開に関するAPIを担当するハンドラー
type Handler struct {
	// appURLはWikinoのオリジン (例: `https://example.com`)。応答の `url` を組み立てるのに使う
	appURL                          string
	createAPIDraftPagePublicationUC *usecase.CreateAPIDraftPagePublicationUsecase
}

// NewHandlerは新しいHandlerを作成する
func NewHandler(appURL string, createAPIDraftPagePublicationUC *usecase.CreateAPIDraftPagePublicationUsecase) *Handler {
	return &Handler{
		appURL:                          appURL,
		createAPIDraftPagePublicationUC: createAPIDraftPagePublicationUC,
	}
}
