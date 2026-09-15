package validator_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/password_reset"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// 128文字を超えるパスワードのテスト用文字列
var longPassword = strings.Repeat("a", 129)

func TestPasswordUpdateValidator_Validate_FormValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          validator.PasswordUpdateValidatorInput
		wantFieldError string
		wantGlobal     bool
	}{
		{
			name: "パスワードが空",
			input: validator.PasswordUpdateValidatorInput{
				Token:                "valid-token",
				Password:             "",
				PasswordConfirmation: "password123",
			},
			wantFieldError: "password",
		},
		{
			name: "パスワード確認が空",
			input: validator.PasswordUpdateValidatorInput{
				Token:                "valid-token",
				Password:             "password123",
				PasswordConfirmation: "",
			},
			wantFieldError: "password_confirmation",
		},
		{
			name: "パスワードが短すぎる",
			input: validator.PasswordUpdateValidatorInput{
				Token:                "valid-token",
				Password:             "pass",
				PasswordConfirmation: "pass",
			},
			wantFieldError: "password",
		},
		{
			name: "パスワードが長すぎる",
			input: validator.PasswordUpdateValidatorInput{
				Token:                "valid-token",
				Password:             longPassword,
				PasswordConfirmation: longPassword,
			},
			wantFieldError: "password",
		},
		{
			name: "パスワードに無効な文字が含まれる",
			input: validator.PasswordUpdateValidatorInput{
				Token:                "valid-token",
				Password:             "パスワード123",
				PasswordConfirmation: "パスワード123",
			},
			wantFieldError: "password",
		},
		{
			name: "パスワードが一致しない",
			input: validator.PasswordUpdateValidatorInput{
				Token:                "valid-token",
				Password:             "password123",
				PasswordConfirmation: "different456",
			},
			wantFieldError: "password_confirmation",
		},
		{
			name: "トークンが空",
			input: validator.PasswordUpdateValidatorInput{
				Token:                "",
				Password:             "password123",
				PasswordConfirmation: "password123",
			},
			wantGlobal: true,
		},
	}

	// DBアクセスなしでテスト (形式バリデーションのみ)
	v := validator.NewPasswordUpdateValidator(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, "ja")

			output, err := v.Validate(ctx, tt.input)

			if output != nil {
				t.Error("バリデーションエラーなのに出力がnilではない")
			}

			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatal("ValidationErrorを期待したが、nilだった")
			}
			if tt.wantFieldError != "" && !ve.HasFieldError(tt.wantFieldError) {
				t.Errorf("%sのフィールドエラーが見つからない", tt.wantFieldError)
			}
			if tt.wantGlobal && len(ve.Global) == 0 {
				t.Error("グローバルエラーが見つからない")
			}
		})
	}
}

func TestPasswordUpdateValidator_Validate_TokenValidation(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// テストユーザーを作成
	userID := testutil.NewUserBuilder(t, tx).Build()

	// 有効なトークンを作成
	validToken := "valid-test-token-12345"
	validTokenDigest := password_reset.HashToken(validToken)
	testutil.NewPasswordResetTokenBuilder(t, tx).
		WithUserID(userID).
		WithTokenDigest(validTokenDigest).
		WithExpiresAt(time.Now().Add(1 * time.Hour)).
		Build()

	// 使用済みトークンを作成
	usedToken := "used-test-token-12345"
	usedTokenDigest := password_reset.HashToken(usedToken)
	usedAt := time.Now().Add(-30 * time.Minute)
	testutil.NewPasswordResetTokenBuilder(t, tx).
		WithUserID(userID).
		WithTokenDigest(usedTokenDigest).
		WithExpiresAt(time.Now().Add(1 * time.Hour)).
		WithUsedAt(usedAt).
		Build()

	// 期限切れトークンを作成
	expiredToken := "expired-test-token-12345"
	expiredTokenDigest := password_reset.HashToken(expiredToken)
	testutil.NewPasswordResetTokenBuilder(t, tx).
		WithUserID(userID).
		WithTokenDigest(expiredTokenDigest).
		WithExpiresAt(time.Now().Add(-1 * time.Hour)).
		Build()

	passwordResetTokenRepo := repository.NewPasswordResetTokenRepository(queries)
	v := validator.NewPasswordUpdateValidator(passwordResetTokenRepo)

	t.Run("有効なトークン", func(t *testing.T) {
		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, "ja")

		output, err := v.Validate(ctx, validator.PasswordUpdateValidatorInput{
			Token:                validToken,
			Password:             "newpassword123",
			PasswordConfirmation: "newpassword123",
		})

		if err != nil {
			t.Errorf("予期しないエラー: %v", err)
		}
		if output == nil {
			t.Fatal("出力がnil")
		}
		if output.TokenID == "" {
			t.Error("TokenIDが空")
		}
		if output.UserID == "" {
			t.Error("UserIDが空")
		}
	})

	t.Run("存在しないトークン", func(t *testing.T) {
		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, "ja")

		output, err := v.Validate(ctx, validator.PasswordUpdateValidatorInput{
			Token:                "non-existent-token",
			Password:             "newpassword123",
			PasswordConfirmation: "newpassword123",
		})

		if output != nil {
			t.Error("出力がnilではない")
		}
		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatal("ValidationErrorを期待したが、nilだった")
		}
		if len(ve.Global) == 0 {
			t.Error("無効なトークンなのにグローバルエラーが無い")
		}
	})

	t.Run("使用済みトークン", func(t *testing.T) {
		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, "ja")

		output, err := v.Validate(ctx, validator.PasswordUpdateValidatorInput{
			Token:                usedToken,
			Password:             "newpassword123",
			PasswordConfirmation: "newpassword123",
		})

		if output != nil {
			t.Error("出力がnilではない")
		}
		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatal("ValidationErrorを期待したが、nilだった")
		}
		if len(ve.Global) == 0 {
			t.Error("使用済みのトークンなのにグローバルエラーが無い")
		}
	})

	t.Run("期限切れトークン", func(t *testing.T) {
		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, "ja")

		output, err := v.Validate(ctx, validator.PasswordUpdateValidatorInput{
			Token:                expiredToken,
			Password:             "newpassword123",
			PasswordConfirmation: "newpassword123",
		})

		if output != nil {
			t.Error("出力がnilではない")
		}
		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatal("ValidationErrorを期待したが、nilだった")
		}
		if len(ve.Global) == 0 {
			t.Error("期限切れのトークンなのにグローバルエラーが無い")
		}
	})
}

