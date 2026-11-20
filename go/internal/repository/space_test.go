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
			t.Fatalf("FindByIdentifier()のエラー = %v", err)
		}
		if space == nil {
			t.Fatal("FindByIdentifier()がnilを返した、期待値 = スペース")
		}
		if space.ID != spaceID {
			t.Errorf("space.ID = %v、期待値 = %v", space.ID, spaceID)
		}
		if space.Identifier != "find-by-identifier" {
			t.Errorf("space.Identifier = %v、期待値 = find-by-identifier", space.Identifier)
		}
		if space.Name != "Find By Identifier Space" {
			t.Errorf("space.Name = %v、期待値 = Find By Identifier Space", space.Name)
		}
		if space.Plan != model.PlanSmall {
			t.Errorf("space.Plan = %v、期待値 = PlanSmall", space.Plan)
		}
		if space.DiscardedAt != nil {
			t.Errorf("space.DiscardedAt = %v、期待値 = nil", space.DiscardedAt)
		}
	})

	t.Run("存在しない識別子はnilを返す", func(t *testing.T) {
		space, err := repo.FindByIdentifier(context.Background(), "not-exist")
		if err != nil {
			t.Fatalf("FindByIdentifier()のエラー = %v", err)
		}
		if space != nil {
			t.Errorf("FindByIdentifier() = %v、期待値 = nil", space)
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
			t.Fatalf("FindByIdentifier()のエラー = %v", err)
		}
		if space != nil {
			t.Errorf("FindByIdentifier() = %v、期待値 = nil (削除済みスペース)", space)
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
			t.Fatalf("FindByIdentifier()のエラー = %v", err)
		}
		if space == nil {
			t.Fatal("FindByIdentifier()がnilを返した、期待値 = スペース")
		}
		if space.Plan != model.PlanFree {
			t.Errorf("space.Plan = %v、期待値 = PlanFree", space.Plan)
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

		// 対象ユーザーがアクティブメンバーのスペース (含まれるべき)
		joinedSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("active-user-joined").
			WithName("Joined Space").
			Build()
		testutil.NewSpaceMemberBuilder(t, tx).
			WithSpaceID(joinedSpaceID).
			WithUserID(userID).
			Build()

		// 別ユーザーが所有するスペース (含まれないべき)
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
			t.Fatalf("ListActiveByUser()のエラー = %v", err)
		}
		if len(spaces) != 1 {
			t.Fatalf("len(spaces) = %d、期待値 = 1", len(spaces))
		}
		if spaces[0].ID != joinedSpaceID {
			t.Errorf("spaces[0].ID = %v、期待値 = %v", spaces[0].ID, joinedSpaceID)
		}
		if spaces[0].Identifier != "active-user-joined" {
			t.Errorf("spaces[0].Identifier = %v、期待値 = active-user-joined", spaces[0].Identifier)
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
			t.Fatalf("ListActiveByUser()のエラー = %v", err)
		}
		if len(spaces) != 1 {
			t.Fatalf("len(spaces) = %d、期待値 = 1", len(spaces))
		}
		if spaces[0].ID != liveSpaceID {
			t.Errorf("spaces[0].ID = %v、期待値 = %v", spaces[0].ID, liveSpaceID)
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

		// 退会メンバーのスペース (active = false)
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
			t.Fatalf("ListActiveByUser()のエラー = %v", err)
		}
		if len(spaces) != 1 {
			t.Fatalf("len(spaces) = %d、期待値 = 1", len(spaces))
		}
		if spaces[0].ID != activeSpaceID {
			t.Errorf("spaces[0].ID = %v、期待値 = %v", spaces[0].ID, activeSpaceID)
		}
	})

	t.Run("space_members.joined_atの降順で返す", func(t *testing.T) {
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
			t.Fatalf("ListActiveByUser()のエラー = %v", err)
		}
		if len(spaces) != 3 {
			t.Fatalf("len(spaces) = %d、期待値 = 3", len(spaces))
		}
		want := []model.SpaceID{newSpaceID, middleSpaceID, oldSpaceID}
		for i, expected := range want {
			if spaces[i].ID != expected {
				t.Errorf("spaces[%d].ID = %v、期待値 = %v", i, spaces[i].ID, expected)
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
			t.Fatalf("ListActiveByUser()のエラー = %v", err)
		}
		if len(spaces) != 0 {
			t.Errorf("len(spaces) = %d、期待値 = 0", len(spaces))
		}
	})
}

