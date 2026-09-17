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

func TestFeatureFlagRepository_IsEnabledForDevice(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewFeatureFlagRepository(q)
	ctx := context.Background()

	t.Run("device_tokenでフラグが有効な場合trueを返す", func(t *testing.T) {
		testutil.NewFeatureFlagBuilder(t, tx).
			WithDeviceToken("device-token-enabled").
			WithName("go_page_edit").
			Build()

		enabled, err := repo.IsEnabledForDevice(ctx, "device-token-enabled", "", model.FeatureFlagName("go_page_edit"))
		if err != nil {
			t.Fatalf("予期しないエラー: %v", err)
		}
		if !enabled {
			t.Error("enabled = false、期待値 = true")
		}
	})

	t.Run("device_tokenでフラグが無効な場合falseを返す", func(t *testing.T) {
		enabled, err := repo.IsEnabledForDevice(ctx, "unknown-device-token", "", model.FeatureFlagName("go_page_edit"))
		if err != nil {
			t.Fatalf("予期しないエラー: %v", err)
		}
		if enabled {
			t.Error("enabled = true、期待値 = false")
		}
	})

	t.Run("セッショントークン経由のuser_idでフラグが有効な場合trueを返す", func(t *testing.T) {
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("ff-device-session@example.com").
			WithAtname("ff_device_session").
			Build()

		sessionToken := testutil.NewSessionBuilder(t, tx).
			WithUserID(userID).
			WithToken("ff-device-session-token").
			BuildAndGetToken()

		testutil.NewFeatureFlagBuilder(t, tx).
			WithUserID(userID).
			WithName("go_page_edit_session").
			Build()

		enabled, err := repo.IsEnabledForDevice(ctx, "", sessionToken, model.FeatureFlagName("go_page_edit_session"))
		if err != nil {
			t.Fatalf("予期しないエラー: %v", err)
		}
		if !enabled {
			t.Error("enabled = false、期待値 = true")
		}
	})

	t.Run("セッショントークン経由のuser_idでフラグが無効な場合falseを返す", func(t *testing.T) {
		otherUserID := testutil.NewUserBuilder(t, tx).
			WithEmail("ff-device-other@example.com").
			WithAtname("ff_device_other").
			Build()

		otherToken := testutil.NewSessionBuilder(t, tx).
			WithUserID(otherUserID).
			WithToken("ff-device-other-token").
			BuildAndGetToken()

		enabled, err := repo.IsEnabledForDevice(ctx, "", otherToken, model.FeatureFlagName("go_page_edit_session"))
		if err != nil {
			t.Fatalf("予期しないエラー: %v", err)
		}
		if enabled {
			t.Error("enabled = true、期待値 = false")
		}
	})

	t.Run("device_tokenとsessionTokenの両方で判定できる", func(t *testing.T) {
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("ff-device-both@example.com").
			WithAtname("ff_device_both").
			Build()

		sessionToken := testutil.NewSessionBuilder(t, tx).
			WithUserID(userID).
			WithToken("ff-device-both-token").
			BuildAndGetToken()

		// device_tokenでフラグを設定
		testutil.NewFeatureFlagBuilder(t, tx).
			WithDeviceToken("device-token-both").
			WithName("go_page_edit_both").
			Build()

		// user_idでフラグを設定
		testutil.NewFeatureFlagBuilder(t, tx).
			WithUserID(userID).
			WithName("go_page_edit_user_only").
			Build()

		// device_tokenのフラグはdevice_tokenで有効
		enabled, err := repo.IsEnabledForDevice(ctx, "device-token-both", sessionToken, model.FeatureFlagName("go_page_edit_both"))
		if err != nil {
			t.Fatalf("予期しないエラー: %v", err)
		}
		if !enabled {
			t.Error("device_tokenのフラグでenabled = false、期待値 = true")
		}

		// user_idのフラグはsessionToken経由で有効
		enabled, err = repo.IsEnabledForDevice(ctx, "device-token-both", sessionToken, model.FeatureFlagName("go_page_edit_user_only"))
		if err != nil {
			t.Fatalf("予期しないエラー: %v", err)
		}
		if !enabled {
			t.Error("セッション経由のuser_idのフラグでenabled = false、期待値 = true")
		}
	})

	t.Run("両方のCookieが空の場合falseを返す", func(t *testing.T) {
		enabled, err := repo.IsEnabledForDevice(ctx, "", "", model.FeatureFlagName("go_page_edit"))
		if err != nil {
			t.Fatalf("予期しないエラー: %v", err)
		}
		if enabled {
			t.Error("enabled = true、期待値 = false")
		}
	})

	t.Run("存在しないフラグ名に対してfalseを返す", func(t *testing.T) {
		enabled, err := repo.IsEnabledForDevice(ctx, "device-token-enabled", "", model.FeatureFlagName("nonexistent_flag"))
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

	deviceFlagID := testutil.NewFeatureFlagBuilder(t, tx).
		WithDeviceToken("ff-cascade-device-token").
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

		// デバイストークンのフラグ (user_idがNULL) は連鎖削除に巻き込まれないこと。
		var deviceFlagCount int
		if err := tx.QueryRowContext(
			ctx,
			"SELECT COUNT(*) FROM feature_flags WHERE id = $1",
			string(deviceFlagID),
		).Scan(&deviceFlagCount); err != nil {
			t.Fatalf("SELECT COUNT(*) FROM feature_flagsのエラー = %v", err)
		}
		if deviceFlagCount != 1 {
			t.Errorf("デバイスのフラグの件数 = %d、期待値 = 1 (カスケード削除されている)", deviceFlagCount)
		}
	})
}
