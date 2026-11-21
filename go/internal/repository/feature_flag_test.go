package repository

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestFeatureFlagRepository_IsEnabled(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewFeatureFlagRepository(q)
	ctx := context.Background()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("ff-enabled@example.com").
		WithAtname("ff_enabled_user").
		Build()

	testutil.NewFeatureFlagBuilder(t, tx).
		WithUserID(userID).
		WithName("go_page_edit").
		Build()

	t.Run("フラグが有効なユーザーに対してtrueを返す", func(t *testing.T) {
		enabled, err := repo.IsEnabled(ctx, userID, model.FeatureFlagName("go_page_edit"))
		if err != nil {
			t.Fatalf("予期しないエラー: %v", err)
		}
		if !enabled {
			t.Error("enabled = false、期待値 = true")
		}
	})

	t.Run("フラグが無効なユーザーに対してfalseを返す", func(t *testing.T) {
		otherUserID := testutil.NewUserBuilder(t, tx).
			WithEmail("ff-disabled@example.com").
			WithAtname("ff_disabled_user").
			Build()

		enabled, err := repo.IsEnabled(ctx, otherUserID, model.FeatureFlagName("go_page_edit"))
		if err != nil {
			t.Fatalf("予期しないエラー: %v", err)
		}
		if enabled {
			t.Error("enabled = true、期待値 = false")
		}
	})

	t.Run("存在しないフラグ名に対してfalseを返す", func(t *testing.T) {
		enabled, err := repo.IsEnabled(ctx, userID, model.FeatureFlagName("nonexistent_flag"))
		if err != nil {
			t.Fatalf("予期しないエラー: %v", err)
		}
		if enabled {
			t.Error("enabled = true、期待値 = false")
		}
	})
}

// Rails側の削除経路が頼るON DELETE CASCADEの契約を検証する。
// usersの行を直接DELETEしたとき、feature_flagsへの明示的な
// DELETEなしでフラグも一緒に消えること。
func TestFeatureFlagRepository_CascadeOnUserDelete(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("ff-cascade@example.com").
		WithAtname("ff_cascade_user").
		Build()

	userFlagID := testutil.NewFeatureFlagBuilder(t, tx).
		WithUserID(userID).
		WithName("go_cascade_test").
		Build()

	t.Run("ユーザーの行を直接DELETEするとユーザー紐づけのフラグも消える", func(t *testing.T) {
		// 親のusersの行を直接削除 (アプリケーションコードを経由しない)
		_, err := tx.ExecContext(ctx, "DELETE FROM users WHERE id = $1", string(userID))
		if err != nil {
			t.Fatalf("DELETE usersのエラー = %v", err)
		}

		var userFlagCount int
		if err := tx.QueryRowContext(
			ctx,
			"SELECT COUNT(*) FROM feature_flags WHERE id = $1",
			string(userFlagID),
		).Scan(&userFlagCount); err != nil {
			t.Fatalf("SELECT COUNT(*) FROM feature_flagsのエラー = %v", err)
		}
		if userFlagCount != 0 {
			t.Errorf("ユーザーのフラグの件数 = %d、期待値 = 0 (カスケード削除されていない)", userFlagCount)
		}
	})
}
