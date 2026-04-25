package model

import "errors"

// SuggestionApplyError は編集提案反映時のバリデーションエラーを表す。
// 各 SuggestionPage ごとのエラーを構造化して保持し、テンプレート側で
// ページタイトル（自動エスケープ）とメッセージ（@templ.Raw で HTML 展開）を
// 別々にレンダリングできるようにする。
type SuggestionApplyError struct {
	PageErrors []SuggestionApplyPageError
}

func (e *SuggestionApplyError) Error() string { return "suggestion apply validation failed" }

// SuggestionApplyPageError は反映対象の 1 ページ分のバリデーションエラー
type SuggestionApplyPageError struct {
	// PageTitle は SuggestionPage.Title。テンプレート側で自動エスケープされる前提で生文字列を保持する
	PageTitle string
	// Message は HTML を含む可能性がある翻訳済みメッセージ。テンプレート側で @templ.Raw で展開する
	Message string
}

// AsSuggestionApplyError は err から *SuggestionApplyError を取り出す。
// 取り出せない場合は nil を返す。
func AsSuggestionApplyError(err error) *SuggestionApplyError {
	var ae *SuggestionApplyError
	if errors.As(err, &ae) {
		return ae
	}
	return nil
}
