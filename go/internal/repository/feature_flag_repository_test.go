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

func TestFeatureFlagRepository_IsEnabledBySessionToken(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewFeatureFlagRepository(q)
	ctx := context.Background()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("ff-session@example.com").
		WithAtname("ff_session_user").
		Build()

	sessionToken := testutil.NewSessionBuilder(t, tx).
		WithUserID(userID).
		WithToken("ff-test-session-token").
		BuildAndGetToken()

	testutil.NewFeatureFlagBuilder(t, tx).
		WithUserID(userID).
		WithName("go_page_edit").
		Build()

	t.Run("フラグが有効なユーザーのセッショントークンに対してtrueを返す", func(t *testing.T) {
		enabled, err := repo.IsEnabledBySessionToken(ctx, sessionToken, model.FeatureFlagName("go_page_edit"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !enabled {
			t.Error("expected enabled to be true, got false")
		}
	})

	t.Run("フラグが無効なユーザーのセッショントークンに対してfalseを返す", func(t *testing.T) {
		otherUserID := testutil.NewUserBuilder(t, tx).
			WithEmail("ff-session-other@example.com").
			WithAtname("ff_session_other").
			Build()

		otherToken := testutil.NewSessionBuilder(t, tx).
			WithUserID(otherUserID).
			WithToken("ff-other-session-token").
			BuildAndGetToken()

		enabled, err := repo.IsEnabledBySessionToken(ctx, otherToken, model.FeatureFlagName("go_page_edit"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if enabled {
			t.Error("expected enabled to be false, got true")
		}
	})

	t.Run("存在しないセッショントークンに対してfalseを返す", func(t *testing.T) {
		enabled, err := repo.IsEnabledBySessionToken(ctx, "nonexistent-token", model.FeatureFlagName("go_page_edit"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if enabled {
			t.Error("expected enabled to be false, got true")
		}
	})

	t.Run("存在しないフラグ名に対してfalseを返す", func(t *testing.T) {
		enabled, err := repo.IsEnabledBySessionToken(ctx, sessionToken, model.FeatureFlagName("nonexistent_flag"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if enabled {
			t.Error("expected enabled to be false, got true")
		}
	})
}
