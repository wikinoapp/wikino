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

func TestSpaceRepository_ListActiveByUser(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewSpaceRepository(q)
	ctx := context.Background()

	t.Run("アクティブメンバーの削除されていないスペースのみを返す", func(t *testing.T) {
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("active-user@example.com").
			WithAtname("active-user").
			Build()

		// 対象ユーザーがアクティブメンバーのスペース（含まれるべき）
		joinedSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("active-user-joined").
			WithName("Joined Space").
			Build()
		testutil.NewSpaceMemberBuilder(t, tx).
			WithSpaceID(joinedSpaceID).
			WithUserID(userID).
			Build()

		// 別ユーザーが所有するスペース（含まれないべき）
		otherUserID := testutil.NewUserBuilder(t, tx).
			WithEmail("other-user@example.com").
			WithAtname("other-user").
			Build()
		otherSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("other-user-space").
			WithName("Other User Space").
			Build()
		testutil.NewSpaceMemberBuilder(t, tx).
			WithSpaceID(otherSpaceID).
			WithUserID(otherUserID).
			Build()

		spaces, err := repo.ListActiveByUser(ctx, userID)
		if err != nil {
			t.Fatalf("ListActiveByUser() error = %v", err)
		}
		if len(spaces) != 1 {
			t.Fatalf("len(spaces) = %d, want 1", len(spaces))
		}
		if spaces[0].ID != joinedSpaceID {
			t.Errorf("spaces[0].ID = %v, want %v", spaces[0].ID, joinedSpaceID)
		}
		if spaces[0].Identifier != "active-user-joined" {
			t.Errorf("spaces[0].Identifier = %v, want active-user-joined", spaces[0].Identifier)
		}
	})

	t.Run("削除済みのスペースは除外される", func(t *testing.T) {
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("discarded-target@example.com").
			WithAtname("discarded-target").
			Build()

		// 通常のスペース
		liveSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("live-space").
			WithName("Live Space").
			Build()
		testutil.NewSpaceMemberBuilder(t, tx).
			WithSpaceID(liveSpaceID).
			WithUserID(userID).
			Build()

		// 削除済みスペース
		discardedSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("discarded-target-space").
			WithName("Discarded Space").
			Build()
		testutil.NewSpaceMemberBuilder(t, tx).
			WithSpaceID(discardedSpaceID).
			WithUserID(userID).
			Build()
		if _, err := tx.ExecContext(
			ctx,
			`UPDATE spaces SET discarded_at = $1 WHERE id = $2`,
			time.Now(), string(discardedSpaceID),
		); err != nil {
			t.Fatalf("削除済みスペースの更新に失敗: %v", err)
		}

		spaces, err := repo.ListActiveByUser(ctx, userID)
		if err != nil {
			t.Fatalf("ListActiveByUser() error = %v", err)
		}
		if len(spaces) != 1 {
			t.Fatalf("len(spaces) = %d, want 1", len(spaces))
		}
		if spaces[0].ID != liveSpaceID {
			t.Errorf("spaces[0].ID = %v, want %v", spaces[0].ID, liveSpaceID)
		}
	})

	t.Run("退会したメンバーのスペースは除外される", func(t *testing.T) {
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("inactive-target@example.com").
			WithAtname("inactive-target").
			Build()

		// アクティブメンバーのスペース
		activeSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("inactive-target-active").
			WithName("Active Space").
			Build()
		testutil.NewSpaceMemberBuilder(t, tx).
			WithSpaceID(activeSpaceID).
			WithUserID(userID).
			Build()

		// 退会メンバーのスペース（active = false）
		inactiveSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("inactive-target-inactive").
			WithName("Inactive Space").
			Build()
		testutil.NewSpaceMemberBuilder(t, tx).
			WithSpaceID(inactiveSpaceID).
			WithUserID(userID).
			WithActive(false).
			Build()

		spaces, err := repo.ListActiveByUser(ctx, userID)
		if err != nil {
			t.Fatalf("ListActiveByUser() error = %v", err)
		}
		if len(spaces) != 1 {
			t.Fatalf("len(spaces) = %d, want 1", len(spaces))
		}
		if spaces[0].ID != activeSpaceID {
			t.Errorf("spaces[0].ID = %v, want %v", spaces[0].ID, activeSpaceID)
		}
	})

	t.Run("space_members.joined_at の降順で返す", func(t *testing.T) {
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("ordered-user@example.com").
			WithAtname("ordered-user").
			Build()

		baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		oldSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("ordered-old").
			WithName("Old Space").
			Build()
		testutil.NewSpaceMemberBuilder(t, tx).
			WithSpaceID(oldSpaceID).
			WithUserID(userID).
			WithJoinedAt(baseTime).
			Build()

		middleSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("ordered-middle").
			WithName("Middle Space").
			Build()
		testutil.NewSpaceMemberBuilder(t, tx).
			WithSpaceID(middleSpaceID).
			WithUserID(userID).
			WithJoinedAt(baseTime.AddDate(0, 1, 0)).
			Build()

		newSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("ordered-new").
			WithName("New Space").
			Build()
		testutil.NewSpaceMemberBuilder(t, tx).
			WithSpaceID(newSpaceID).
			WithUserID(userID).
			WithJoinedAt(baseTime.AddDate(0, 2, 0)).
			Build()

		spaces, err := repo.ListActiveByUser(ctx, userID)
		if err != nil {
			t.Fatalf("ListActiveByUser() error = %v", err)
		}
		if len(spaces) != 3 {
			t.Fatalf("len(spaces) = %d, want 3", len(spaces))
		}
		want := []model.SpaceID{newSpaceID, middleSpaceID, oldSpaceID}
		for i, expected := range want {
			if spaces[i].ID != expected {
				t.Errorf("spaces[%d].ID = %v, want %v", i, spaces[i].ID, expected)
			}
		}
	})

	t.Run("参加スペースが無いユーザーは空スライスを返す", func(t *testing.T) {
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("empty-user@example.com").
			WithAtname("empty-user").
			Build()

		spaces, err := repo.ListActiveByUser(ctx, userID)
		if err != nil {
			t.Fatalf("ListActiveByUser() error = %v", err)
		}
		if len(spaces) != 0 {
			t.Errorf("len(spaces) = %d, want 0", len(spaces))
		}
	})
}
