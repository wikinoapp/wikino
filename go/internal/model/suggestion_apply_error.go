package model

import "errors"

// SuggestionApplyErrorは編集提案反映時のバリデーションエラーを表す。
// 各SuggestionPageごとのエラーを構造化して保持し、テンプレート側で
// ページタイトル (自動エスケープ) とメッセージ (@templ.RawでHTML展開) を
// 別々にレンダリングできるようにする。
type SuggestionApplyError struct {
	PageErrors []SuggestionApplyPageError
}

func (e *SuggestionApplyError) Error() string { return "suggestion apply validation failed" }

// SuggestionApplyPageErrorは反映対象の1ページ分のバリデーションエラー
type SuggestionApplyPageError struct {
	// PageTitleはSuggestionPage.Title。テンプレート側で自動エスケープされる前提で生文字列を保持する
	PageTitle string
	// MessageはHTMLを含む可能性がある翻訳済みメッセージ。テンプレート側で @templ.Rawで展開する
	Message string
}

// AsSuggestionApplyErrorはerrから *SuggestionApplyErrorを取り出す。
// 取り出せない場合はnilを返す。
func AsSuggestionApplyError(err error) *SuggestionApplyError {
	var ae *SuggestionApplyError
	if errors.As(err, &ae) {
		return ae
	}
	return nil
}