func TestSpaceRepository_ExistsByIdentifier(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewSpaceRepository(testutil.QueriesWithTx(tx))

	testutil.NewSpaceBuilder(t, tx).WithIdentifier("exists-by-identifier").Build()
	testutil.NewSpaceBuilder(t, tx).WithIdentifier("exists-discarded").WithDiscarded().Build()

	tests := []struct {
		name       string
		identifier model.SpaceIdentifier
		want       bool
	}{
		{name: "同じ識別子のスペースがあればtrue", identifier: "exists-by-identifier", want: true},
		{name: "大文字と小文字を区別しない", identifier: "EXISTS-By-Identifier", want: true},
		{name: "削除済みのスペースも数える", identifier: "exists-discarded", want: true},
		{name: "同じ識別子のスペースが無ければfalse", identifier: "exists-not-found", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.ExistsByIdentifier(context.Background(), tt.identifier)
			if err != nil {
				t.Fatalf("ExistsByIdentifier()のエラー = %v", err)
			}
			if got != tt.want {
				t.Errorf("ExistsByIdentifier(%q) = %v、期待値 = %v", tt.identifier, got, tt.want)
			}
		})
	}
}

func TestSpaceRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewSpaceRepository(testutil.QueriesWithTx(tx))

	space, err := repo.Create(context.Background(), CreateSpaceInput{
		Identifier: "repo-create-space",
		Name:       "作成したスペース",
		Plan:       model.PlanFree,
	})
	if err != nil {
		t.Fatalf("Create()のエラー = %v", err)
	}

	found, err := repo.FindByIdentifier(context.Background(), "repo-create-space")
	if err != nil {
		t.Fatalf("FindByIdentifier()のエラー = %v", err)
	}
	if found == nil {
		t.Fatal("作成したスペースが見つからない")
	}
	if found.ID != space.ID {
		t.Errorf("found.ID = %v、期待値 = %v", found.ID, space.ID)
	}
	if found.Name != "作成したスペース" {
		t.Errorf("found.Name = %q、期待値 = %q", found.Name, "作成したスペース")
	}
	if found.Plan != model.PlanFree {
		t.Errorf("found.Plan = %v、期待値 = PlanFree", found.Plan)
	}
	if found.JoinedAt.IsZero() {
		t.Error("found.JoinedAtがゼロ値")
	}
}

func TestSpaceRepository_ExistsOtherByIdentifier(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewSpaceRepository(testutil.QueriesWithTx(tx))

	selfID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("exists-other-self").Build()
	testutil.NewSpaceBuilder(t, tx).WithIdentifier("exists-other-taken").Build()
	testutil.NewSpaceBuilder(t, tx).WithIdentifier("exists-other-gone").WithDiscarded().Build()

	tests := []struct {
		name       string
		identifier model.SpaceIdentifier
		want       bool
	}{
		{name: "自分の識別子は数えない", identifier: "exists-other-self", want: false},
		{name: "自分の識別子の大文字と小文字だけを変えたものも数えない", identifier: "EXISTS-Other-Self", want: false},
		{name: "ほかのスペースの識別子は数える", identifier: "exists-other-taken", want: true},
		{name: "ほかのスペースの識別子は大文字と小文字を区別しない", identifier: "Exists-Other-Taken", want: true},
		{name: "削除済みのスペースも数える", identifier: "exists-other-gone", want: true},
		{name: "どのスペースも持たない識別子は数えない", identifier: "exists-other-free", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.ExistsOtherByIdentifier(context.Background(), selfID, tt.identifier)
			if err != nil {
				t.Fatalf("ExistsOtherByIdentifier()のエラー = %v", err)
			}
			if got != tt.want {
				t.Errorf("ExistsOtherByIdentifier(%q) = %v、期待値 = %v", tt.identifier, got, tt.want)
			}
		})
	}
}

func TestSpaceRepository_Update(t *testing.T) {
	t.Parallel()

	t.Run("識別子と名前を更新する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		repo := NewSpaceRepository(testutil.QueriesWithTx(tx))
		spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("repo-update-space").WithName("元の名前").Build()
		otherID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("repo-update-other").WithName("ほかの名前").Build()

		updated, err := repo.Update(context.Background(), UpdateSpaceInput{
			ID:         spaceID,
			Identifier: "repo-update-renamed",
			Name:       "新しい名前",
		})
		if err != nil {
			t.Fatalf("Update()のエラー = %v", err)
		}
		if updated == nil || updated.ID != spaceID || updated.Identifier != "repo-update-renamed" || updated.Name != "新しい名前" {
			t.Fatalf("Update()の戻り値 = %+v、更新したスペースを期待", updated)
		}

		other, err := repo.FindByID(context.Background(), otherID)
		if err != nil {
			t.Fatalf("FindByID()のエラー = %v", err)
		}
		if other.Identifier != "repo-update-other" || other.Name != "ほかの名前" {
			t.Errorf("ほかのスペース = (%q, %q)、変わっていないことを期待", other.Identifier, other.Name)
		}
	})

	t.Run("削除済みのスペースは更新せずnilを返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		repo := NewSpaceRepository(testutil.QueriesWithTx(tx))
		spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("repo-update-gone").WithDiscarded().Build()

		updated, err := repo.Update(context.Background(), UpdateSpaceInput{
			ID:         spaceID,
			Identifier: "repo-update-revived",
			Name:       "新しい名前",
		})
		if err != nil {
			t.Fatalf("Update()のエラー = %v", err)
		}
		if updated != nil {
			t.Errorf("Update()の戻り値 = %+v、nilを期待", updated)
		}
	})
}

