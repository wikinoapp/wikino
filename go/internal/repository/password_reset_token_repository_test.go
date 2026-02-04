package repository

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestPasswordResetTokenRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewPasswordResetTokenRepository(q)

	// テストユーザーを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("create-token@example.com").
		WithAtname("create_token_user").
		Build()

	t.Run("パスワードリセットトークンを作成できる", func(t *testing.T) {
		expiresAt := time.Now().Add(1 * time.Hour)
		input := CreatePasswordResetTokenInput{
			UserID:      userID,
			TokenDigest: "test_token_digest_create",
			ExpiresAt:   expiresAt,
		}

		token, err := repo.Create(context.Background(), input)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if token == nil {
			t.Fatal("Create() returned nil, want password reset token")
		}
		if token.UserID != userID {
			t.Errorf("token.UserID = %v, want %v", token.UserID, userID)
		}
		if token.TokenDigest != "test_token_digest_create" {
			t.Errorf("token.TokenDigest = %v, want test_token_digest_create", token.TokenDigest)
		}
		if token.UsedAt != nil {
			t.Errorf("token.UsedAt = %v, want nil", token.UsedAt)
		}
	})
}

func TestPasswordResetTokenRepository_FindByTokenDigest(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewPasswordResetTokenRepository(q)

	// テストユーザーを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("find-token@example.com").
		WithAtname("find_token_user").
		Build()

	// テストトークンを作成
	testutil.NewPasswordResetTokenBuilder(t, tx).
		WithUserID(userID).
		WithTokenDigest("unique_token_digest").
		Build()

	t.Run("トークンダイジェストでトークンを取得できる", func(t *testing.T) {
		token, err := repo.FindByTokenDigest(context.Background(), "unique_token_digest")
		if err != nil {
			t.Fatalf("FindByTokenDigest() error = %v", err)
		}
		if token == nil {
			t.Fatal("FindByTokenDigest() returned nil, want password reset token")
		}
		if token.TokenDigest != "unique_token_digest" {
			t.Errorf("token.TokenDigest = %v, want unique_token_digest", token.TokenDigest)
		}
		if token.UserID != userID {
			t.Errorf("token.UserID = %v, want %v", token.UserID, userID)
		}
	})

	t.Run("存在しないトークンダイジェストはnilを返す", func(t *testing.T) {
		token, err := repo.FindByTokenDigest(context.Background(), "nonexistent_token_digest")
		if err != nil {
			t.Fatalf("FindByTokenDigest() error = %v", err)
		}
		if token != nil {
			t.Errorf("FindByTokenDigest() = %v, want nil", token)
		}
	})
}

func TestPasswordResetTokenRepository_MarkAsUsed(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewPasswordResetTokenRepository(q)

	// テストユーザーを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("mark-used@example.com").
		WithAtname("mark_used_user").
		Build()

	// テストトークンを作成
	tokenID := testutil.NewPasswordResetTokenBuilder(t, tx).
		WithUserID(userID).
		WithTokenDigest("mark_used_token_digest").
		Build()

	t.Run("トークンを使用済みにマークできる", func(t *testing.T) {
		err := repo.MarkAsUsed(context.Background(), tokenID)
		if err != nil {
			t.Fatalf("MarkAsUsed() error = %v", err)
		}

		// 更新後のトークンを確認
		token, err := repo.FindByTokenDigest(context.Background(), "mark_used_token_digest")
		if err != nil {
			t.Fatalf("FindByTokenDigest() error = %v", err)
		}
		if token == nil {
			t.Fatal("FindByTokenDigest() returned nil, want password reset token")
		}
		if token.UsedAt == nil {
			t.Error("token.UsedAt = nil, want not nil")
		}
	})
}

