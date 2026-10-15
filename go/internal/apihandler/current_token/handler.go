// Package current_tokenは公開Web APIのリクエストに使ったトークン自身の情報を返すハンドラーを提供する
package current_token

import (
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはリクエストに使ったトークンに関するAPIを担当するハンドラー
type Handler struct {
	getAPICurrentTokenUC *usecase.GetAPICurrentTokenUsecase
}

// NewHandlerは新しいHandlerを作成する
func NewHandler(getAPICurrentTokenUC *usecase.GetAPICurrentTokenUsecase) *Handler {
	return &Handler{getAPICurrentTokenUC: getAPICurrentTokenUC}
}
