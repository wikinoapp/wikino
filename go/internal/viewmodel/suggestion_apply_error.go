package viewmodel

import "github.com/wikinoapp/wikino/go/internal/model"

// SuggestionApplyError は編集提案反映時のバリデーションエラーをテンプレート向けに表したものです
type SuggestionApplyError struct {
	PageErrors []SuggestionApplyPageError
}

// SuggestionApplyPageError は反映対象の 1 ページ分のバリデーションエラー
type SuggestionApplyPageError struct {
	// PageTitle は SuggestionPage.Title。テンプレート側で自動エスケープされる前提で生文字列を保持する
	PageTitle string
	// Message は HTML を含む可能性がある翻訳済みメッセージ。テンプレート側で @templ.Raw で展開する
	Message string
}

// NewSuggestionApplyError は model.SuggestionApplyError から ViewModel を生成します
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
