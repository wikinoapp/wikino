// Package current_space_memberは公開Web APIのトークンの持ち主のメンバーを返すハンドラーを提供する
package current_space_member

import (
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはトークンの持ち主のメンバーに関するAPIを担当するハンドラー
type Handler struct {
	getAPICurrentSpaceMemberUC *usecase.GetAPICurrentSpaceMemberUsecase
}

// NewHandlerは新しいHandlerを作成する
func NewHandler(getAPICurrentSpaceMemberUC *usecase.GetAPICurrentSpaceMemberUsecase) *Handler {
	return &Handler{getAPICurrentSpaceMemberUC: getAPICurrentSpaceMemberUC}
}