func TestSpaceRepository_DiscardedSpace(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewSpaceRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	activeID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("discarded-space-active").Build()
	discardedID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("discarded-space-target").WithDiscarded().Build()
	const missingID model.SpaceID = "00000000-0000-0000-0000-000000000000"

	tests := []struct {
		name string
		id   model.SpaceID
		want bool
	}{
		{name: "論理削除したスペースはtrue", id: discardedID, want: true},
		{name: "論理削除していないスペースはfalse", id: activeID, want: false},
		{name: "行の無いスペースはfalse", id: missingID, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := repo.ExistsDiscarded(ctx, tt.id)
			if err != nil {
				t.Fatalf("ExistsDiscarded()のエラー = %v", err)
			}
			if exists != tt.want {
				t.Errorf("ExistsDiscarded() = %v、期待値 = %v", exists, tt.want)
			}
			locked, err := repo.LockDiscarded(ctx, tt.id)
			if err != nil {
				t.Fatalf("LockDiscarded()のエラー = %v", err)
			}
			if locked != tt.want {
				t.Errorf("LockDiscarded() = %v、期待値 = %v", locked, tt.want)
			}
		})
	}

	t.Run("論理削除していないスペースは物理削除しない", func(t *testing.T) {
		deleted, err := repo.DeleteDiscarded(ctx, activeID)
		if err != nil {
			t.Fatalf("DeleteDiscarded()のエラー = %v", err)
		}
		if deleted {
			t.Error("DeleteDiscarded() = true、期待値 = false")
		}
		space, err := repo.FindByID(ctx, activeID)
		if err != nil || space == nil {
			t.Fatalf("FindByID() = %v, %v、期待値 = スペースが残っている", space, err)
		}
	})

	t.Run("論理削除したスペースを物理削除する", func(t *testing.T) {
		deleted, err := repo.DeleteDiscarded(ctx, discardedID)
		if err != nil {
			t.Fatalf("DeleteDiscarded()のエラー = %v", err)
		}
		if !deleted {
			t.Error("DeleteDiscarded() = false、期待値 = true")
		}
		exists, err := repo.ExistsDiscarded(ctx, discardedID)
		if err != nil {
			t.Fatalf("ExistsDiscarded()のエラー = %v", err)
		}
		if exists {
			t.Error("物理削除したあとのExistsDiscarded() = true、期待値 = false")
		}
	})
}

func TestSpaceRepository_DiscardAndUndiscard(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewSpaceRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("discard-undiscard-space").Build()
	discardedAt := time.Now().Truncate(time.Microsecond)

	discarded, err := repo.Discard(ctx, spaceID, discardedAt)
	if err != nil {
		t.Fatalf("Discard()のエラー = %v", err)
	}
	if !discarded {
		t.Fatal("Discard() = false、期待値 = true")
	}
	if space, err := repo.FindByID(ctx, spaceID); err != nil || space != nil {
		t.Fatalf("FindByID() = %v, %v、論理削除したスペースは見えないことを期待", space, err)
	}

	t.Run("既に論理削除したスペースはfalse", func(t *testing.T) {
		again, err := repo.Discard(ctx, spaceID, discardedAt.Add(time.Second))
		if err != nil {
			t.Fatalf("Discard()のエラー = %v", err)
		}
		if again {
			t.Error("Discard() = true、期待値 = false")
		}
	})

	t.Run("違う時刻を指定した取り消しは戻さない", func(t *testing.T) {
		if err := repo.Undiscard(ctx, spaceID, discardedAt.Add(time.Second), time.Now()); err != nil {
			t.Fatalf("Undiscard()のエラー = %v", err)
		}
		if space, err := repo.FindByID(ctx, spaceID); err != nil || space != nil {
			t.Errorf("FindByID() = %v, %v、論理削除したままを期待", space, err)
		}
	})

	t.Run("同じ時刻を指定した取り消しは戻す", func(t *testing.T) {
		if err := repo.Undiscard(ctx, spaceID, discardedAt, time.Now()); err != nil {
			t.Fatalf("Undiscard()のエラー = %v", err)
		}
		if space, err := repo.FindByID(ctx, spaceID); err != nil || space == nil {
			t.Errorf("FindByID() = %v, %v、論理削除が取り消されることを期待", space, err)
		}
	})
}
