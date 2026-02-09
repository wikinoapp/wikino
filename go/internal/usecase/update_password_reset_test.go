package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/password_reset"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestUpdatePasswordResetUsecase_Execute(t *testing.T) {
	db := testutil.GetTestDB()
	q := query.New(db)

	// テストユーザーを作成
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("update-password@example.com").
		WithAtname("update_password_user").
		Build()

	// ユーザーのパスワードを作成
	testutil.NewUserPasswordBuilderDB(t, db).
		WithUserID(userID).
		WithPasswordDigest("$2a$10$oldpasswordhashhashhash").
		Build()

	// 有効なトークンを作成
	plainToken := "valid_test_token_for_update"
	tokenDigest := password_reset.HashToken(plainToken)
	tokenID := testutil.NewPasswordResetTokenBuilderDB(t, db).
		WithUserID(userID).
		WithTokenDigest(tokenDigest).
		WithExpiresAt(time.Now().Add(1 * time.Hour)).
		Build()

	passwordResetTokenRepo := repository.NewPasswordResetTokenRepository(q)
	userPasswordRepo := repository.NewUserPasswordRepository(q)
	uc := NewUpdatePasswordResetUsecase(db, passwordResetTokenRepo, userPasswordRepo)

	ctx := i18n.SetLocale(context.Background(), "ja")
	input := UpdatePasswordResetInput{
		TokenID:     tokenID,
		UserID:      userID,
		NewPassword: "newpassword123",
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
}