func TestPasswordUpdateValidator_Validate_I18nMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		locale   string
		input    validator.PasswordUpdateValidatorInput
		wantText string
	}{
		{
			name:   "日本語: パスワード必須エラー",
			locale: "ja",
			input: validator.PasswordUpdateValidatorInput{
				Token:                "valid-token",
				Password:             "",
				PasswordConfirmation: "password123",
			},
			wantText: "パスワードを入力してください",
		},
		{
			name:   "英語: パスワード必須エラー",
			locale: "en",
			input: validator.PasswordUpdateValidatorInput{
				Token:                "valid-token",
				Password:             "",
				PasswordConfirmation: "password123",
			},
			wantText: "Please enter a password",
		},
		{
			name:   "日本語: パスワード不一致エラー",
			locale: "ja",
			input: validator.PasswordUpdateValidatorInput{
				Token:                "valid-token",
				Password:             "password123",
				PasswordConfirmation: "different456",
			},
			wantText: "パスワードが一致しません",
		},
		{
			name:   "英語: パスワード不一致エラー",
			locale: "en",
			input: validator.PasswordUpdateValidatorInput{
				Token:                "valid-token",
				Password:             "password123",
				PasswordConfirmation: "different456",
			},
			wantText: "Passwords do not match",
		},
		{
			name:   "日本語: パスワード長すぎエラー",
			locale: "ja",
			input: validator.PasswordUpdateValidatorInput{
				Token:                "valid-token",
				Password:             longPassword,
				PasswordConfirmation: longPassword,
			},
			wantText: "パスワードは128文字以内で入力してください",
		},
		{
			name:   "英語: パスワード長すぎエラー",
			locale: "en",
			input: validator.PasswordUpdateValidatorInput{
				Token:                "valid-token",
				Password:             longPassword,
				PasswordConfirmation: longPassword,
			},
			wantText: "Password must be at most 128 characters",
		},
		{
			name:   "日本語: パスワード無効文字エラー",
			locale: "ja",
			input: validator.PasswordUpdateValidatorInput{
				Token:                "valid-token",
				Password:             "パスワード123",
				PasswordConfirmation: "パスワード123",
			},
			wantText: "パスワードには印字可能なASCII文字のみ使用できます",
		},
		{
			name:   "英語: パスワード無効文字エラー",
			locale: "en",
			input: validator.PasswordUpdateValidatorInput{
				Token:                "valid-token",
				Password:             "パスワード123",
				PasswordConfirmation: "パスワード123",
			},
			wantText: "Password can only contain printable ASCII characters",
		},
	}

	v := validator.NewPasswordUpdateValidator(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, tt.locale)

			_, err := v.Validate(ctx, tt.input)

			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatal("ValidationErrorを期待したが、nilだった")
			}

			// エラーメッセージに期待するテキストが含まれているか確認
			found := false

			// フィールドエラーをチェック
			for _, errs := range ve.Fields {
				for _, e := range errs {
					if strings.Contains(e, tt.wantText) {
						found = true
						break
					}
				}
			}

			// グローバルエラーをチェック
			for _, e := range ve.Global {
				if strings.Contains(e, tt.wantText) {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("%qを含むエラーメッセージが無い: fields = %v、global = %v", tt.wantText, ve.Fields, ve.Global)
			}
		})
	}
}
