package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"

	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// テスト用のシークレットを生成する
func generateTestSecret(t *testing.T) string {
	t.Helper()
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Wikino",
		AccountName: "test@example.com",
	})
	if err != nil {
		t.Fatalf("シークレット生成に失敗: %v", err)
	}
	return key.Secret()
}

// テスト用の有効なTOTPコードを生成する
func generateValidTOTPCode(t *testing.T, secret string) string {
	t.Helper()
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("TOTPコード生成に失敗: %v", err)
	}
	return code
}

func TestVerifyTwoFactorUsecase_VerifyTOTP(t *testing.T) {
	t.Parallel()

	t.Run("有効なTOTPコードで検証に成功する", func(t *testing.T) {
		_, tx := testutil.SetupTestDB(t)
		q := testutil.QueriesWithTx(tx)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
		uc := NewVerifyTwoFactorUsecase(userTwoFactorAuthRepo)

		secret := generateTestSecret(t)
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("totp-valid@example.com").
			WithAtname("totpvalid").
			BuildWithTwoFactorAuth(secret, true)

		code := generateValidTOTPCode(t, secret)

		err := uc.VerifyTOTP(context.Background(), VerifyTOTPInput{
			UserID:   userID,
			TOTPCode: code,
		})
		if err != nil {
			t.Errorf("VerifyTOTP() error = %v, want nil", err)
		}
	})

	t.Run("無効なTOTPコードで検証に失敗する", func(t *testing.T) {
		_, tx := testutil.SetupTestDB(t)
		q := testutil.QueriesWithTx(tx)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
		uc := NewVerifyTwoFactorUsecase(userTwoFactorAuthRepo)

		secret := generateTestSecret(t)
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("totp-invalid@example.com").
			WithAtname("totpinvalid").
			BuildWithTwoFactorAuth(secret, true)

		err := uc.VerifyTOTP(context.Background(), VerifyTOTPInput{
			UserID:   userID,
			TOTPCode: "000000", // 無効なコード
		})
		if !errors.Is(err, ErrInvalidTOTPCode) {
			t.Errorf("VerifyTOTP() error = %v, want %v", err, ErrInvalidTOTPCode)
		}
	})

	t.Run("2FAが有効でないユーザーの場合はエラーを返す", func(t *testing.T) {
		_, tx := testutil.SetupTestDB(t)
		q := testutil.QueriesWithTx(tx)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
		uc := NewVerifyTwoFactorUsecase(userTwoFactorAuthRepo)

		secret := generateTestSecret(t)
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("totp-disabled@example.com").
			WithAtname("totpdisabled").
			BuildWithTwoFactorAuth(secret, false) // 2FAが無効

		code := generateValidTOTPCode(t, secret)

		err := uc.VerifyTOTP(context.Background(), VerifyTOTPInput{
			UserID:   userID,
			TOTPCode: code,
		})
		if !errors.Is(err, ErrTwoFactorNotEnabled) {
			t.Errorf("VerifyTOTP() error = %v, want %v", err, ErrTwoFactorNotEnabled)
		}
	})

	t.Run("2FAが設定されていないユーザーの場合はエラーを返す", func(t *testing.T) {
		_, tx := testutil.SetupTestDB(t)
		q := testutil.QueriesWithTx(tx)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
		uc := NewVerifyTwoFactorUsecase(userTwoFactorAuthRepo)

		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("totp-nosetup@example.com").
			WithAtname("totpnosetup").
			Build() // 2FA設定なし

		err := uc.VerifyTOTP(context.Background(), VerifyTOTPInput{
			UserID:   userID,
			TOTPCode: "123456",
		})
		if !errors.Is(err, ErrTwoFactorNotEnabled) {
			t.Errorf("VerifyTOTP() error = %v, want %v", err, ErrTwoFactorNotEnabled)
		}
	})
}

