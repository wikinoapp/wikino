package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/password_reset"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestUpdatePasswordResetUsecase_Execute(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)

	t.Run("正常系: パスワードが更新されトークンが使用済みになる", func(t *testing.T) {
		t.Parallel()

		// テストユーザーを作成
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("update-password-success@example.com").
			WithAtname("update_password_success").
			Build()

		// ユーザーのパスワードを作成
		testutil.NewUserPasswordBuilderDB(t, db).
			WithUserID(userID).
			WithPasswordDigest("$2a$10$oldpasswordhashhashhash").
			Build()

		// 有効なトークンを作成
		plainToken := "valid_test_token_for_update_success"
		tokenDigest := password_reset.HashToken(plainToken)
		testutil.NewPasswordResetTokenBuilderDB(t, db).
			WithUserID(userID).
			WithTokenDigest(tokenDigest).
			WithExpiresAt(time.Now().Add(1 * time.Hour)).
			Build()

		passwordResetTokenRepo := repository.NewPasswordResetTokenRepository(q)
		userPasswordRepo := repository.NewUserPasswordRepository(q)
		updateValidator := validator.NewPasswordUpdateValidator(passwordResetTokenRepo)
		uc := NewUpdatePasswordResetUsecase(db, passwordResetTokenRepo, userPasswordRepo, updateValidator)

		ctx := i18n.SetLocale(context.Background(), "ja")
		input := UpdatePasswordResetInput{
			Token:                plainToken,
			Password:             "newpassword123",
			PasswordConfirmation: "newpassword123",
		}

		output, err := uc.Execute(ctx, input)
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if output.UserID != userID {
			t.Errorf("UserID = %s, want %s", output.UserID, userID)
		}

		// パスワードが更新されていることを確認
		updatedPassword, err := userPasswordRepo.FindByUserID(ctx, userID)
		if err != nil {
			t.Fatalf("FindByUserID() error = %v", err)
		}
		if updatedPassword == nil {
			t.Fatal("パスワードが見つかりません")
		}

		// 新しいパスワードで認証できることを確認
		if !auth.VerifyPassword(updatedPassword.PasswordDigest, "newpassword123") {
			t.Error("新しいパスワードで認証できません")
		}

		// トークンが使用済みになっていることを確認
		usedToken, err := passwordResetTokenRepo.FindByTokenDigest(ctx, tokenDigest)
		if err != nil {
			t.Fatalf("FindByTokenDigest() error = %v", err)
		}
		if usedToken == nil {
			t.Fatal("トークンが見つかりません")
		}
		if !usedToken.IsUsed() {
			t.Error("トークンが使用済みになっていません")
		}
	})

	t.Run("異常系: バリデーションエラー（パスワード不一致）", func(t *testing.T) {
		t.Parallel()

		// テストユーザーを作成
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("update-password-mismatch@example.com").
			WithAtname("update_password_mismatch").
			Build()

		// 有効なトークンを作成
		plainToken := "valid_test_token_for_mismatch"
		tokenDigest := password_reset.HashToken(plainToken)
		testutil.NewPasswordResetTokenBuilderDB(t, db).
			WithUserID(userID).
			WithTokenDigest(tokenDigest).
			WithExpiresAt(time.Now().Add(1 * time.Hour)).
			Build()

		passwordResetTokenRepo := repository.NewPasswordResetTokenRepository(q)
		userPasswordRepo := repository.NewUserPasswordRepository(q)
		updateValidator := validator.NewPasswordUpdateValidator(passwordResetTokenRepo)
		uc := NewUpdatePasswordResetUsecase(db, passwordResetTokenRepo, userPasswordRepo, updateValidator)

		ctx := i18n.SetLocale(context.Background(), "ja")
		input := UpdatePasswordResetInput{
			Token:                plainToken,
			Password:             "newpassword123",
			PasswordConfirmation: "different456",
		}

		output, err := uc.Execute(ctx, input)
		if output != nil {
			t.Error("expected nil output for validation error")
		}

		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatal("expected ValidationError, but got nil")
		}
		if !ve.HasFieldError("password_confirmation") {
			t.Error("expected field error for password_confirmation")
		}
	})

	t.Run("異常系: バリデーションエラー（無効なトークン）", func(t *testing.T) {
		t.Parallel()

		passwordResetTokenRepo := repository.NewPasswordResetTokenRepository(q)
		userPasswordRepo := repository.NewUserPasswordRepository(q)
		updateValidator := validator.NewPasswordUpdateValidator(passwordResetTokenRepo)
		uc := NewUpdatePasswordResetUsecase(db, passwordResetTokenRepo, userPasswordRepo, updateValidator)

		ctx := i18n.SetLocale(context.Background(), "ja")
		input := UpdatePasswordResetInput{
			Token:                "non-existent-token",
			Password:             "newpassword123",
			PasswordConfirmation: "newpassword123",
		}

		output, err := uc.Execute(ctx, input)
		if output != nil {
			t.Error("expected nil output for validation error")
		}

		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatal("expected ValidationError, but got nil")
		}
		if len(ve.Global) == 0 {
			t.Error("expected global error for invalid token")
		}
	})
}
