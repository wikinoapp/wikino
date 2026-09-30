// Package spaceは公開Web APIのスペースのハンドラーを提供する
package space

import (
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはスペースに関するAPIを担当するハンドラー
type Handler struct {
	getAPISpaceUC *usecase.GetAPISpaceUsecase
}

// NewHandlerは新しいHandlerを作成する
func NewHandler(getAPISpaceUC *usecase.GetAPISpaceUsecase) *Handler {
	return &Handler{getAPISpaceUC: getAPISpaceUC}
}
