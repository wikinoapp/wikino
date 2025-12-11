package sign_in_two_factor_recovery

import (
	"context"
	"regexp"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/session"
)

// リカバリーコードは8文字の英小文字と数字のみ
var recoveryCodeRegex = regexp.MustCompile(`^[a-z0-9]{8}$`)

// CreateRequest はリカバリーコード検証リクエストのバリデーション用構造体
type CreateRequest struct {
	RecoveryCode string
}

// NewCreateRequest はリクエストからCreateRequestを生成します
func NewCreateRequest(recoveryCode string) *CreateRequest {
	return &CreateRequest{
		RecoveryCode: recoveryCode,
	}
}

// Validate はリクエストのバリデーションを行います
// エラーがある場合はFormErrorsを返します
func (r *CreateRequest) Validate(ctx context.Context) *session.FormErrors {
	errors := session.NewFormErrors()

	// リカバリーコードのバリデーション
	if r.RecoveryCode == "" {
		errors.AddField("recovery_code", i18n.T(ctx, "validation_required"))
	} else if !recoveryCodeRegex.MatchString(r.RecoveryCode) {
		errors.AddField("recovery_code", i18n.T(ctx, "validation_recovery_code_invalid"))
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}
