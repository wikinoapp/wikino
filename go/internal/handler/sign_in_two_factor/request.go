package sign_in_two_factor

import (
	"context"
	"regexp"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/session"
)

// TOTPコードは6桁の数字のみ
var totpCodeRegex = regexp.MustCompile(`^\d{6}$`)

// CreateRequest は2FAコード検証リクエストのバリデーション用構造体
type CreateRequest struct {
	TOTPCode string
}

// NewCreateRequest はリクエストからCreateRequestを生成します
func NewCreateRequest(totpCode string) *CreateRequest {
	return &CreateRequest{
		TOTPCode: totpCode,
	}
}

// Validate はリクエストのバリデーションを行います
// エラーがある場合はFormErrorsを返します
func (r *CreateRequest) Validate(ctx context.Context) *session.FormErrors {
	errors := session.NewFormErrors()

	// TOTPコードのバリデーション
	if r.TOTPCode == "" {
		errors.AddField("totp_code", i18n.T(ctx, "validation_required"))
	} else if !totpCodeRegex.MatchString(r.TOTPCode) {
		errors.AddField("totp_code", i18n.T(ctx, "validation_totp_code_invalid"))
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}