func TestVerifyTwoFactorUsecase_VerifyRecoveryCode(t *testing.T) {
	t.Parallel()

	t.Run("有効なリカバリーコードで検証に成功し、コードが消費される", func(t *testing.T) {
		_, tx := testutil.SetupTestDB(t)
		q := testutil.QueriesWithTx(tx)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
		uc := NewVerifyTwoFactorUsecase(userTwoFactorAuthRepo)

		secret := generateTestSecret(t)
		recoveryCodes := []string{"abc12345", "def67890", "ghi11111"}
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("recovery-valid@example.com").
			WithAtname("recoveryvalid").
			BuildWithTwoFactorAuthAndRecoveryCodes(secret, true, recoveryCodes)

		// 最初のリカバリーコードで検証
		err := uc.VerifyRecoveryCode(context.Background(), VerifyRecoveryCodeInput{
			UserID:       userID,
			RecoveryCode: "abc12345",
		})
		if err != nil {
			t.Errorf("VerifyRecoveryCode() error = %v, want nil", err)
		}

		// リカバリーコードが消費されていることを確認
		twoFactorAuth, err := userTwoFactorAuthRepo.FindEnabledByUserID(context.Background(), userID)
		if err != nil {
			t.Fatalf("FindEnabledByUserID() error = %v", err)
		}
		if twoFactorAuth == nil {
			t.Fatal("FindEnabledByUserID() returned nil")
		}
		if len(twoFactorAuth.RecoveryCodes) != 2 {
			t.Errorf("RecoveryCodes length = %d, want 2", len(twoFactorAuth.RecoveryCodes))
		}
		// abc12345 が削除されていることを確認
		for _, code := range twoFactorAuth.RecoveryCodes {
			if code == "abc12345" {
				t.Error("RecoveryCodes still contains 'abc12345' after consumption")
			}
		}
	})

	t.Run("同じリカバリーコードは一度しか使えない", func(t *testing.T) {
		_, tx := testutil.SetupTestDB(t)
		q := testutil.QueriesWithTx(tx)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
		uc := NewVerifyTwoFactorUsecase(userTwoFactorAuthRepo)

		secret := generateTestSecret(t)
		recoveryCodes := []string{"unique123", "unique456"}
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("recovery-twice@example.com").
			WithAtname("recoverytwice").
			BuildWithTwoFactorAuthAndRecoveryCodes(secret, true, recoveryCodes)

		// 1回目: 成功
		err := uc.VerifyRecoveryCode(context.Background(), VerifyRecoveryCodeInput{
			UserID:       userID,
			RecoveryCode: "unique123",
		})
		if err != nil {
			t.Errorf("VerifyRecoveryCode() first call error = %v, want nil", err)
		}

		// 2回目: 失敗（既に消費済み）
		err = uc.VerifyRecoveryCode(context.Background(), VerifyRecoveryCodeInput{
			UserID:       userID,
			RecoveryCode: "unique123",
		})
		if !errors.Is(err, ErrInvalidRecoveryCode) {
			t.Errorf("VerifyRecoveryCode() second call error = %v, want %v", err, ErrInvalidRecoveryCode)
		}
	})

	t.Run("無効なリカバリーコードで検証に失敗する", func(t *testing.T) {
		_, tx := testutil.SetupTestDB(t)
		q := testutil.QueriesWithTx(tx)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
		uc := NewVerifyTwoFactorUsecase(userTwoFactorAuthRepo)

		secret := generateTestSecret(t)
		recoveryCodes := []string{"valid123", "valid456"}
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("recovery-invalid@example.com").
			WithAtname("recoveryinvalid").
			BuildWithTwoFactorAuthAndRecoveryCodes(secret, true, recoveryCodes)

		err := uc.VerifyRecoveryCode(context.Background(), VerifyRecoveryCodeInput{
			UserID:       userID,
			RecoveryCode: "invalid1", // 無効なコード
		})
		if !errors.Is(err, ErrInvalidRecoveryCode) {
			t.Errorf("VerifyRecoveryCode() error = %v, want %v", err, ErrInvalidRecoveryCode)
		}
	})

	t.Run("2FAが有効でないユーザーの場合はエラーを返す", func(t *testing.T) {
		_, tx := testutil.SetupTestDB(t)
		q := testutil.QueriesWithTx(tx)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
		uc := NewVerifyTwoFactorUsecase(userTwoFactorAuthRepo)

		secret := generateTestSecret(t)
		recoveryCodes := []string{"code1234"}
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("recovery-disabled@example.com").
			WithAtname("recoverydisabled").
			BuildWithTwoFactorAuthAndRecoveryCodes(secret, false, recoveryCodes) // 2FAが無効

		err := uc.VerifyRecoveryCode(context.Background(), VerifyRecoveryCodeInput{
			UserID:       userID,
			RecoveryCode: "code1234",
		})
		if !errors.Is(err, ErrTwoFactorNotEnabled) {
			t.Errorf("VerifyRecoveryCode() error = %v, want %v", err, ErrTwoFactorNotEnabled)
		}
	})

	t.Run("2FAが設定されていないユーザーの場合はエラーを返す", func(t *testing.T) {
		_, tx := testutil.SetupTestDB(t)
		q := testutil.QueriesWithTx(tx)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
		uc := NewVerifyTwoFactorUsecase(userTwoFactorAuthRepo)

		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("recovery-nosetup@example.com").
			WithAtname("recoverynosetup").
			Build() // 2FA設定なし

		err := uc.VerifyRecoveryCode(context.Background(), VerifyRecoveryCodeInput{
			UserID:       userID,
			RecoveryCode: "anycode1",
		})
		if !errors.Is(err, ErrTwoFactorNotEnabled) {
			t.Errorf("VerifyRecoveryCode() error = %v, want %v", err, ErrTwoFactorNotEnabled)
		}
	})
}
