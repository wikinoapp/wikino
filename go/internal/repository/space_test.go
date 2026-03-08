package repository

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestSpaceRepository_FindByIdentifier(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewSpaceRepository(q)

	// テストスペースを作成
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("find-by-identifier").
		WithName("Find By Identifier Space").
		Build()

	t.Run("存在するスペースを識別子で取得できる", func(t *testing.T) {
		space, err := repo.FindByIdentifier(context.Background(), "find-by-identifier")
		if err != nil {
			t.Fatalf("FindByIdentifier() error = %v", err)
		}
		if space == nil {
			t.Fatal("FindByIdentifier() returned nil, want space")
		}
		if space.ID != spaceID {
			t.Errorf("space.ID = %v, want %v", space.ID, spaceID)
		}
		if space.Identifier != "find-by-identifier" {
			t.Errorf("space.Identifier = %v, want find-by-identifier", space.Identifier)
		}
		if space.Name != "Find By Identifier Space" {
			t.Errorf("space.Name = %v, want Find By Identifier Space", space.Name)
		}
		if space.Plan != model.PlanSmall {
			t.Errorf("space.Plan = %v, want PlanSmall", space.Plan)
		}
		if space.DiscardedAt != nil {
			t.Errorf("space.DiscardedAt = %v, want nil", space.DiscardedAt)
		}
	})

	t.Run("存在しない識別子はnilを返す", func(t *testing.T) {
		space, err := repo.FindByIdentifier(context.Background(), "not-exist")
		if err != nil {
			t.Fatalf("FindByIdentifier() error = %v", err)
		}
		if space != nil {
			t.Errorf("FindByIdentifier() = %v, want nil", space)
		}
	})

	t.Run("削除済みスペースはnilを返す", func(t *testing.T) {
		discardedSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("discarded-space").
			WithName("Discarded Space").
			Build()

		now := time.Now()
		_, err := tx.ExecContext(
			context.Background(),
			`UPDATE spaces SET discarded_at = $1 WHERE id = $2`,
			now, string(discardedSpaceID),
		)
		if err != nil {
			t.Fatalf("削除済みスペースの更新に失敗: %v", err)
		}

		space, err := repo.FindByIdentifier(context.Background(), "discarded-space")
		if err != nil {
			t.Fatalf("FindByIdentifier() error = %v", err)
		}
		if space != nil {
			t.Errorf("FindByIdentifier() = %v, want nil for discarded space", space)
		}
	})

	t.Run("Planが正しく変換される", func(t *testing.T) {
		testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("free-plan-space").
			WithName("Free Plan Space").
			WithPlan(int32(model.PlanFree)).
			Build()

		space, err := repo.FindByIdentifier(context.Background(), "free-plan-space")
		if err != nil {
			t.Fatalf("FindByIdentifier() error = %v", err)
		}
		if space == nil {
			t.Fatal("FindByIdentifier() returned nil, want space")
		}
		if space.Plan != model.PlanFree {
			t.Errorf("space.Plan = %v, want PlanFree", space.Plan)
		}
	})
}
