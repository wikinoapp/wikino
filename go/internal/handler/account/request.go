package account

import (
	"context"
	"regexp"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/session"
)

// アットネームのバリデーション定数
const (
	AtnameMaxLength = 20
)

// アットネームの形式（英数字とアンダースコアのみ）
var atnameRegex = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// パスワードのバリデーション定数
const (
	PasswordMinLength = 8
)

// CreateRequest はアカウント作成リクエストのバリデーション用構造体
type CreateRequest struct {
	Atname   string
	Password string
}

// NewCreateRequest はリクエストからCreateRequestを生成します
func NewCreateRequest(atname, password string) *CreateRequest {
	return &CreateRequest{
		Atname:   atname,
		Password: password,
	}
}

// Validate はリクエストのバリデーションを行います
// エラーがある場合はFormErrorsを返します
func (r *CreateRequest) Validate(ctx context.Context) *session.FormErrors {
	errors := session.NewFormErrors()

	// アットネームのバリデーション
	if r.Atname == "" {
		errors.AddField("atname", i18n.T(ctx, "validation_atname_required"))
	} else {
		if len(r.Atname) > AtnameMaxLength {
			errors.AddField("atname", i18n.T(ctx, "validation_atname_too_long"))
		}
		if !atnameRegex.MatchString(r.Atname) {
			errors.AddField("atname", i18n.T(ctx, "validation_atname_invalid_format"))
		}
	}

	// パスワードのバリデーション
	if r.Password == "" {
		errors.AddField("password", i18n.T(ctx, "validation_password_required"))
	} else if len(r.Password) < PasswordMinLength {
		errors.AddField("password", i18n.T(ctx, "validation_password_too_short"))
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}
