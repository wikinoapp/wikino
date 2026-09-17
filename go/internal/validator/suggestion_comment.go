package validator

import (
	"context"
	"unicode/utf8"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
)

const suggestionCommentBodyMaxLength = 10000

// SuggestionCommentCreateValidatorは編集提案コメント作成のバリデーションを行う
type SuggestionCommentCreateValidator struct{}

// NewSuggestionCommentCreateValidatorはSuggestionCommentCreateValidatorを生成する
func NewSuggestionCommentCreateValidator() *SuggestionCommentCreateValidator {
	return &SuggestionCommentCreateValidator{}
}

// SuggestionCommentCreateValidatorInputはバリデーションの入力パラメータ
type SuggestionCommentCreateValidatorInput struct {
	Body string
}

// Validateはバリデーションを行う
func (v *SuggestionCommentCreateValidator) Validate(ctx context.Context, input SuggestionCommentCreateValidatorInput) error {
	ve := model.NewValidationError()

	if input.Body == "" {
		ve.AddField("body", i18n.T(ctx, "validation_suggestion_comment_body_required"))
	}

	if input.Body != "" && utf8.RuneCountInString(input.Body) > suggestionCommentBodyMaxLength {
		ve.AddField("body", i18n.T(ctx, "validation_suggestion_comment_body_too_long"))
	}

	if ve.HasErrors() {
		return ve
	}

	return nil
}

// SuggestionCommentUpdateValidatorは編集提案コメント更新のバリデーションを行う
type SuggestionCommentUpdateValidator struct{}

// NewSuggestionCommentUpdateValidatorはSuggestionCommentUpdateValidatorを生成する
func NewSuggestionCommentUpdateValidator() *SuggestionCommentUpdateValidator {
	return &SuggestionCommentUpdateValidator{}
}

// SuggestionCommentUpdateValidatorInputはバリデーションの入力パラメータ
type SuggestionCommentUpdateValidatorInput struct {
	Body string
}

// Validateはバリデーションを行う
func (v *SuggestionCommentUpdateValidator) Validate(ctx context.Context, input SuggestionCommentUpdateValidatorInput) error {
	ve := model.NewValidationError()

	if input.Body == "" {
		ve.AddField("body", i18n.T(ctx, "validation_suggestion_comment_body_required"))
	}

	if input.Body != "" && utf8.RuneCountInString(input.Body) > suggestionCommentBodyMaxLength {
		ve.AddField("body", i18n.T(ctx, "validation_suggestion_comment_body_too_long"))
	}

	if ve.HasErrors() {
		return ve
	}

	return nil
}
