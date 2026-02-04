package password_test

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/handler/password"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/password_reset"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestUpdateValidator_Validate_FormValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          password.UpdateValidatorInput
		wantErrors     bool
		wantFieldError string
		wantGlobal     bool
	}{
		{
			name: "パスワードが空",
			input: password.UpdateValidatorInput{
				Token:                "valid-token",
				Password:             "",
				PasswordConfirmation: "password123",
			},
			wantErrors:     true,
			wantFieldError: "password",
		},
		{
			name: "パスワード確認が空",
			input: password.UpdateValidatorInput{
				Token:                "valid-token",
				Password:             "password123",
				PasswordConfirmation: "",
			},
			wantErrors:     true,
			wantFieldError: "password_confirmation",
		},
		{
			name: "パスワードが短すぎる",
			input: password.UpdateValidatorInput{
				Token:                "valid-token",
				Password:             "pass",
				PasswordConfirmation: "pass",
			},
			wantErrors:     true,
			wantFieldError: "password",
		},
		{
			name: "パスワードが一致しない",
			input: password.UpdateValidatorInput{
				Token:                "valid-token",
				Password:             "password123",
				PasswordConfirmation: "different456",
			},
			wantErrors:     true,
			wantFieldError: "password_confirmation",
		},
		{
			name: "トークンが空",
			input: password.UpdateValidatorInput{
				Token:                "",
				Password:             "password123",
				PasswordConfirmation: "password123",
			},
			wantErrors: true,
			wantGlobal: true,
		},
	}

	// DBアクセスなしでテスト（形式バリデーションのみ）
	validator := password.NewUpdateValidator(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, "ja")

			result := validator.Validate(ctx, tt.input)

			if tt.wantErrors {
				if result.FormErrors == nil || !result.FormErrors.HasErrors() {
					t.Error("expected errors, but got none")
				}
				if tt.wantFieldError != "" && !result.FormErrors.HasFieldError(tt.wantFieldError) {
					t.Errorf("expected field error for %s, but not found", tt.wantFieldError)
				}
				if tt.wantGlobal && len(result.FormErrors.Global) == 0 {
					t.Error("expected global error, but not found")
				}
			}
		})
	}
}

func TestUpdateValidator_Validate_TokenValidation(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTestDB(t)
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
	validator := password.NewUpdateValidator(passwordResetTokenRepo)

	t.Run("有効なトークン", func(t *testing.T) {
		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, "ja")

		result := validator.Validate(ctx, password.UpdateValidatorInput{
			Token:                validToken,
			Password:             "newpassword123",
			PasswordConfirmation: "newpassword123",
		})

		if result.FormErrors != nil && result.FormErrors.HasErrors() {
			t.Errorf("unexpected errors: %v", result.FormErrors)
		}
		if result.TokenID == "" {
			t.Error("expected TokenID, but got empty")
		}
		if result.UserID == "" {
			t.Error("expected UserID, but got empty")
		}
	})

	t.Run("存在しないトークン", func(t *testing.T) {
		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, "ja")

		result := validator.Validate(ctx, password.UpdateValidatorInput{
			Token:                "non-existent-token",
			Password:             "newpassword123",
			PasswordConfirmation: "newpassword123",
		})

		if result.FormErrors == nil || !result.FormErrors.HasErrors() {
			t.Error("expected errors, but got none")
		}
		if len(result.FormErrors.Global) == 0 {
			t.Error("expected global error for invalid token")
		}
	})

	t.Run("使用済みトークン", func(t *testing.T) {
		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, "ja")

		result := validator.Validate(ctx, password.UpdateValidatorInput{
			Token:                usedToken,
			Password:             "newpassword123",
			PasswordConfirmation: "newpassword123",
		})

		if result.FormErrors == nil || !result.FormErrors.HasErrors() {
			t.Error("expected errors, but got none")
		}
		if len(result.FormErrors.Global) == 0 {
			t.Error("expected global error for used token")
		}
	})

	t.Run("期限切れトークン", func(t *testing.T) {
		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, "ja")

		result := validator.Validate(ctx, password.UpdateValidatorInput{
			Token:                expiredToken,
			Password:             "newpassword123",
			PasswordConfirmation: "newpassword123",
		})

		if result.FormErrors == nil || !result.FormErrors.HasErrors() {
			t.Error("expected errors, but got none")
		}
		if len(result.FormErrors.Global) == 0 {
			t.Error("expected global error for expired token")
		}
	})
}

func TestUpdateValidator_Validate_I18nMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		locale   string
		input    password.UpdateValidatorInput
		wantText string
	}{
		{
			name:   "日本語: パスワード必須エラー",
			locale: "ja",
			input: password.UpdateValidatorInput{
				Token:                "valid-token",
				Password:             "",
				PasswordConfirmation: "password123",
			},
			wantText: "パスワードを入力してください",
		},
		{
			name:   "英語: パスワード必須エラー",
			locale: "en",
			input: password.UpdateValidatorInput{
				Token:                "valid-token",
				Password:             "",
				PasswordConfirmation: "password123",
			},
			wantText: "Please enter a password",
		},
		{
			name:   "日本語: パスワード不一致エラー",
			locale: "ja",
			input: password.UpdateValidatorInput{
				Token:                "valid-token",
				Password:             "password123",
				PasswordConfirmation: "different456",
			},
			wantText: "パスワードが一致しません",
		},
		{
			name:   "英語: パスワード不一致エラー",
			locale: "en",
			input: password.UpdateValidatorInput{
				Token:                "valid-token",
				Password:             "password123",
				PasswordConfirmation: "different456",
			},
			wantText: "Passwords do not match",
		},
	}

	validator := password.NewUpdateValidator(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, tt.locale)

			result := validator.Validate(ctx, tt.input)

			if result.FormErrors == nil || !result.FormErrors.HasErrors() {
				t.Fatal("expected errors, but got none")
			}

			// エラーメッセージに期待するテキストが含まれているか確認
			found := false

			// フィールドエラーをチェック
			for _, errors := range result.FormErrors.Fields {
				for _, err := range errors {
					if containsSubstring(err, tt.wantText) {
						found = true
						break
					}
				}
			}

			// グローバルエラーをチェック
			for _, err := range result.FormErrors.Global {
				if containsSubstring(err, tt.wantText) {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("expected error message containing %q, got fields=%v, global=%v", tt.wantText, result.FormErrors.Fields, result.FormErrors.Global)
			}
		})
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstringHelper(s, substr))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
