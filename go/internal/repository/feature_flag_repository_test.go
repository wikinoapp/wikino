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
			t.Fatalf("unexpected error: %v", err)
		}
		if !enabled {
			t.Error("expected enabled to be true, got false")
		}
	})

	t.Run("フラグが無効なユーザーに対してfalseを返す", func(t *testing.T) {
		otherUserID := testutil.NewUserBuilder(t, tx).
			WithEmail("ff-disabled@example.com").
			WithAtname("ff_disabled_user").
			Build()

		enabled, err := repo.IsEnabled(ctx, otherUserID, model.FeatureFlagName("go_page_edit"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if enabled {
			t.Error("expected enabled to be false, got true")
		}
	})

	t.Run("存在しないフラグ名に対してfalseを返す", func(t *testing.T) {
		enabled, err := repo.IsEnabled(ctx, userID, model.FeatureFlagName("nonexistent_flag"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if enabled {
			t.Error("expected enabled to be false, got true")
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
			t.Fatalf("unexpected error: %v", err)
		}
		if !enabled {
			t.Error("expected enabled to be true, got false")
		}
	})

	t.Run("device_tokenでフラグが無効な場合falseを返す", func(t *testing.T) {
		enabled, err := repo.IsEnabledForDevice(ctx, "unknown-device-token", "", model.FeatureFlagName("go_page_edit"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if enabled {
			t.Error("expected enabled to be false, got true")
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
			t.Fatalf("unexpected error: %v", err)
		}
		if !enabled {
			t.Error("expected enabled to be true, got false")
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
			t.Fatalf("unexpected error: %v", err)
		}
		if enabled {
			t.Error("expected enabled to be false, got true")
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
			t.Fatalf("unexpected error: %v", err)
		}
		if !enabled {
			t.Error("expected enabled to be true for device_token flag, got false")
		}

		// user_idのフラグはsessionToken経由で有効
		enabled, err = repo.IsEnabledForDevice(ctx, "device-token-both", sessionToken, model.FeatureFlagName("go_page_edit_user_only"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !enabled {
			t.Error("expected enabled to be true for user_id flag via session, got false")
		}
	})

	t.Run("両方のCookieが空の場合falseを返す", func(t *testing.T) {
		enabled, err := repo.IsEnabledForDevice(ctx, "", "", model.FeatureFlagName("go_page_edit"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if enabled {
			t.Error("expected enabled to be false, got true")
		}
	})

	t.Run("存在しないフラグ名に対してfalseを返す", func(t *testing.T) {
		enabled, err := repo.IsEnabledForDevice(ctx, "device-token-enabled", "", model.FeatureFlagName("nonexistent_flag"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if enabled {
			t.Error("expected enabled to be false, got true")
		}
	})
}
