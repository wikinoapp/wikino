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

	_, tx := testutil.SetupTx(t)
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
			t.Fatalf("Create()のエラー = %v", err)
		}
		if token == nil {
			t.Fatal("Create()がnilを返した、期待値 = パスワードリセットトークン")
		}
		if token.UserID != userID {
			t.Errorf("token.UserID = %v、期待値 = %v", token.UserID, userID)
		}
		if token.TokenDigest != "test_token_digest_create" {
			t.Errorf("token.TokenDigest = %v、期待値 = test_token_digest_create", token.TokenDigest)
		}
		if token.UsedAt != nil {
			t.Errorf("token.UsedAt = %v、期待値 = nil", token.UsedAt)
		}
	})
}

func TestPasswordResetTokenRepository_FindByTokenDigest(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
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
			t.Fatalf("FindByTokenDigest()のエラー = %v", err)
		}
		if token == nil {
			t.Fatal("FindByTokenDigest()がnilを返した、期待値 = パスワードリセットトークン")
		}
		if token.TokenDigest != "unique_token_digest" {
			t.Errorf("token.TokenDigest = %v、期待値 = unique_token_digest", token.TokenDigest)
		}
		if token.UserID != userID {
			t.Errorf("token.UserID = %v、期待値 = %v", token.UserID, userID)
		}
	})

	t.Run("存在しないトークンダイジェストはnilを返す", func(t *testing.T) {
		token, err := repo.FindByTokenDigest(context.Background(), "nonexistent_token_digest")
		if err != nil {
			t.Fatalf("FindByTokenDigest()のエラー = %v", err)
		}
		if token != nil {
			t.Errorf("FindByTokenDigest() = %v、期待値 = nil", token)
		}
	})
}

func TestPasswordResetTokenRepository_MarkAsUsed(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
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
			t.Fatalf("MarkAsUsed()のエラー = %v", err)
		}

		// 更新後のトークンを確認
		token, err := repo.FindByTokenDigest(context.Background(), "mark_used_token_digest")
		if err != nil {
			t.Fatalf("FindByTokenDigest()のエラー = %v", err)
		}
		if token == nil {
			t.Fatal("FindByTokenDigest()がnilを返した、期待値 = パスワードリセットトークン")
		}
		if token.UsedAt == nil {
			t.Error("token.UsedAt = nil、期待値 = nilではない")
		}
	})
}

func TestPasswordResetTokenRepository_DeleteUnusedByUserID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
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
			t.Fatalf("DeleteUnusedByUserID()のエラー = %v", err)
		}

		// 未使用のトークンが削除されていることを確認
		unusedToken, err := repo.FindByTokenDigest(context.Background(), "unused_token_to_delete")
		if err != nil {
			t.Fatalf("FindByTokenDigest()のエラー = %v", err)
		}
		if unusedToken != nil {
			t.Errorf("未使用トークンが削除されていません: %v", unusedToken)
		}

		// 使用済みのトークンは削除されていないことを確認
		usedToken, err := repo.FindByTokenDigest(context.Background(), "used_token_not_to_delete")
		if err != nil {
			t.Fatalf("FindByTokenDigest()のエラー = %v", err)
		}
		if usedToken == nil {
			t.Error("使用済みトークンが誤って削除されています")
		}
	})
}

// Rails側の削除経路が頼るON DELETE CASCADEの契約を検証する。
// usersの行を直接DELETEしたとき、password_reset_tokensへの明示的な
// DELETEなしでトークンも一緒に消えること。
func TestPasswordResetTokenRepository_CascadeOnUserDelete(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewPasswordResetTokenRepository(q)
	ctx := context.Background()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("prt-cascade@example.com").
		WithAtname("prt_cascade_user").
		Build()

	testutil.NewPasswordResetTokenBuilder(t, tx).
		WithUserID(userID).
		WithTokenDigest("cascade_user_delete_token").
		Build()

	t.Run("ユーザーの行を直接DELETEするとトークンも消える", func(t *testing.T) {
		// 親のusersの行を直接削除 (アプリケーションコードを経由しない)
		_, err := tx.ExecContext(ctx, "DELETE FROM users WHERE id = $1", string(userID))
		if err != nil {
			t.Fatalf("DELETE usersのエラー = %v", err)
		}

		token, err := repo.FindByTokenDigest(ctx, "cascade_user_delete_token")
		if err != nil {
			t.Fatalf("FindByTokenDigest()のエラー = %v", err)
		}
		if token != nil {
			t.Errorf("FindByTokenDigest() = %v、期待値 = nil (トークンがカスケード削除されていない)", token)
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
			t.Error("IsExpired() = true、期待値 = false (残り30分)")
		}
	})

	t.Run("有効期限を過ぎると期限切れ", func(t *testing.T) {
		token := &model.PasswordResetToken{
			ExpiresAt: time.Now().Add(-1 * time.Minute),
		}
		if !token.IsExpired() {
			t.Error("IsExpired() = false、期待値 = true (1分前に期限切れ)")
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
			t.Error("IsUsed() = true、期待値 = false")
		}
	})

	t.Run("UsedAtが設定されている場合は使用済み", func(t *testing.T) {
		now := time.Now()
		token := &model.PasswordResetToken{
			UsedAt: &now,
		}
		if !token.IsUsed() {
			t.Error("IsUsed() = false、期待値 = true")
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
			t.Error("IsValid() = false、期待値 = true")
		}
	})

	t.Run("使用済みの場合は無効", func(t *testing.T) {
		now := time.Now()
		token := &model.PasswordResetToken{
			UsedAt:    &now,
			ExpiresAt: time.Now().Add(30 * time.Minute),
		}
		if token.IsValid() {
			t.Error("IsValid() = true、期待値 = false (使用済み)")
		}
	})

	t.Run("有効期限切れの場合は無効", func(t *testing.T) {
		token := &model.PasswordResetToken{
			UsedAt:    nil,
			ExpiresAt: time.Now().Add(-1 * time.Minute),
		}
		if token.IsValid() {
			t.Error("IsValid() = true、期待値 = false (期限切れ)")
		}
	})

	t.Run("使用済みかつ有効期限切れの場合は無効", func(t *testing.T) {
		now := time.Now()
		token := &model.PasswordResetToken{
			UsedAt:    &now,
			ExpiresAt: time.Now().Add(-1 * time.Minute),
		}
		if token.IsValid() {
			t.Error("IsValid() = true、期待値 = false (使用済みかつ期限切れ)")
		}
	})
}
