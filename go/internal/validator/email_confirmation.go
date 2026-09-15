package validator

import (
	"context"
	"strings"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// EmailConfirmationCreateValidatorはメール確認コード送信のバリデーションを行う
type EmailConfirmationCreateValidator struct {
	userRepo *repository.UserRepository
}

// NewEmailConfirmationCreateValidatorはEmailConfirmationCreateValidatorを生成する
func NewEmailConfirmationCreateValidator(userRepo *repository.UserRepository) *EmailConfirmationCreateValidator {
	return &EmailConfirmationCreateValidator{userRepo: userRepo}
}

// EmailConfirmationCreateValidatorInputはバリデーションの入力パラメータ
type EmailConfirmationCreateValidatorInput struct {
	Email string
	Event model.EmailConfirmationEvent
}

// Validateはバリデーションを行う
func (v *EmailConfirmationCreateValidator) Validate(ctx context.Context, input EmailConfirmationCreateValidatorInput) error {
	// 1. 形式バリデーション
	ve := model.NewValidationError()

	if input.Email == "" {
		ve.AddField("email", i18n.T(ctx, "validation_required"))
	} else if !IsValidEmail(input.Email) {
		ve.AddField("email", i18n.T(ctx, "validation_email_invalid"))
	}

	if ve.HasErrors() {
		return ve
	}

	// 2. 状態バリデーション (DB検証)
	// signupイベントの場合のみ、メールアドレス重複チェックを行う
	if input.Event != model.EmailConfirmationEventSignUp {
		return nil
	}

	user, err := v.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		return err
	}
	if user != nil {
		ve.AddField("email", i18n.T(ctx, "validation_email_already_registered"))
		return ve
	}

	return nil
}

// EmailConfirmationUpdateValidatorは確認コード検証のバリデーションを行う
type EmailConfirmationUpdateValidator struct {
	emailConfirmationRepo *repository.EmailConfirmationRepository
}

// NewEmailConfirmationUpdateValidatorはEmailConfirmationUpdateValidatorを生成する
func NewEmailConfirmationUpdateValidator(emailConfirmationRepo *repository.EmailConfirmationRepository) *EmailConfirmationUpdateValidator {
	return &EmailConfirmationUpdateValidator{emailConfirmationRepo: emailConfirmationRepo}
}

// EmailConfirmationUpdateValidatorInputはバリデーションの入力パラメータ
type EmailConfirmationUpdateValidatorInput struct {
	EmailConfirmationID string
	Code                string
}

// Validateはバリデーションを行う
func (v *EmailConfirmationUpdateValidator) Validate(ctx context.Context, input EmailConfirmationUpdateValidatorInput) (*model.EmailConfirmation, error) {
	// 1. 形式バリデーション
	ve := model.NewValidationError()

	if input.Code == "" {
		ve.AddField("code", i18n.T(ctx, "validation_required"))
	} else if len(input.Code) != 6 {
		ve.AddField("code", i18n.T(ctx, "validation_confirmation_code_invalid_length"))
	}

	if ve.HasErrors() {
		return nil, ve
	}

	// 2. 状態バリデーション (DB検証)
	// IDでメール確認情報を取得
	confirmation, err := v.emailConfirmationRepo.FindByID(ctx, input.EmailConfirmationID)
	if err != nil {
		return nil, err
	}
	if confirmation == nil {
		ve.AddGlobal(i18n.T(ctx, "validation_confirmation_not_found"))
		return nil, ve
	}

	// 既に確認済みの場合はAppErrorを返す (Handlerはリダイレクトする)
	if confirmation.IsSucceeded() {
		return nil, &model.AppError{
			Code:    model.AppErrCodeConflict,
			UserMsg: i18n.T(ctx, "validation_confirmation_already_succeeded"),
		}
	}

	// 有効期限チェック (15分)
	if confirmation.IsExpired() {
		ve.AddGlobal(i18n.T(ctx, "validation_confirmation_code_expired"))
		return nil, ve
	}

	// コードの一致チェック (大文字小文字を区別しない)
	if !strings.EqualFold(confirmation.Code, input.Code) {
		ve.AddField("code", i18n.T(ctx, "validation_confirmation_code_mismatch"))
		return nil, ve
	}

	// 検証成功
	return confirmation, nil
}
