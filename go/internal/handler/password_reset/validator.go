package password_reset

import (
	"context"
	"regexp"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/session"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// CreateValidator はパスワードリセット申請のバリデーションを行う
type CreateValidator struct{}

// NewCreateValidator は CreateValidator を生成する
func NewCreateValidator() *CreateValidator {
	return &CreateValidator{}
}

// CreateValidatorInput はバリデーションの入力パラメータ
type CreateValidatorInput struct {
	Email string
}

// CreateValidatorResult はバリデーションの結果
type CreateValidatorResult struct {
	FormErrors *session.FormErrors
}

// Validate はバリデーションを行う
func (v *CreateValidator) Validate(ctx context.Context, input CreateValidatorInput) *CreateValidatorResult {
	formErrors := session.NewFormErrors()

	// メールアドレス必須チェック
	if input.Email == "" {
		formErrors.AddField("email", i18n.T(ctx, "validation_required"))
		return &CreateValidatorResult{FormErrors: formErrors}
	}

	// フォーマットチェック
	if !emailRegex.MatchString(input.Email) {
		formErrors.AddField("email", i18n.T(ctx, "validation_email_invalid"))
	}

	if formErrors.HasErrors() {
		return &CreateValidatorResult{FormErrors: formErrors}
	}

	return &CreateValidatorResult{}
}
