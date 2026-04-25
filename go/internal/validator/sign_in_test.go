package validator_test

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestSignInCreateValidator_Validate(t *testing.T) {
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
				v := validator.NewSignInCreateValidator(nil, nil, nil)
				output, err := v.Validate(ctx, validator.SignInCreateValidatorInput{
					Email:    tt.email,
					Password: tt.password,
				})

				if output != nil {
					t.Error("expected nil output for validation error")
				}
				ve := model.AsValidationError(err)
				if ve == nil {
					t.Fatal("expected ValidationError, got nil")
				}
				if tt.wantFieldError != "" && !ve.HasFieldError(tt.wantFieldError) {
					t.Errorf("expected field error for %s, but not found", tt.wantFieldError)
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

			password := "testpassword123"
			passwordDigest, err := auth.HashPassword(password)
			if err != nil {
				t.Fatalf("パスワードのハッシュ化に失敗: %v", err)
			}

			userID := testutil.NewUserBuilder(t, tx).
				WithEmail("signin-valid@example.com").
				WithAtname("signinvalid").
				BuildWithPassword(passwordDigest)

			userRepo := repository.NewUserRepository(queries)
			userPasswordRepo := repository.NewUserPasswordRepository(queries)
			userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(queries)
			v := validator.NewSignInCreateValidator(userRepo, userPasswordRepo, userTwoFactorAuthRepo)

			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, "ja")

			output, err := v.Validate(ctx, validator.SignInCreateValidatorInput{
				Email:    "signin-valid@example.com",
				Password: password,
			})

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if output == nil {
				t.Fatal("expected output, got nil")
			}
			if output.User == nil {
				t.Fatal("expected user, got nil")
			}
			if output.User.ID != userID {
				t.Errorf("wrong user ID: got %v want %v", output.User.ID, userID)
			}
		})

		t.Run("ユーザーが見つからない場合、バリデーションエラーを返す", func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			queries := testutil.QueriesWithTx(tx)

			userRepo := repository.NewUserRepository(queries)
			userPasswordRepo := repository.NewUserPasswordRepository(queries)
			userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(queries)
			v := validator.NewSignInCreateValidator(userRepo, userPasswordRepo, userTwoFactorAuthRepo)

			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, "ja")

			output, err := v.Validate(ctx, validator.SignInCreateValidatorInput{
				Email:    "nonexistent@example.com",
				Password: "password123",
			})

			if output != nil {
				t.Error("expected nil output")
			}
			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatal("expected ValidationError, got nil")
			}
			if len(ve.Global) == 0 {
				t.Error("expected global error")
			}
		})

		t.Run("パスワードが正しくない場合、バリデーションエラーを返す", func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			queries := testutil.QueriesWithTx(tx)

			password := "testpassword123"
			passwordDigest, err := auth.HashPassword(password)
			if err != nil {
				t.Fatalf("パスワードのハッシュ化に失敗: %v", err)
			}

			_ = testutil.NewUserBuilder(t, tx).
				WithEmail("signin-wrong@example.com").
				WithAtname("signinwrong").
				BuildWithPassword(passwordDigest)

			userRepo := repository.NewUserRepository(queries)
			userPasswordRepo := repository.NewUserPasswordRepository(queries)
			userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(queries)
			v := validator.NewSignInCreateValidator(userRepo, userPasswordRepo, userTwoFactorAuthRepo)

			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, "ja")

			output, err := v.Validate(ctx, validator.SignInCreateValidatorInput{
				Email:    "signin-wrong@example.com",
				Password: "wrongpassword",
			})

			if output != nil {
				t.Error("expected nil output")
			}
			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatal("expected ValidationError, got nil")
			}
			if len(ve.Global) == 0 {
				t.Error("expected global error")
			}
		})

		t.Run("パスワードが設定されていない場合、バリデーションエラーを返す", func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			queries := testutil.QueriesWithTx(tx)

			_ = testutil.NewUserBuilder(t, tx).
				WithEmail("signin-nopassword@example.com").
				WithAtname("signinnopass").
				Build()

			userRepo := repository.NewUserRepository(queries)
			userPasswordRepo := repository.NewUserPasswordRepository(queries)
			userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(queries)
			v := validator.NewSignInCreateValidator(userRepo, userPasswordRepo, userTwoFactorAuthRepo)

			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, "ja")

			output, err := v.Validate(ctx, validator.SignInCreateValidatorInput{
				Email:    "signin-nopassword@example.com",
				Password: "password123",
			})

			if output != nil {
				t.Error("expected nil output")
			}
			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatal("expected ValidationError, got nil")
			}
			if len(ve.Global) == 0 {
				t.Error("expected global error")
			}
		})
	})
}
