package sign_in_test

import (
	"context"
	"errors"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/handler/sign_in"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestCreateValidator_Validate(t *testing.T) {
	t.Parallel()

	t.Run("形式バリデーション", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name           string
			email          string
			password       string
			wantFieldError string
		}{
			{
				name:           "メールアドレスが空",
				email:          "",
				password:       "password123",
				wantFieldError: "email",
			},
			{
				name:           "メールアドレスが無効な形式",
				email:          "invalid-email",
				password:       "password123",
				wantFieldError: "email",
			},
			{
				name:           "パスワードが空",
				email:          "test@example.com",
				password:       "",
				wantFieldError: "password",
			},
			{
				name:           "両方が空",
				email:          "",
				password:       "",
				wantFieldError: "email",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				ctx := context.Background()
				// DB不要なので nil を渡す（形式バリデーションのみ）
				validator := sign_in.NewCreateValidator(nil, nil)
				result := validator.Validate(ctx, sign_in.CreateValidatorInput{
					Email:    tt.email,
					Password: tt.password,
				})

				if result.FormErrors == nil {
					t.Error("expected errors, but got nil")
				}
				if result.FormErrors != nil && tt.wantFieldError != "" {
					if !result.FormErrors.HasFieldError(tt.wantFieldError) {
						t.Errorf("expected field error for %s, but not found", tt.wantFieldError)
					}
				}
			})
		}
	})

	t.Run("状態バリデーション", func(t *testing.T) {
		t.Parallel()

		t.Run("有効な認証情報の場合、ユーザーを返す", func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			queries := testutil.QueriesWithTx(tx)

			// テスト用パスワードをハッシュ化
			password := "testpassword123"
			passwordDigest, err := auth.HashPassword(password)
			if err != nil {
				t.Fatalf("パスワードのハッシュ化に失敗: %v", err)
			}

			// テストユーザーを作成
			userID := testutil.NewUserBuilder(t, tx).
				WithEmail("test@example.com").
				WithAtname("testuser").
				BuildWithPassword(passwordDigest)

			userRepo := repository.NewUserRepository(queries)
			userPasswordRepo := repository.NewUserPasswordRepository(queries)
			validator := sign_in.NewCreateValidator(userRepo, userPasswordRepo)

			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, "ja")

			result := validator.Validate(ctx, sign_in.CreateValidatorInput{
				Email:    "test@example.com",
				Password: password,
			})

			if result.Err != nil {
				t.Errorf("unexpected error: %v", result.Err)
			}
			if result.FormErrors != nil {
				t.Errorf("unexpected form errors: %v", result.FormErrors)
			}
			if result.User == nil {
				t.Error("expected user, got nil")
			}
			if result.User != nil && result.User.ID != userID {
				t.Errorf("wrong user ID: got %v want %v", result.User.ID, userID)
			}
		})

		t.Run("ユーザーが見つからない場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			queries := testutil.QueriesWithTx(tx)

			userRepo := repository.NewUserRepository(queries)
			userPasswordRepo := repository.NewUserPasswordRepository(queries)
			validator := sign_in.NewCreateValidator(userRepo, userPasswordRepo)

			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, "ja")

			result := validator.Validate(ctx, sign_in.CreateValidatorInput{
				Email:    "nonexistent@example.com",
				Password: "password123",
			})

			if result.User != nil {
				t.Error("expected nil user")
			}
			if result.FormErrors == nil || !result.FormErrors.HasErrors() {
				t.Error("expected form errors")
			}
			if !errors.Is(result.Err, sign_in.ErrUserNotFound) {
				t.Errorf("expected ErrUserNotFound, got %v", result.Err)
			}
		})

		t.Run("パスワードが正しくない場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			queries := testutil.QueriesWithTx(tx)

			// テスト用パスワードをハッシュ化
			password := "testpassword123"
			passwordDigest, err := auth.HashPassword(password)
			if err != nil {
				t.Fatalf("パスワードのハッシュ化に失敗: %v", err)
			}

			// テストユーザーを作成
			_ = testutil.NewUserBuilder(t, tx).
				WithEmail("test@example.com").
				WithAtname("testuser").
				BuildWithPassword(passwordDigest)

			userRepo := repository.NewUserRepository(queries)
			userPasswordRepo := repository.NewUserPasswordRepository(queries)
			validator := sign_in.NewCreateValidator(userRepo, userPasswordRepo)

			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, "ja")

			result := validator.Validate(ctx, sign_in.CreateValidatorInput{
				Email:    "test@example.com",
				Password: "wrongpassword",
			})

			if result.User != nil {
				t.Error("expected nil user")
			}
			if result.FormErrors == nil || !result.FormErrors.HasErrors() {
				t.Error("expected form errors")
			}
			if !errors.Is(result.Err, sign_in.ErrInvalidPassword) {
				t.Errorf("expected ErrInvalidPassword, got %v", result.Err)
			}
		})

		t.Run("パスワードが設定されていない場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			queries := testutil.QueriesWithTx(tx)

			// パスワードなしでテストユーザーを作成
			_ = testutil.NewUserBuilder(t, tx).
				WithEmail("nopassword@example.com").
				WithAtname("nopassworduser").
				Build()

			userRepo := repository.NewUserRepository(queries)
			userPasswordRepo := repository.NewUserPasswordRepository(queries)
			validator := sign_in.NewCreateValidator(userRepo, userPasswordRepo)

			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, "ja")

			result := validator.Validate(ctx, sign_in.CreateValidatorInput{
				Email:    "nopassword@example.com",
				Password: "password123",
			})

			if result.User != nil {
				t.Error("expected nil user")
			}
			if result.FormErrors == nil || !result.FormErrors.HasErrors() {
				t.Error("expected form errors")
			}
			if !errors.Is(result.Err, sign_in.ErrPasswordNotSet) {
				t.Errorf("expected ErrPasswordNotSet, got %v", result.Err)
			}
		})
	})
}
