package email_confirmation

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
)

// バリデーションのエラー定義
var (
	// ErrEmailConfirmationNotFound は確認情報が見つからない場合のエラー
	ErrEmailConfirmationNotFound = errors.New("メール確認情報が見つかりません")
	// ErrEmailConfirmationAlreadySucceeded は既に確認済みの場合のエラー
	ErrEmailConfirmationAlreadySucceeded = errors.New("このメール確認は既に完了しています")
	// ErrEmailConfirmationExpired は有効期限切れの場合のエラー
	ErrEmailConfirmationExpired = errors.New("確認コードの有効期限が切れています")
	// ErrEmailConfirmationCodeMismatch はコードが一致しない場合のエラー
	ErrEmailConfirmationCodeMismatch = errors.New("確認コードが正しくありません")
	// ErrEmailAlreadyRegistered はメールアドレスが既に登録されている場合のエラー
	ErrEmailAlreadyRegistered = errors.New("このメールアドレスは既に登録されています")
)

// CreateValidator はメール確認コード送信のバリデーションを行う
type CreateValidator struct {
	userRepo *repository.UserRepository
}

// NewCreateValidator は CreateValidator を生成する
func NewCreateValidator(userRepo *repository.UserRepository) *CreateValidator {
	return &CreateValidator{userRepo: userRepo}
}

// CreateValidatorInput はバリデーションの入力パラメータ
type CreateValidatorInput struct {
	Email string
	Event model.EmailConfirmationEvent
}

// CreateValidatorResult はバリデーションの結果
type CreateValidatorResult struct {
	FormErrors *session.FormErrors
	Err        error
}

// Validate はバリデーションを行う
func (v *CreateValidator) Validate(ctx context.Context, input CreateValidatorInput) *CreateValidatorResult {
	// 1. 形式バリデーション
	formErrors := session.NewFormErrors()

	if input.Email == "" {
		formErrors.AddField("email", i18n.T(ctx, "validation_required"))
	} else if !isValidEmail(input.Email) {
		formErrors.AddField("email", i18n.T(ctx, "validation_email_invalid"))
	}

	if formErrors.HasErrors() {
		return &CreateValidatorResult{FormErrors: formErrors}
	}

	// 2. 状態バリデーション（DB検証）
	// signup イベントの場合のみ、メールアドレス重複チェックを行う
	if input.Event != model.EmailConfirmationEventSignUp {
		return &CreateValidatorResult{}
	}

	user, err := v.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		return &CreateValidatorResult{Err: err}
	}
	if user != nil {
		formErrors.AddField("email", i18n.T(ctx, "validation_email_already_registered"))
		return &CreateValidatorResult{
			FormErrors: formErrors,
			Err:        ErrEmailAlreadyRegistered,
		}
	}

	return &CreateValidatorResult{}
}

// UpdateValidator は確認コード検証のバリデーションを行う
type UpdateValidator struct {
	emailConfirmationRepo *repository.EmailConfirmationRepository
}

// NewUpdateValidator は UpdateValidator を生成する
func NewUpdateValidator(emailConfirmationRepo *repository.EmailConfirmationRepository) *UpdateValidator {
	return &UpdateValidator{emailConfirmationRepo: emailConfirmationRepo}
}

// UpdateValidatorInput はバリデーションの入力パラメータ
type UpdateValidatorInput struct {
	EmailConfirmationID string
	Code                string
}

// UpdateValidatorResult はバリデーションの結果
type UpdateValidatorResult struct {
	EmailConfirmation *model.EmailConfirmation
	FormErrors        *session.FormErrors
	Err               error
}

// Validate はバリデーションを行う
func (v *UpdateValidator) Validate(ctx context.Context, input UpdateValidatorInput) *UpdateValidatorResult {
	// 1. 形式バリデーション
	formErrors := session.NewFormErrors()

	if input.Code == "" {
		formErrors.AddField("code", i18n.T(ctx, "validation_required"))
	} else if len(input.Code) != 6 {
		formErrors.AddField("code", i18n.T(ctx, "validation_confirmation_code_invalid_length"))
	}

	if formErrors.HasErrors() {
		return &UpdateValidatorResult{FormErrors: formErrors}
	}

	// 2. 状態バリデーション（DB検証）
	// ID でメール確認情報を取得
	confirmation, err := v.emailConfirmationRepo.FindByID(ctx, input.EmailConfirmationID)
	if err != nil {
		return &UpdateValidatorResult{Err: err}
	}
	if confirmation == nil {
		formErrors.AddGlobal(i18n.T(ctx, "validation_confirmation_not_found"))
		return &UpdateValidatorResult{
			FormErrors: formErrors,
			Err:        ErrEmailConfirmationNotFound,
		}
	}

	// 既に確認済みの場合
	if confirmation.IsSucceeded() {
		return &UpdateValidatorResult{
			EmailConfirmation: confirmation,
			Err:               ErrEmailConfirmationAlreadySucceeded,
		}
	}

	// 有効期限チェック（15分）
	if confirmation.IsExpired() {
		formErrors.AddGlobal(i18n.T(ctx, "validation_confirmation_code_expired"))
		return &UpdateValidatorResult{
			FormErrors: formErrors,
			Err:        ErrEmailConfirmationExpired,
		}
	}

	// コードの一致チェック（大文字小文字を区別しない）
	if !strings.EqualFold(confirmation.Code, input.Code) {
		formErrors.AddField("code", i18n.T(ctx, "validation_confirmation_code_mismatch"))
		return &UpdateValidatorResult{
			FormErrors: formErrors,
			Err:        ErrEmailConfirmationCodeMismatch,
		}
	}

	// 検証成功
	return &UpdateValidatorResult{EmailConfirmation: confirmation}
}

// isValidEmail はメールアドレスの形式をチェックします
func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
