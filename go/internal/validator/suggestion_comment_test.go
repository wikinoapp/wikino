package validator_test

import (
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestSuggestionCommentCreateValidator_本文が空の場合エラーになる(t *testing.T) {
	t.Parallel()

	v := validator.NewSuggestionCommentCreateValidator()
	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)

	err := v.Validate(ctx, validator.SuggestionCommentCreateValidatorInput{
		Body: "",
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("expected ValidationError, got nil")
	}
	if !ve.HasFieldError("body") {
		t.Error("expected body field error")
	}
}

func TestSuggestionCommentCreateValidator_本文が長すぎる場合エラーになる(t *testing.T) {
	t.Parallel()

	v := validator.NewSuggestionCommentCreateValidator()
	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)

	longBody := strings.Repeat("あ", 10001)
	err := v.Validate(ctx, validator.SuggestionCommentCreateValidatorInput{
		Body: longBody,
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("expected ValidationError, got nil")
	}
	if !ve.HasFieldError("body") {
		t.Error("expected body field error")
	}
}

func TestSuggestionCommentCreateValidator_有効な入力の場合エラーにならない(t *testing.T) {
	t.Parallel()

	v := validator.NewSuggestionCommentCreateValidator()
	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)

	err := v.Validate(ctx, validator.SuggestionCommentCreateValidatorInput{
		Body: "コメントの本文です",
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSuggestionCommentCreateValidator_最大文字数ちょうどの場合エラーにならない(t *testing.T) {
	t.Parallel()

	v := validator.NewSuggestionCommentCreateValidator()
	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)

	body := strings.Repeat("あ", 10000)
	err := v.Validate(ctx, validator.SuggestionCommentCreateValidatorInput{
		Body: body,
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSuggestionCommentUpdateValidator_本文が空の場合エラーになる(t *testing.T) {
	t.Parallel()

	v := validator.NewSuggestionCommentUpdateValidator()
	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)

	err := v.Validate(ctx, validator.SuggestionCommentUpdateValidatorInput{
		Body: "",
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("expected ValidationError, got nil")
	}
	if !ve.HasFieldError("body") {
		t.Error("expected body field error")
	}
}

func TestSuggestionCommentUpdateValidator_本文が長すぎる場合エラーになる(t *testing.T) {
	t.Parallel()

	v := validator.NewSuggestionCommentUpdateValidator()
	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)

	longBody := strings.Repeat("あ", 10001)
	err := v.Validate(ctx, validator.SuggestionCommentUpdateValidatorInput{
		Body: longBody,
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("expected ValidationError, got nil")
	}
	if !ve.HasFieldError("body") {
		t.Error("expected body field error")
	}
}

func TestSuggestionCommentUpdateValidator_有効な入力の場合エラーにならない(t *testing.T) {
	t.Parallel()

	v := validator.NewSuggestionCommentUpdateValidator()
	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)

	err := v.Validate(ctx, validator.SuggestionCommentUpdateValidatorInput{
		Body: "更新後のコメント本文",
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
