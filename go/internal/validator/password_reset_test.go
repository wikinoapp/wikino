package validator_test

import (
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestPasswordResetCreateValidator_Validate(t *testing.T) {
	t.Parallel()

	t.Run("形式バリデーション", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name           string
			email          string
			wantErrors     bool
			wantFieldError string
		}{
			{
				name:       "正常系: 有効なメールアドレス",
				email:      "user@example.com",
				wantErrors: false,
			},
			{
				name:       "正常系: サブドメイン付きメールアドレス",
				email:      "user@mail.example.co.jp",
				wantErrors: false,
			},
			{
				name:       "正常系: +記号付きメールアドレス",
				email:      "user+tag@example.com",
				wantErrors: false,
			},
			{
				name:           "異常系: メールアドレスが空",
				email:          "",
				wantErrors:     true,
				wantFieldError: "email",
			},
			{
				name:           "異常系: メールアドレスの形式が不正（@なし）",
				email:          "invalid-email",
				wantErrors:     true,
				wantFieldError: "email",
			},
			{
				name:           "異常系: メールアドレスの形式が不正（ドメインなし）",
				email:          "user@",
				wantErrors:     true,
				wantFieldError: "email",
			},
			{
				name:           "異常系: メールアドレスの形式が不正（ユーザー名なし）",
				email:          "@example.com",
				wantErrors:     true,
				wantFieldError: "email",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				ctx := context.Background()
				ctx = i18n.SetLocale(ctx, "ja")

				v := validator.NewPasswordResetCreateValidator()
				err := v.Validate(ctx, validator.PasswordResetCreateValidatorInput{
					Email: tt.email,
				})

				if tt.wantErrors {
					ve := model.AsValidationError(err)
					if ve == nil {
						t.Error("expected ValidationError, but got nil")
						return
					}
					if tt.wantFieldError != "" && !ve.HasFieldError(tt.wantFieldError) {
						t.Errorf("expected field error for %s, but not found", tt.wantFieldError)
					}
				} else {
					if err != nil {
						t.Errorf("unexpected error: %v", err)
					}
				}
			})
		}
	})
}

func TestPasswordResetCreateValidator_Validate_I18nMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		locale   string
		email    string
		wantText string
	}{
		{
			name:     "日本語: メールアドレス必須エラー",
			locale:   "ja",
			email:    "",
			wantText: "入力してください",
		},
		{
			name:     "英語: メールアドレス必須エラー",
			locale:   "en",
			email:    "",
			wantText: "is required",
		},
		{
			name:     "日本語: メールアドレス形式エラー",
			locale:   "ja",
			email:    "invalid",
			wantText: "有効なメールアドレスを入力してください",
		},
		{
			name:     "英語: メールアドレス形式エラー",
			locale:   "en",
			email:    "invalid",
			wantText: "Please enter a valid email address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, tt.locale)

			v := validator.NewPasswordResetCreateValidator()
			err := v.Validate(ctx, validator.PasswordResetCreateValidatorInput{
				Email: tt.email,
			})

			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatal("expected ValidationError, but got nil")
			}

			// エラーメッセージに期待するテキストが含まれているか確認
			errors := ve.GetFieldErrors("email")
			if len(errors) == 0 {
				t.Fatal("expected email field error, but not found")
			}

			found := false
			for _, e := range errors {
				if strings.Contains(e, tt.wantText) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected error message containing %q, got %v", tt.wantText, errors)
			}
		})
	}
}