func TestPasswordResetTokenRepository_DeleteUnusedByUserID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewPasswordResetTokenRepository(q)

	// テストユーザーを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("delete-unused@example.com").
		WithAtname("delete_unused_user").
		Build()

	// 未使用のトークンを作成
	testutil.NewPasswordResetTokenBuilder(t, tx).
		WithUserID(userID).
		WithTokenDigest("unused_token_to_delete").
		Build()

	// 使用済みのトークンを作成
	testutil.NewPasswordResetTokenBuilder(t, tx).
		WithUserID(userID).
		WithTokenDigest("used_token_not_to_delete").
		BuildUsed()

	t.Run("未使用のトークンのみ削除できる", func(t *testing.T) {
		err := repo.DeleteUnusedByUserID(context.Background(), userID)
		if err != nil {
			t.Fatalf("DeleteUnusedByUserID() error = %v", err)
		}

		// 未使用のトークンが削除されていることを確認
		unusedToken, err := repo.FindByTokenDigest(context.Background(), "unused_token_to_delete")
		if err != nil {
			t.Fatalf("FindByTokenDigest() error = %v", err)
		}
		if unusedToken != nil {
			t.Errorf("未使用トークンが削除されていません: %v", unusedToken)
		}

		// 使用済みのトークンは削除されていないことを確認
		usedToken, err := repo.FindByTokenDigest(context.Background(), "used_token_not_to_delete")
		if err != nil {
			t.Fatalf("FindByTokenDigest() error = %v", err)
		}
		if usedToken == nil {
			t.Error("使用済みトークンが誤って削除されています")
		}
	})
}

func TestPasswordResetToken_IsExpired(t *testing.T) {
	t.Parallel()

	t.Run("有効期限内は期限切れではない", func(t *testing.T) {
		token := &model.PasswordResetToken{
			ExpiresAt: time.Now().Add(30 * time.Minute),
		}
		if token.IsExpired() {
			t.Error("IsExpired() = true, want false (30 minutes left)")
		}
	})

	t.Run("有効期限を過ぎると期限切れ", func(t *testing.T) {
		token := &model.PasswordResetToken{
			ExpiresAt: time.Now().Add(-1 * time.Minute),
		}
		if !token.IsExpired() {
			t.Error("IsExpired() = false, want true (expired 1 minute ago)")
		}
	})
}

func TestPasswordResetToken_IsUsed(t *testing.T) {
	t.Parallel()

	t.Run("UsedAtがnilの場合は未使用", func(t *testing.T) {
		token := &model.PasswordResetToken{
			UsedAt: nil,
		}
		if token.IsUsed() {
			t.Error("IsUsed() = true, want false")
		}
	})

	t.Run("UsedAtが設定されている場合は使用済み", func(t *testing.T) {
		now := time.Now()
		token := &model.PasswordResetToken{
			UsedAt: &now,
		}
		if !token.IsUsed() {
			t.Error("IsUsed() = false, want true")
		}
	})
}

func TestPasswordResetToken_IsValid(t *testing.T) {
	t.Parallel()

	t.Run("未使用かつ有効期限内は有効", func(t *testing.T) {
		token := &model.PasswordResetToken{
			UsedAt:    nil,
			ExpiresAt: time.Now().Add(30 * time.Minute),
		}
		if !token.IsValid() {
			t.Error("IsValid() = false, want true")
		}
	})

	t.Run("使用済みの場合は無効", func(t *testing.T) {
		now := time.Now()
		token := &model.PasswordResetToken{
			UsedAt:    &now,
			ExpiresAt: time.Now().Add(30 * time.Minute),
		}
		if token.IsValid() {
			t.Error("IsValid() = true, want false (used)")
		}
	})

	t.Run("有効期限切れの場合は無効", func(t *testing.T) {
		token := &model.PasswordResetToken{
			UsedAt:    nil,
			ExpiresAt: time.Now().Add(-1 * time.Minute),
		}
		if token.IsValid() {
			t.Error("IsValid() = true, want false (expired)")
		}
	})

	t.Run("使用済みかつ有効期限切れの場合は無効", func(t *testing.T) {
		now := time.Now()
		token := &model.PasswordResetToken{
			UsedAt:    &now,
			ExpiresAt: time.Now().Add(-1 * time.Minute),
		}
		if token.IsValid() {
			t.Error("IsValid() = true, want false (used and expired)")
		}
	})
}
