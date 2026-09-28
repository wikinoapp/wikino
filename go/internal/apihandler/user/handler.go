// Package userは公開Web APIのトークンの持ち主のユーザーを返すハンドラーを提供する
package user

// Handlerはトークンの持ち主のユーザーに関するAPIを担当するハンドラー
type Handler struct{}

// NewHandlerは新しいHandlerを作成する
func NewHandler() *Handler {
	return &Handler{}
}
