// Package page_og_imageはページのog:image用カード画像を配信するエンドポイントのハンドラーを提供します。
//
// ゲストに見せてよいページ (公開トピック・ゴミ箱に入っていない・タイトルあり) だけ、
// スペース名・トピック名・ページタイトルを描いたカード画像をその場で描画して返します。
package page_og_image

import (
	"github.com/wikinoapp/wikino/go/internal/ogcard"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはページのog:image用カード画像の配信ハンドラー
type Handler struct {
	renderer         *ogcard.Renderer
	getPageOgImageUC *usecase.GetPageOgImageUsecase
}

// NewHandlerは新しいHandlerを作成する
func NewHandler(
	renderer *ogcard.Renderer,
	getPageOgImageUC *usecase.GetPageOgImageUsecase,
) *Handler {
	return &Handler{
		renderer:         renderer,
		getPageOgImageUC: getPageOgImageUC,
	}
}
