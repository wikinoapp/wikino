package sign_in_two_factor_recovery_test

import (
	"context"
	"errors"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/handler/sign_in_two_factor_recovery"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestCreateValidator_Validate_FormatValidation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		recoveryCode string
		wantError    bool
		errorField   string
	}{
		{
			name:         "正常なリカバリーコード",
			recoveryCode: "code1234",
			wantError:    false,
		},
		{
			name:         "正常なリカバリーコード（全て数字）",
			recoveryCode: "12345678",
			wantError:    false,
		},
		{
			name:         "正常なリカバリーコード（全て小文字）",
			recoveryCode: "abcdefgh",
			wantError:    false,
		},
		{
			name:         "空のリカバリーコード",
			recoveryCode: "",
			wantError:    true,
			errorField:   "recovery_code",
		},
		{
			name:         "7文字（短すぎる）",
			recoveryCode: "code123",
			wantError:    true,
			errorField:   "recovery_code",
		},
		{
			name:         "9文字（長すぎる）",
			recoveryCode: "code12345",
			wantError:    true,
			errorField:   "recovery_code",
		},
		{
			name:         "大文字を含む",
			recoveryCode: "CODE1234",
			wantError:    true,
			errorField:   "recovery_code",
		},
		{
			name:         "一部大文字",
			recoveryCode: "Code1234",
			wantError:    true,
			errorField:   "recovery_code",
		},
		{
			name:         "記号を含む",
			recoveryCode: "code123!",
			wantError:    true,
			errorField:   "recovery_code",
		},
		{
			name:         "空白を含む",
			recoveryCode: "code 123",
			wantError:    true,
			errorField:   "recovery_code",
		},
		{
			name:         "ハイフンを含む",
			recoveryCode: "code-123",
			wantError:    true,
			errorField:   "recovery_code",
		},
		{
			name:         "アンダースコアを含む",
			recoveryCode: "code_123",
			wantError:    true,
			errorField:   "recovery_code",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTestDB(t)
			q := testutil.QueriesWithTx(tx)
			userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
			validator := sign_in_two_factor_recovery.NewCreateValidator(userTwoFactorAuthRepo)

			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, "ja")

			result := validator.Validate(ctx, sign_in_two_factor_recovery.CreateValidatorInput{
				UserID:       "dummy-user-id",
				RecoveryCode: tc.recoveryCode,
			})

			if tc.wantError {
				if result.FormErrors == nil {
					t.Errorf("expected error but got nil")
					return
				}
				if !result.FormErrors.HasFieldError(tc.errorField) {
					t.Errorf("expected error for field %s but not found", tc.errorField)
				}
			}
		})
	}
}

func TestCreateValidator_Validate_StateValidation(t *testing.T) {
	t.Parallel()

	t.Run("有効なリカバリーコードで検証に成功する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTestDB(t)
		q := testutil.QueriesWithTx(tx)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
		validator := sign_in_two_factor_recovery.NewCreateValidator(userTwoFactorAuthRepo)

		secret := "JBSWY3DPEHPK3PXP"
		recoveryCodes := []string{"code1234", "code5678", "abcd1234"}
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("recovery-valid@example.com").
			WithAtname("recoveryvalid").
			BuildWithTwoFactorAuthAndRecoveryCodes(secret, true, recoveryCodes)

		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, "ja")

		result := validator.Validate(ctx, sign_in_two_factor_recovery.CreateValidatorInput{
			UserID:       userID,
			RecoveryCode: "code1234",
		})

		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.FormErrors != nil {
			t.Errorf("unexpected form errors: %v", result.FormErrors)
		}
		if result.TwoFactorAuth == nil {
			t.Error("expected TwoFactorAuth to be set")
		}
	})

	t.Run("無効なリカバリーコードで検証に失敗する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTestDB(t)
		q := testutil.QueriesWithTx(tx)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
		validator := sign_in_two_factor_recovery.NewCreateValidator(userTwoFactorAuthRepo)

		secret := "JBSWY3DPEHPK3PXP"
		recoveryCodes := []string{"code1234", "code5678"}
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("recovery-invalid@example.com").
			WithAtname("recoveryinvalid").
			BuildWithTwoFactorAuthAndRecoveryCodes(secret, true, recoveryCodes)

		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, "ja")

		result := validator.Validate(ctx, sign_in_two_factor_recovery.CreateValidatorInput{
			UserID:       userID,
			RecoveryCode: "wrongcod", // 無効なコード
		})

		if !errors.Is(result.Err, sign_in_two_factor_recovery.ErrInvalidRecoveryCode) {
			t.Errorf("expected ErrInvalidRecoveryCode, got %v", result.Err)
		}
		if result.FormErrors == nil || !result.FormErrors.HasErrors() {
			t.Error("expected form errors")
		}
	})

	t.Run("2FAが有効でないユーザーの場合はエラーを返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTestDB(t)
		q := testutil.QueriesWithTx(tx)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
		validator := sign_in_two_factor_recovery.NewCreateValidator(userTwoFactorAuthRepo)

		secret := "JBSWY3DPEHPK3PXP"
		recoveryCodes := []string{"code1234"}
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("recovery-disabled@example.com").
			WithAtname("recoverydisabled").
			BuildWithTwoFactorAuthAndRecoveryCodes(secret, false, recoveryCodes) // 2FAが無効

		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, "ja")

		result := validator.Validate(ctx, sign_in_two_factor_recovery.CreateValidatorInput{
			UserID:       userID,
			RecoveryCode: "code1234",
		})

		if !errors.Is(result.Err, sign_in_two_factor_recovery.ErrTwoFactorNotEnabled) {
			t.Errorf("expected ErrTwoFactorNotEnabled, got %v", result.Err)
		}
		// 2FAが無効の場合はFormErrorsは設定されない（リダイレクト処理のため）
		if result.FormErrors != nil {
			t.Errorf("unexpected form errors: %v", result.FormErrors)
		}
	})

	t.Run("2FAが設定されていないユーザーの場合はエラーを返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTestDB(t)
		q := testutil.QueriesWithTx(tx)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
		validator := sign_in_two_factor_recovery.NewCreateValidator(userTwoFactorAuthRepo)

		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("recovery-nosetup@example.com").
			WithAtname("recoverynosetup").
			Build() // 2FA設定なし

		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, "ja")

		result := validator.Validate(ctx, sign_in_two_factor_recovery.CreateValidatorInput{
			UserID:       userID,
			RecoveryCode: "anycode1",
		})

		if !errors.Is(result.Err, sign_in_two_factor_recovery.ErrTwoFactorNotEnabled) {
			t.Errorf("expected ErrTwoFactorNotEnabled, got %v", result.Err)
		}
	})
}
