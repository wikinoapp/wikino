package validator

import (
	"context"
	"unicode/utf8"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/session"
)

const suggestionCommentBodyMaxLength = 10000

// SuggestionCommentCreateValidator は編集提案コメント作成のバリデーションを行う
type SuggestionCommentCreateValidator struct{}

// NewSuggestionCommentCreateValidator は SuggestionCommentCreateValidator を生成する
func NewSuggestionCommentCreateValidator() *SuggestionCommentCreateValidator {
	return &SuggestionCommentCreateValidator{}
}

// SuggestionCommentCreateValidatorInput はバリデーションの入力パラメータ
type SuggestionCommentCreateValidatorInput struct {
	Body string
}

// SuggestionCommentCreateValidatorResult はバリデーションの結果
type SuggestionCommentCreateValidatorResult struct {
	FormErrors *session.FormErrors
}

// Validate はバリデーションを行う
func (v *SuggestionCommentCreateValidator) Validate(ctx context.Context, input SuggestionCommentCreateValidatorInput) *SuggestionCommentCreateValidatorResult {
	formErrors := session.NewFormErrors()

	if input.Body == "" {
		formErrors.AddField("body", i18n.T(ctx, "validation_suggestion_comment_body_required"))
	}

	if input.Body != "" && utf8.RuneCountInString(input.Body) > suggestionCommentBodyMaxLength {
		formErrors.AddField("body", i18n.T(ctx, "validation_suggestion_comment_body_too_long"))
	}

	return &SuggestionCommentCreateValidatorResult{FormErrors: formErrors}
}

// SuggestionCommentUpdateValidator は編集提案コメント更新のバリデーションを行う
type SuggestionCommentUpdateValidator struct{}

// NewSuggestionCommentUpdateValidator は SuggestionCommentUpdateValidator を生成する
func NewSuggestionCommentUpdateValidator() *SuggestionCommentUpdateValidator {
	return &SuggestionCommentUpdateValidator{}
}

// SuggestionCommentUpdateValidatorInput はバリデーションの入力パラメータ
type SuggestionCommentUpdateValidatorInput struct {
	Body string
}

// SuggestionCommentUpdateValidatorResult はバリデーションの結果
type SuggestionCommentUpdateValidatorResult struct {
	FormErrors *session.FormErrors
}

// Validate はバリデーションを行う
func (v *SuggestionCommentUpdateValidator) Validate(ctx context.Context, input SuggestionCommentUpdateValidatorInput) *SuggestionCommentUpdateValidatorResult {
	formErrors := session.NewFormErrors()

	if input.Body == "" {
		formErrors.AddField("body", i18n.T(ctx, "validation_suggestion_comment_body_required"))
	}

	if input.Body != "" && utf8.RuneCountInString(input.Body) > suggestionCommentBodyMaxLength {
		formErrors.AddField("body", i18n.T(ctx, "validation_suggestion_comment_body_too_long"))
	}

	return &SuggestionCommentUpdateValidatorResult{FormErrors: formErrors}
}
