package validator_test

import (
	"context"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

const settingsTwoFactorAuthSecret = "JBSWY3DPEHPK3PXPJBSWY3DPEHPK3PXP"

func TestSettingsTwoFactorAuthCreateValidator_Validate(t *testing.T) {
	t.Parallel()

	code, err := totp.GenerateCode(settingsTwoFactorAuthSecret, time.Now())
	if err != nil {
		t.Fatalf("コードの生成に失敗: %v", err)
	}
	// 現在のコードだけでなく、許容する前後のタイムステップでも通らない値を使う。
	wrongCode := ""
	for _, candidate := range []string{"000000", "111111", "222222", "333333", "444444"} {
		if !auth.ValidateTOTPCode(candidate, settingsTwoFactorAuthSecret, time.Now()) {
			wrongCode = candidate
			break
		}
	}
	if wrongCode == "" {
		t.Fatal("誤りのコードを用意できなかった")
	}

	testCases := []struct {
		name       string
		password   string
		totpCode   string
		wantFields []string
	}{
		{name: "正しいパスワードとコード", password: settingsEmailPassword, totpCode: code},
		{name: "パスワードが空", password: "", totpCode: code, wantFields: []string{"password"}},
		{name: "コードが空", password: settingsEmailPassword, totpCode: "", wantFields: []string{"totp_code"}},
		{name: "コードが6桁の数字でない", password: settingsEmailPassword, totpCode: "12ab56", wantFields: []string{"totp_code"}},
		{name: "パスワードが違う", password: "wrong-password", totpCode: code, wantFields: []string{"password"}},
		{name: "コードが違う", password: settingsEmailPassword, totpCode: wrongCode, wantFields: []string{"totp_code"}},
		{name: "パスワードとコードの両方が違う", password: "wrong-password", totpCode: wrongCode, wantFields: []string{"password", "totp_code"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			userID := settingsEmailUser(t, tx, "settings_2fa_create")

			v := validator.NewSettingsTwoFactorAuthCreateValidator(repository.NewUserPasswordRepository(q))
			ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
			err := v.Validate(ctx, validator.SettingsTwoFactorAuthCreateValidatorInput{
				UserID:   userID,
				Password: tc.password,
				TOTPCode: tc.totpCode,
				Secret:   settingsTwoFactorAuthSecret,
			})

			if len(tc.wantFields) == 0 {
				if err != nil {
					t.Fatalf("Validate()のエラー = %v", err)
				}
				return
			}
			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatalf("エラー = %v、ValidationErrorを期待", err)
			}
			for _, field := range []string{"password", "totp_code"} {
				want := false
				for _, f := range tc.wantFields {
					want = want || f == field
				}
				if got := ve.HasFieldError(field); got != want {
					t.Errorf("%qのフィールドエラーがあるか = %v、期待値 = %v (%v)", field, got, want, ve.Fields)
				}
			}
		})
	}
}

func TestSettingsTwoFactorAuthDeleteValidator_Validate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		password  string
		wantError bool
	}{
		{name: "正しいパスワード", password: settingsEmailPassword},
		{name: "パスワードが空", password: "", wantError: true},
		{name: "パスワードが違う", password: "wrong-password", wantError: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			userID := settingsEmailUser(t, tx, "settings_2fa_delete")

			v := validator.NewSettingsTwoFactorAuthDeleteValidator(repository.NewUserPasswordRepository(testutil.QueriesWithTx(tx)))
			ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
			err := v.Validate(ctx, validator.SettingsTwoFactorAuthDeleteValidatorInput{UserID: userID, Password: tc.password})

			assertPasswordValidation(t, err, tc.wantError)
		})
	}
}

func TestSettingsTwoFactorAuthRecoveryCodeCreateValidator_Validate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		password  string
		wantError bool
	}{
		{name: "正しいパスワード", password: settingsEmailPassword},
		{name: "パスワードが空", password: "", wantError: true},
		{name: "パスワードが違う", password: "wrong-password", wantError: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			userID := settingsEmailUser(t, tx, "settings_2fa_regenerate")

			v := validator.NewSettingsTwoFactorAuthRecoveryCodeCreateValidator(repository.NewUserPasswordRepository(testutil.QueriesWithTx(tx)))
			ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
			err := v.Validate(ctx, validator.SettingsTwoFactorAuthRecoveryCodeCreateValidatorInput{UserID: userID, Password: tc.password})

			assertPasswordValidation(t, err, tc.wantError)
		})
	}
}

// assertPasswordValidationは、wantErrorならpasswordの項目のValidationErrorを、そうでなければnilを期待する
func assertPasswordValidation(t *testing.T, err error, wantError bool) {
	t.Helper()

	if !wantError {
		if err != nil {
			t.Fatalf("Validate()のエラー = %v", err)
		}
		return
	}
	ve := model.AsValidationError(err)
	if ve == nil || !ve.HasFieldError("password") {
		t.Fatalf("エラー = %v、passwordのValidationErrorを期待", err)
	}
}
