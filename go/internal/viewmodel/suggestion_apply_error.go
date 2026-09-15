package viewmodel

import "github.com/wikinoapp/wikino/go/internal/model"

// SuggestionApplyErrorは編集提案反映時のバリデーションエラーをテンプレート向けに表したものです
type SuggestionApplyError struct {
	PageErrors []SuggestionApplyPageError
}

// SuggestionApplyPageErrorは反映対象の1ページ分のバリデーションエラー
type SuggestionApplyPageError struct {
	// PageTitleはSuggestionPage.Title。テンプレート側で自動エスケープされる前提で生文字列を保持する
	PageTitle string
	// MessageはHTMLを含む可能性がある翻訳済みメッセージ。テンプレート側で @templ.Rawで展開する
	Message string
}

// NewSuggestionApplyErrorはmodel.SuggestionApplyErrorからViewModelを生成します
func NewSuggestionApplyError(src *model.SuggestionApplyError) *SuggestionApplyError {
	if src == nil {
		return nil
	}
	pageErrors := make([]SuggestionApplyPageError, len(src.PageErrors))
	for i, pe := range src.PageErrors {
		pageErrors[i] = SuggestionApplyPageError{
			PageTitle: pe.PageTitle,
			Message:   pe.Message,
		}
	}
	return &SuggestionApplyError{PageErrors: pageErrors}
}
