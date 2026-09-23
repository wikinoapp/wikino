package validator_test

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestSignInTwoFactorRecoveryCreateValidator_Validate_FormatValidation(t *testing.T) {
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
			name:         "正常なリカバリーコード (全て数字)",
			recoveryCode: "12345678",
			wantError:    false,
		},
		{
			name:         "正常なリカバリーコード (全て小文字)",
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
			name:         "7文字 (短すぎる)",
			recoveryCode: "code123",
			wantError:    true,
			errorField:   "recovery_code",
		},
		{
			name:         "9文字 (長すぎる)",
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

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
			v := validator.NewSignInTwoFactorRecoveryCreateValidator(userTwoFactorAuthRepo)

			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, "ja")

			_, err := v.Validate(ctx, validator.SignInTwoFactorRecoveryCreateValidatorInput{
				UserID:       "dummy-user-id",
				RecoveryCode: tc.recoveryCode,
			})

			if tc.wantError {
				ve := model.AsValidationError(err)
				if ve == nil {
					t.Errorf("エラー = %v、期待値 = バリデーションエラー", err)
					return
				}
				if !ve.HasFieldError(tc.errorField) {
					t.Errorf("フィールド%sのエラーが見つからない", tc.errorField)
				}
			}
		})
	}
}

func TestSignInTwoFactorRecoveryCreateValidator_Validate_StateValidation(t *testing.T) {
	t.Parallel()

	t.Run("有効なリカバリーコードで検証に成功する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
		v := validator.NewSignInTwoFactorRecoveryCreateValidator(userTwoFactorAuthRepo)

		secret := "JBSWY3DPEHPK3PXP"
		recoveryCodes := []string{"code1234", "code5678", "abcd1234"}
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("val-recovery-valid@example.com").
			WithAtname("val-recoveryvalid").
			BuildWithTwoFactorAuthAndRecoveryCodes(secret, true, recoveryCodes)

		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, "ja")

		twoFactorAuth, err := v.Validate(ctx, validator.SignInTwoFactorRecoveryCreateValidatorInput{
			UserID:       userID,
			RecoveryCode: "code1234",
		})

		if err != nil {
			t.Errorf("予期しないエラー: %v", err)
		}
		if twoFactorAuth == nil {
			t.Error("TwoFactorAuthが設定されていない")
		}
	})

	t.Run("無効なリカバリーコードで検証に失敗する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
		v := validator.NewSignInTwoFactorRecoveryCreateValidator(userTwoFactorAuthRepo)

		secret := "JBSWY3DPEHPK3PXP"
		recoveryCodes := []string{"code1234", "code5678"}
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("val-recovery-invalid@example.com").
			WithAtname("val-recoveryinvalid").
			BuildWithTwoFactorAuthAndRecoveryCodes(secret, true, recoveryCodes)

		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, "ja")

		_, err := v.Validate(ctx, validator.SignInTwoFactorRecoveryCreateValidatorInput{
			UserID:       userID,
			RecoveryCode: "wrongcod", // 無効なコード
		})

		ve := model.AsValidationError(err)
		if ve == nil {
			t.Errorf("ValidationErrorを期待したが、%vだった", err)
		} else if !ve.HasErrors() {
			t.Error("フォームのエラーが無い")
		}
	})

	t.Run("2FAが有効でないユーザーの場合はAppErrorを返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
		v := validator.NewSignInTwoFactorRecoveryCreateValidator(userTwoFactorAuthRepo)

		secret := "JBSWY3DPEHPK3PXP"
		recoveryCodes := []string{"code1234"}
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("val-recovery-disabled@example.com").
			WithAtname("val-recoverydisabled").
			BuildWithTwoFactorAuthAndRecoveryCodes(secret, false, recoveryCodes) // 2FAが無効

		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, "ja")

		_, err := v.Validate(ctx, validator.SignInTwoFactorRecoveryCreateValidatorInput{
			UserID:       userID,
			RecoveryCode: "code1234",
		})

		ae := model.AsAppError(err)
		if ae == nil {
			t.Errorf("AppErrorを期待したが、%vだった", err)
		} else if ae.Code != model.AppErrCodeTwoFactorNotEnabled {
			t.Errorf("AppErrCodeTwoFactorNotEnabledを期待したが、%dだった", ae.Code)
		}
	})

	t.Run("2FAが設定されていないユーザーの場合はAppErrorを返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
		v := validator.NewSignInTwoFactorRecoveryCreateValidator(userTwoFactorAuthRepo)

		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("val-recovery-nosetup@example.com").
			WithAtname("val-recoverynosetup").
			Build() // 2FA設定なし

		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, "ja")

		_, err := v.Validate(ctx, validator.SignInTwoFactorRecoveryCreateValidatorInput{
			UserID:       userID,
			RecoveryCode: "anycode1",
		})

		ae := model.AsAppError(err)
		if ae == nil {
			t.Errorf("AppErrorを期待したが、%vだった", err)
		} else if ae.Code != model.AppErrCodeTwoFactorNotEnabled {
			t.Errorf("AppErrCodeTwoFactorNotEnabledを期待したが、%dだった", ae.Code)
		}
	})
}
