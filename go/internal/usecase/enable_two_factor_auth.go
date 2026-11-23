package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// errTwoFactorAuthNotPendingは、有効にしようとした時点で登録の途中の設定が無いことを表す内部エラー
var errTwoFactorAuthNotPending = errors.New("登録の途中の二要素認証の設定がありません")

// EnableTwoFactorAuthUsecaseは、現在のパスワードと認証アプリのコードを照合して二要素認証を有効にし、
// リカバリーコードを発行する
type EnableTwoFactorAuthUsecase struct {
	createValidator       *validator.SettingsTwoFactorAuthCreateValidator
	userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository
}

// NewEnableTwoFactorAuthUsecaseはEnableTwoFactorAuthUsecaseを生成する
func NewEnableTwoFactorAuthUsecase(
	createValidator *validator.SettingsTwoFactorAuthCreateValidator,
	userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository,
) *EnableTwoFactorAuthUsecase {
	return &EnableTwoFactorAuthUsecase{
		createValidator:       createValidator,
		userTwoFactorAuthRepo: userTwoFactorAuthRepo,
	}
}

// EnableTwoFactorAuthInputはEnableTwoFactorAuthUsecase.Executeの入力
type EnableTwoFactorAuthInput struct {
	UserID   model.UserID
	Password string
	// TOTPCodeは入力されたままのコード。貼り付けで混じった空白はここで取り除く
	TOTPCode string
}

// Executeは登録の途中の秘密鍵でコードを照合し、二要素認証を有効にしてリカバリーコードを保存する。
//
// パスワードとコードの誤りは*model.ValidationErrorで返し、登録の途中の秘密鍵は作り直さない。
// 登録の途中の設定が無いとき (既に有効にした・照合のあとに別の画面で秘密鍵が差し替わった) は、
// AppErrCodeConflictの*model.AppErrorを返す。
// 照合と保存は1つの更新で行うため、トランザクションを開かない。
func (uc *EnableTwoFactorAuthUsecase) Execute(ctx context.Context, input EnableTwoFactorAuthInput) error {
	pending, err := uc.userTwoFactorAuthRepo.FindByUserID(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("二要素認証の設定の取得に失敗: %w", err)
	}
	if pending == nil || pending.Enabled {
		return notPendingTwoFactorAuthError(ctx, input.UserID)
	}

	if err := uc.createValidator.Validate(ctx, validator.SettingsTwoFactorAuthCreateValidatorInput{
		UserID:   input.UserID,
		Password: input.Password,
		TOTPCode: strings.Join(strings.Fields(input.TOTPCode), ""),
		Secret:   pending.Secret,
	}); err != nil {
		return err
	}

	recoveryCodes, err := auth.GenerateRecoveryCodes()
	if err != nil {
		return fmt.Errorf("リカバリーコードの生成に失敗: %w", err)
	}

	// 照合した秘密鍵の行だけを有効にし、照合のあとに秘密鍵が差し替わっていれば有効にしない
	enabled, err := uc.userTwoFactorAuthRepo.Enable(ctx, input.UserID, pending.Secret, recoveryCodes)
	if err != nil {
		return fmt.Errorf("二要素認証の有効化に失敗: %w", err)
	}
	if !enabled {
		return notPendingTwoFactorAuthError(ctx, input.UserID)
	}

	return nil
}

// notPendingTwoFactorAuthErrorは、登録の途中の設定が無いことを表すAppErrCodeConflictの*model.AppErrorを返す。
// ハンドラーはこのエラーで二要素認証の設定の画面へ送り、今の状態を見せる。
func notPendingTwoFactorAuthError(ctx context.Context, userID model.UserID) *model.AppError {
	return &model.AppError{
		Code:     model.AppErrCodeConflict,
		UserMsg:  i18n.T(ctx, "flash_two_factor_auth_not_pending"),
		Internal: errTwoFactorAuthNotPending,
		Metadata: map[string]string{"user_id": userID.String()},
	}
}
