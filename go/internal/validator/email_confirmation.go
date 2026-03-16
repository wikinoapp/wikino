package validator

import (
	"context"
	"errors"
	"strings"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
)

// バリデーションのエラー定義
var (
	// ErrEmailConfirmationNotFound はメール確認情報が見つからない場合のエラー
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

// EmailConfirmationCreateValidator はメール確認コード送信のバリデーションを行う
type EmailConfirmationCreateValidator struct {
	userRepo *repository.UserRepository
}

// NewEmailConfirmationCreateValidator は EmailConfirmationCreateValidator を生成する
func NewEmailConfirmationCreateValidator(userRepo *repository.UserRepository) *EmailConfirmationCreateValidator {
	return &EmailConfirmationCreateValidator{userRepo: userRepo}
}

// EmailConfirmationCreateValidatorInput はバリデーションの入力パラメータ
type EmailConfirmationCreateValidatorInput struct {
	Email string
	Event model.EmailConfirmationEvent
}

// EmailConfirmationCreateValidatorResult はバリデーションの結果
type EmailConfirmationCreateValidatorResult struct {
	FormErrors *session.FormErrors
	Err        error
}

// Validate はバリデーションを行う
func (v *EmailConfirmationCreateValidator) Validate(ctx context.Context, input EmailConfirmationCreateValidatorInput) *EmailConfirmationCreateValidatorResult {
	// 1. 形式バリデーション
	formErrors := session.NewFormErrors()

	if input.Email == "" {
		formErrors.AddField("email", i18n.T(ctx, "validation_required"))
	} else if !isValidEmail(input.Email) {
		formErrors.AddField("email", i18n.T(ctx, "validation_email_invalid"))
	}

	if formErrors.HasErrors() {
		return &EmailConfirmationCreateValidatorResult{FormErrors: formErrors}
	}

	// 2. 状態バリデーション（DB検証）
	// signup イベントの場合のみ、メールアドレス重複チェックを行う
	if input.Event != model.EmailConfirmationEventSignUp {
		return &EmailConfirmationCreateValidatorResult{}
	}

	user, err := v.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		return &EmailConfirmationCreateValidatorResult{Err: err}
	}
	if user != nil {
		formErrors.AddField("email", i18n.T(ctx, "validation_email_already_registered"))
		return &EmailConfirmationCreateValidatorResult{
			FormErrors: formErrors,
			Err:        ErrEmailAlreadyRegistered,
		}
	}

	return &EmailConfirmationCreateValidatorResult{}
}

// EmailConfirmationUpdateValidator は確認コード検証のバリデーションを行う
type EmailConfirmationUpdateValidator struct {
	emailConfirmationRepo *repository.EmailConfirmationRepository
}

// NewEmailConfirmationUpdateValidator は EmailConfirmationUpdateValidator を生成する
func NewEmailConfirmationUpdateValidator(emailConfirmationRepo *repository.EmailConfirmationRepository) *EmailConfirmationUpdateValidator {
	return &EmailConfirmationUpdateValidator{emailConfirmationRepo: emailConfirmationRepo}
}

// EmailConfirmationUpdateValidatorInput はバリデーションの入力パラメータ
type EmailConfirmationUpdateValidatorInput struct {
	EmailConfirmationID string
	Code                string
}

// EmailConfirmationUpdateValidatorResult はバリデーションの結果
type EmailConfirmationUpdateValidatorResult struct {
	EmailConfirmation *model.EmailConfirmation
	FormErrors        *session.FormErrors
	Err               error
}

// Validate はバリデーションを行う
func (v *EmailConfirmationUpdateValidator) Validate(ctx context.Context, input EmailConfirmationUpdateValidatorInput) *EmailConfirmationUpdateValidatorResult {
	// 1. 形式バリデーション
	formErrors := session.NewFormErrors()

	if input.Code == "" {
		formErrors.AddField("code", i18n.T(ctx, "validation_required"))
	} else if len(input.Code) != 6 {
		formErrors.AddField("code", i18n.T(ctx, "validation_confirmation_code_invalid_length"))
	}

	if formErrors.HasErrors() {
		return &EmailConfirmationUpdateValidatorResult{FormErrors: formErrors}
	}

	// 2. 状態バリデーション（DB検証）
	// ID でメール確認情報を取得
	confirmation, err := v.emailConfirmationRepo.FindByID(ctx, input.EmailConfirmationID)
	if err != nil {
		return &EmailConfirmationUpdateValidatorResult{Err: err}
	}
	if confirmation == nil {
		formErrors.AddGlobal(i18n.T(ctx, "validation_confirmation_not_found"))
		return &EmailConfirmationUpdateValidatorResult{
			FormErrors: formErrors,
			Err:        ErrEmailConfirmationNotFound,
		}
	}

	// 既に確認済みの場合
	if confirmation.IsSucceeded() {
		return &EmailConfirmationUpdateValidatorResult{
			EmailConfirmation: confirmation,
			Err:               ErrEmailConfirmationAlreadySucceeded,
		}
	}

	// 有効期限チェック（15分）
	if confirmation.IsExpired() {
		formErrors.AddGlobal(i18n.T(ctx, "validation_confirmation_code_expired"))
		return &EmailConfirmationUpdateValidatorResult{
			FormErrors: formErrors,
			Err:        ErrEmailConfirmationExpired,
		}
	}

	// コードの一致チェック（大文字小文字を区別しない）
	if !strings.EqualFold(confirmation.Code, input.Code) {
		formErrors.AddField("code", i18n.T(ctx, "validation_confirmation_code_mismatch"))
		return &EmailConfirmationUpdateValidatorResult{
			FormErrors: formErrors,
			Err:        ErrEmailConfirmationCodeMismatch,
		}
	}

	// 検証成功
	return &EmailConfirmationUpdateValidatorResult{EmailConfirmation: confirmation}
}
