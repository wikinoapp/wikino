package repository

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestUserPasswordRepository_FindByUserID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewUserPasswordRepository(q)

	// テストユーザーとパスワードを作成
	passwordDigest := "$2a$10$XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("password@example.com").
		WithAtname("passworduser").
		BuildWithPassword(passwordDigest)

	t.Run("存在するユーザーのパスワードを取得できる", func(t *testing.T) {
		password, err := repo.FindByUserID(context.Background(), userID)
		if err != nil {
			t.Fatalf("FindByUserID()のエラー = %v", err)
		}
		if password == nil {
			t.Fatal("FindByUserID()がnilを返した、期待値 = パスワード")
		}
		if password.UserID != userID {
			t.Errorf("password.UserID = %v、期待値 = %v", password.UserID, userID)
		}
		if password.PasswordDigest != passwordDigest {
			t.Errorf("password.PasswordDigest = %v、期待値 = %v", password.PasswordDigest, passwordDigest)
		}
	})

	t.Run("存在しないユーザーIDはnilを返す", func(t *testing.T) {
		password, err := repo.FindByUserID(context.Background(), "00000000-0000-0000-0000-000000000000")
		if err != nil {
			t.Fatalf("FindByUserID()のエラー = %v", err)
		}
		if password != nil {
			t.Errorf("FindByUserID() = %v、期待値 = nil", password)
		}
	})
}

func TestUserPasswordRepository_UpdatePasswordDigest(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewUserPasswordRepository(q)

	// テストユーザーとパスワードを作成
	oldPasswordDigest := "$2a$10$XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("update-password@example.com").
		WithAtname("updatepassworduser").
		BuildWithPassword(oldPasswordDigest)

	t.Run("パスワードダイジェストを更新できる", func(t *testing.T) {
		newPasswordDigest := "$2a$10$YYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYY"

		err := repo.UpdatePasswordDigest(context.Background(), userID, newPasswordDigest)
		if err != nil {
			t.Fatalf("UpdatePasswordDigest()のエラー = %v", err)
		}

		// 更新後のパスワードを確認
		password, err := repo.FindByUserID(context.Background(), userID)
		if err != nil {
			t.Fatalf("FindByUserID()のエラー = %v", err)
		}
		if password == nil {
			t.Fatal("FindByUserID()がnilを返した、期待値 = パスワード")
		}
		if password.PasswordDigest != newPasswordDigest {
			t.Errorf("password.PasswordDigest = %v、期待値 = %v", password.PasswordDigest, newPasswordDigest)
		}
	})
}
