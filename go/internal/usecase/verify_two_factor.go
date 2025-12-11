package usecase

import (
	"context"
	"time"

	"github.com/pquerna/otp/totp"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// VerifyTwoFactorError は2FA検証エラーの種類
type VerifyTwoFactorError string

const (
	// ErrTwoFactorNotEnabled は2FAが有効でない場合のエラー
	ErrTwoFactorNotEnabled VerifyTwoFactorError = "two_factor_not_enabled"
	// ErrInvalidTOTPCode はTOTPコードが無効な場合のエラー
	ErrInvalidTOTPCode VerifyTwoFactorError = "invalid_totp_code"
	// ErrInvalidRecoveryCode はリカバリーコードが無効な場合のエラー
	ErrInvalidRecoveryCode VerifyTwoFactorError = "invalid_recovery_code"
)

func (e VerifyTwoFactorError) Error() string {
	return string(e)
}

// VerifyTwoFactorUsecase は2FA検証ユースケース
type VerifyTwoFactorUsecase struct {
	userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository
}

// NewVerifyTwoFactorUsecase は VerifyTwoFactorUsecase を生成する
func NewVerifyTwoFactorUsecase(
	userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository,
) *VerifyTwoFactorUsecase {
	return &VerifyTwoFactorUsecase{
		userTwoFactorAuthRepo: userTwoFactorAuthRepo,
	}
}

// VerifyTOTPInput はTOTPコード検証の入力パラメータ
type VerifyTOTPInput struct {
	UserID   string
	TOTPCode string
}

// VerifyTOTP はTOTPコードを検証する
func (uc *VerifyTwoFactorUsecase) VerifyTOTP(ctx context.Context, input VerifyTOTPInput) error {
	twoFactorAuth, err := uc.userTwoFactorAuthRepo.FindEnabledByUserID(ctx, input.UserID)
	if err != nil {
		return err
	}
	if twoFactorAuth == nil {
		return ErrTwoFactorNotEnabled
	}

	// TOTPコードを検証
	// Rails版と同様に前後15秒のドリフトを許容
	valid := totp.Validate(input.TOTPCode, twoFactorAuth.Secret)
	if !valid {
		// ドリフトを考慮した検証
		valid = validateWithDrift(input.TOTPCode, twoFactorAuth.Secret, 15)
	}

	if !valid {
		return ErrInvalidTOTPCode
	}

	return nil
}

// validateWithDrift は前後のタイムステップも考慮してTOTPコードを検証する
func validateWithDrift(code, secret string, driftSeconds int) bool {
	now := time.Now()

	// 現在時刻で検証
	if totp.Validate(code, secret) {
		return true
	}

	// 前のタイムステップで検証
	opts := totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    6,
		Algorithm: 0, // SHA1 (default)
	}

	// 前後のタイムステップで検証
	pastTime := now.Add(-time.Duration(driftSeconds) * time.Second)
	if valid, _ := totp.ValidateCustom(code, secret, pastTime, opts); valid {
		return true
	}

	futureTime := now.Add(time.Duration(driftSeconds) * time.Second)
	if valid, _ := totp.ValidateCustom(code, secret, futureTime, opts); valid {
		return true
	}

	return false
}

// VerifyRecoveryCodeInput はリカバリーコード検証の入力パラメータ
type VerifyRecoveryCodeInput struct {
	UserID       string
	RecoveryCode string
}

// VerifyRecoveryCode はリカバリーコードを検証し、使用済みにする
func (uc *VerifyTwoFactorUsecase) VerifyRecoveryCode(ctx context.Context, input VerifyRecoveryCodeInput) error {
	twoFactorAuth, err := uc.userTwoFactorAuthRepo.FindEnabledByUserID(ctx, input.UserID)
	if err != nil {
		return err
	}
	if twoFactorAuth == nil {
		return ErrTwoFactorNotEnabled
	}

	// リカバリーコードを検証
	if !isValidRecoveryCode(twoFactorAuth, input.RecoveryCode) {
		return ErrInvalidRecoveryCode
	}

	// リカバリーコードを消費（リストから削除）
	newRecoveryCodes := removeRecoveryCode(twoFactorAuth.RecoveryCodes, input.RecoveryCode)
	err = uc.userTwoFactorAuthRepo.UpdateRecoveryCodes(ctx, input.UserID, newRecoveryCodes)
	if err != nil {
		return err
	}

	return nil
}

// isValidRecoveryCode はリカバリーコードが有効かどうかを確認する
func isValidRecoveryCode(twoFactorAuth *model.UserTwoFactorAuth, code string) bool {
	for _, recoveryCode := range twoFactorAuth.RecoveryCodes {
		if recoveryCode == code {
			return true
		}
	}
	return false
}

// removeRecoveryCode はリカバリーコードをリストから削除する
func removeRecoveryCode(codes []string, codeToRemove string) []string {
	result := make([]string, 0, len(codes))
	for _, code := range codes {
		if code != codeToRemove {
			result = append(result, code)
		}
	}
	return result
}
