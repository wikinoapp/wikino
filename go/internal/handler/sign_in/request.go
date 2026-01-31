package sign_in

import (
	"context"
	"net/mail"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/session"
)

// CreateRequest はログインリクエストのバリデーション用構造体
type CreateRequest struct {
	Email    string
	Password string
}

// NewCreateRequest はリクエストからCreateRequestを生成します
func NewCreateRequest(email, password string) *CreateRequest {
	return &CreateRequest{
		Email:    email,
		Password: password,
	}
}

// Validate はリクエストのバリデーションを行います
// エラーがある場合はFormErrorsを返します
func (r *CreateRequest) Validate(ctx context.Context) *session.FormErrors {
	errors := session.NewFormErrors()

	// メールアドレスのバリデーション
	if r.Email == "" {
		errors.AddField("email", i18n.T(ctx, "validation_required"))
	} else if !isValidEmail(r.Email) {
		errors.AddField("email", i18n.T(ctx, "validation_email_invalid"))
	}

	// パスワードのバリデーション
	if r.Password == "" {
		errors.AddField("password", i18n.T(ctx, "validation_required"))
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}

// isValidEmail はメールアドレスの形式をチェックします
func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
