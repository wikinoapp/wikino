package validator_test

import (
	"context"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
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

func TestSignInTwoFactorCreateValidator_Validate_FormatValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		totpCode       string
		wantErrors     bool
		wantFieldError string
	}{
		{
			name:       "正常な6桁コード",
			totpCode:   "123456",
			wantErrors: false,
		},
		{
			name:           "TOTPコードが空",
			totpCode:       "",
			wantErrors:     true,
			wantFieldError: "totp_code",
		},
		{
			name:           "5桁のコード",
			totpCode:       "12345",
			wantErrors:     true,
			wantFieldError: "totp_code",
		},
		{
			name:           "7桁のコード",
			totpCode:       "1234567",
			wantErrors:     true,
			wantFieldError: "totp_code",
		},
		{
			name:           "英字を含むコード",
			totpCode:       "12345a",
			wantErrors:     true,
			wantFieldError: "totp_code",
		},
		{
			name:           "スペースを含むコード",
			totpCode:       "123 456",
			wantErrors:     true,
			wantFieldError: "totp_code",
		},
		{
			name:       "全て0のコード",
			totpCode:   "000000",
			wantErrors: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
			v := validator.NewSignInTwoFactorCreateValidator(userTwoFactorAuthRepo)

			// ダミーのユーザーIDを使用（形式バリデーションのテストなのでDB検証には到達しない）
			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, "ja")

			err := v.Validate(ctx, validator.SignInTwoFactorCreateValidatorInput{
				UserID:   "dummy-user-id",
				TOTPCode: tt.totpCode,
			})

			if tt.wantErrors {
				ve := model.AsValidationError(err)
				if ve == nil {
					t.Error("expected validation error, but got nil or different error type")
				} else if tt.wantFieldError != "" && !ve.HasFieldError(tt.wantFieldError) {
					t.Errorf("expected field error for %s, but not found", tt.wantFieldError)
				}
			}
		})
	}
}

func TestSignInTwoFactorCreateValidator_Validate_StateValidation(t *testing.T) {
	t.Parallel()

	t.Run("有効なTOTPコードで検証に成功する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
		v := validator.NewSignInTwoFactorCreateValidator(userTwoFactorAuthRepo)

		secret := generateTestSecret(t)
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("totp-valid@example.com").
			WithAtname("totpvalid").
			BuildWithTwoFactorAuth(secret, true)

		code := generateValidTOTPCode(t, secret)

		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, "ja")

		err := v.Validate(ctx, validator.SignInTwoFactorCreateValidatorInput{
			UserID:   userID,
			TOTPCode: code,
		})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("無効なTOTPコードで検証に失敗する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
		v := validator.NewSignInTwoFactorCreateValidator(userTwoFactorAuthRepo)

		secret := generateTestSecret(t)
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("totp-invalid@example.com").
			WithAtname("totpinvalid").
			BuildWithTwoFactorAuth(secret, true)

		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, "ja")

		err := v.Validate(ctx, validator.SignInTwoFactorCreateValidatorInput{
			UserID:   userID,
			TOTPCode: "000000", // 無効なコード
		})

		ve := model.AsValidationError(err)
		if ve == nil {
			t.Errorf("expected ValidationError, got %v", err)
		} else if !ve.HasErrors() {
			t.Error("expected form errors")
		}
	})

	t.Run("2FAが有効でないユーザーの場合はAppErrorを返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
		v := validator.NewSignInTwoFactorCreateValidator(userTwoFactorAuthRepo)

		secret := generateTestSecret(t)
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("totp-disabled@example.com").
			WithAtname("totpdisabled").
			BuildWithTwoFactorAuth(secret, false) // 2FAが無効

		code := generateValidTOTPCode(t, secret)

		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, "ja")

		err := v.Validate(ctx, validator.SignInTwoFactorCreateValidatorInput{
			UserID:   userID,
			TOTPCode: code,
		})

		ae := model.AsAppError(err)
		if ae == nil {
			t.Errorf("expected AppError, got %v", err)
		} else if ae.Code != model.AppErrCodeTwoFactorNotEnabled {
			t.Errorf("expected AppErrCodeTwoFactorNotEnabled, got %d", ae.Code)
		}
	})

	t.Run("2FAが設定されていないユーザーの場合はAppErrorを返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
		v := validator.NewSignInTwoFactorCreateValidator(userTwoFactorAuthRepo)

		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("totp-nosetup@example.com").
			WithAtname("totpnosetup").
			Build() // 2FA設定なし

		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, "ja")

		err := v.Validate(ctx, validator.SignInTwoFactorCreateValidatorInput{
			UserID:   userID,
			TOTPCode: "123456",
		})

		ae := model.AsAppError(err)
		if ae == nil {
			t.Errorf("expected AppError, got %v", err)
		} else if ae.Code != model.AppErrCodeTwoFactorNotEnabled {
			t.Errorf("expected AppErrCodeTwoFactorNotEnabled, got %d", ae.Code)
		}
	})
}
