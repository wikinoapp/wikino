package validator

import (
	"context"
	"regexp"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/session"
)

var passwordResetEmailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// PasswordResetCreateValidator はパスワードリセット申請のバリデーションを行う
type PasswordResetCreateValidator struct{}

// NewPasswordResetCreateValidator は PasswordResetCreateValidator を生成する
func NewPasswordResetCreateValidator() *PasswordResetCreateValidator {
	return &PasswordResetCreateValidator{}
}

// PasswordResetCreateValidatorInput はバリデーションの入力パラメータ
type PasswordResetCreateValidatorInput struct {
	Email string
}

// PasswordResetCreateValidatorResult はバリデーションの結果
type PasswordResetCreateValidatorResult struct {
	FormErrors *session.FormErrors
}

// Validate はバリデーションを行う
func (v *PasswordResetCreateValidator) Validate(ctx context.Context, input PasswordResetCreateValidatorInput) *PasswordResetCreateValidatorResult {
	formErrors := session.NewFormErrors()

	// メールアドレス必須チェック
	if input.Email == "" {
		formErrors.AddField("email", i18n.T(ctx, "validation_required"))
		return &PasswordResetCreateValidatorResult{FormErrors: formErrors}
	}

	// フォーマットチェック
	if !passwordResetEmailRegex.MatchString(input.Email) {
		formErrors.AddField("email", i18n.T(ctx, "validation_email_invalid"))
	}

	if formErrors.HasErrors() {
		return &PasswordResetCreateValidatorResult{FormErrors: formErrors}
	}

	return &PasswordResetCreateValidatorResult{}
}
