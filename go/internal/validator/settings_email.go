package validator

import (
	"context"
	"strings"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// SettingsEmailUpdateValidatorはメールアドレスの変更で、確認コードを送る前の入力を検証する
type SettingsEmailUpdateValidator struct {
	userRepo         *repository.UserRepository
	userPasswordRepo *repository.UserPasswordRepository
}

// NewSettingsEmailUpdateValidatorはSettingsEmailUpdateValidatorを生成する
func NewSettingsEmailUpdateValidator(
	userRepo *repository.UserRepository,
	userPasswordRepo *repository.UserPasswordRepository,
) *SettingsEmailUpdateValidator {
	return &SettingsEmailUpdateValidator{
		userRepo:         userRepo,
		userPasswordRepo: userPasswordRepo,
	}
}

// SettingsEmailUpdateValidatorInputはバリデーションの入力パラメータ
type SettingsEmailUpdateValidatorInput struct {
	UserID   model.UserID
	NewEmail string
	Password string
}

// Validateは新しいメールアドレスの形式と現在のパスワードを確かめ、パスワードが合っているときだけ
// 新しいメールアドレスが登録済みでないかを確かめる。パスワードを知らない人に、メールアドレスが
// 登録済みかどうかを教えないため。
func (v *SettingsEmailUpdateValidator) Validate(ctx context.Context, input SettingsEmailUpdateValidatorInput) error {
	ve := model.NewValidationError()

	// 表示名付きや前後に空白のある値は、ParseAddressが受理しても書かれた文字列のまま保存されて
	// ログインで一致しなくなるため、アドレスだけの値に限る
	if input.NewEmail == "" {
		ve.AddField("new_email", i18n.T(ctx, "validation_required"))
	} else if _, canonical := CanonicalEmail(input.NewEmail); !canonical {
		ve.AddField("new_email", i18n.T(ctx, "validation_email_invalid"))
	}
	if input.Password == "" {
		ve.AddField("password", i18n.T(ctx, "validation_required"))
	}
	if ve.HasErrors() {
		return ve
	}

	userPassword, err := v.userPasswordRepo.FindByUserID(ctx, input.UserID)
	if err != nil {
		return err
	}
	if userPassword == nil || !auth.VerifyPassword(userPassword.PasswordDigest, input.Password) {
		ve.AddField("password", i18n.T(ctx, "validation_current_password_invalid"))
		return ve
	}

	exists, err := v.userRepo.ExistsByEmail(ctx, input.NewEmail)
	if err != nil {
		return err
	}
	if exists {
		ve.AddField("new_email", i18n.T(ctx, "validation_email_already_registered"))
		return ve
	}

	return nil
}

// SettingsEmailConfirmationUpdateValidatorは、メールアドレスの変更の確認コードを検証する。
// 新規登録の確認 (EmailConfirmationUpdateValidator) と違い、メールアドレスの変更として始めた
// 確認だけを受け付け、確定の時点でも新しいメールアドレスが登録済みでないかを確かめる。
type SettingsEmailConfirmationUpdateValidator struct {
	emailConfirmationRepo *repository.EmailConfirmationRepository
	userRepo              *repository.UserRepository
}

// NewSettingsEmailConfirmationUpdateValidatorはSettingsEmailConfirmationUpdateValidatorを生成する
func NewSettingsEmailConfirmationUpdateValidator(
	emailConfirmationRepo *repository.EmailConfirmationRepository,
	userRepo *repository.UserRepository,
) *SettingsEmailConfirmationUpdateValidator {
	return &SettingsEmailConfirmationUpdateValidator{
		emailConfirmationRepo: emailConfirmationRepo,
		userRepo:              userRepo,
	}
}

// SettingsEmailConfirmationUpdateValidatorInputはバリデーションの入力パラメータ
type SettingsEmailConfirmationUpdateValidatorInput struct {
	EmailConfirmationID string
	Code                string
}

// Validateは確認コードを検証し、成功したら確認を返す
func (v *SettingsEmailConfirmationUpdateValidator) Validate(ctx context.Context, input SettingsEmailConfirmationUpdateValidatorInput) (*model.EmailConfirmation, error) {
	ve := model.NewValidationError()

	if input.Code == "" {
		ve.AddField("code", i18n.T(ctx, "validation_required"))
	} else if len(input.Code) != 6 {
		ve.AddField("code", i18n.T(ctx, "validation_confirmation_code_invalid_length"))
	}
	if ve.HasErrors() {
		return nil, ve
	}

	confirmation, err := v.emailConfirmationRepo.FindByID(ctx, input.EmailConfirmationID)
	if err != nil {
		return nil, err
	}
	// 新規登録やパスワードリセットの確認では、メールアドレスを変更させない
	if confirmation == nil || confirmation.Event != model.EmailConfirmationEventEmailUpdate {
		ve.AddGlobal(i18n.T(ctx, "validation_confirmation_not_found"))
		return nil, ve
	}
	if confirmation.IsSucceeded() {
		ve.AddGlobal(i18n.T(ctx, "validation_confirmation_already_succeeded"))
		return nil, ve
	}
	if confirmation.IsExpired() {
		ve.AddGlobal(i18n.T(ctx, "validation_confirmation_code_expired"))
		return nil, ve
	}
	if !strings.EqualFold(confirmation.Code, input.Code) {
		ve.AddField("code", i18n.T(ctx, "validation_confirmation_code_mismatch"))
		return nil, ve
	}

	// 確認コードを待つ間に、ほかのユーザーが同じメールアドレスで登録しうる
	exists, err := v.userRepo.ExistsByEmail(ctx, confirmation.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		ve.AddGlobal(i18n.T(ctx, "validation_email_already_registered"))
		return nil, ve
	}

	return confirmation, nil
}
