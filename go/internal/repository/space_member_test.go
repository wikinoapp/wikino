package repository

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestSpaceMemberRepository_FindActiveBySpaceAndUser(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewSpaceMemberRepository(q)

	// テストデータを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("spacemember@example.com").
		WithAtname("spacemember").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("member-test-space").
		WithName("Member Test Space").
		Build()

	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithActive(true).
		Build()

	t.Run("アクティブなスペースメンバーを取得できる", func(t *testing.T) {
		member, err := repo.FindActiveBySpaceAndUser(context.Background(), spaceID, model.UserID(userID))
		if err != nil {
			t.Fatalf("FindActiveBySpaceAndUser() error = %v", err)
		}
		if member == nil {
			t.Fatal("FindActiveBySpaceAndUser() returned nil, want member")
		}
		if member.ID != spaceMemberID {
			t.Errorf("member.ID = %v, want %v", member.ID, spaceMemberID)
		}
		if member.SpaceID != spaceID {
			t.Errorf("member.SpaceID = %v, want %v", member.SpaceID, spaceID)
		}
		if member.UserID != model.UserID(userID) {
			t.Errorf("member.UserID = %v, want %v", member.UserID, userID)
		}
		if len(member.Scopes) != 1 || member.Scopes[0] != model.ScopeSpaceAdmin {
			t.Errorf("member.Scopes = %v, want [%v]", member.Scopes, model.ScopeSpaceAdmin)
		}
		if !member.Active {
			t.Error("member.Active = false, want true")
		}
	})

	t.Run("存在しないスペースIDはnilを返す", func(t *testing.T) {
		member, err := repo.FindActiveBySpaceAndUser(context.Background(), "00000000-0000-0000-0000-000000000000", model.UserID(userID))
		if err != nil {
			t.Fatalf("FindActiveBySpaceAndUser() error = %v", err)
		}
		if member != nil {
			t.Errorf("FindActiveBySpaceAndUser() = %v, want nil", member)
		}
	})

	t.Run("存在しないユーザーIDはnilを返す", func(t *testing.T) {
		member, err := repo.FindActiveBySpaceAndUser(context.Background(), spaceID, model.UserID("00000000-0000-0000-0000-000000000000"))
		if err != nil {
			t.Fatalf("FindActiveBySpaceAndUser() error = %v", err)
		}
		if member != nil {
			t.Errorf("FindActiveBySpaceAndUser() = %v, want nil", member)
		}
	})

	t.Run("非アクティブなスペースメンバーはnilを返す", func(t *testing.T) {
		// 非アクティブなメンバーを作成
		inactiveUserID := testutil.NewUserBuilder(t, tx).
			WithEmail("inactive@example.com").
			WithAtname("inactive").
			Build()

		testutil.NewSpaceMemberBuilder(t, tx).
			WithSpaceID(spaceID).
			WithUserID(inactiveUserID).
			WithActive(false).
			Build()

		member, err := repo.FindActiveBySpaceAndUser(context.Background(), spaceID, model.UserID(inactiveUserID))
		if err != nil {
			t.Fatalf("FindActiveBySpaceAndUser() error = %v", err)
		}
		if member != nil {
			t.Errorf("FindActiveBySpaceAndUser() = %v, want nil", member)
		}
	})
}

func TestSpaceMemberRepository_ListActiveByUserAndSpaceIDs(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewSpaceMemberRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("list-active-user@example.com").
		WithAtname("listactiveuser").
		Build()
	otherUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("list-active-other@example.com").
		WithAtname("listactiveother").
		Build()

	// Two spaces the target user actively belongs to.
	// [Ja] 対象ユーザーがアクティブに参加する 2 スペース。
	spaceID1 := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("list-active-space-1").
		WithName("List Active Space 1").
		Build()
	spaceID2 := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("list-active-space-2").
		WithName("List Active Space 2").
		Build()
	// A space the target user belongs to but is inactive in.
	// [Ja] 対象ユーザーが非アクティブに参加するスペース。
	spaceID3 := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("list-active-space-3").
		WithName("List Active Space 3").
		Build()

	memberID1 := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID1).
		WithUserID(model.UserID(userID)).
		WithActive(true).
		Build()
	memberID2 := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID2).
		WithUserID(model.UserID(userID)).
		WithActive(true).
		Build()
	// Inactive member: expected to be excluded from the result.
	// [Ja] 非アクティブなメンバー (結果に含まれない想定)。
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID3).
		WithUserID(model.UserID(userID)).
		WithActive(false).
		Build()
	// Another user's member: expected to be excluded from the result.
	// [Ja] 別ユーザーのメンバー (結果に含まれない想定)。
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID1).
		WithUserID(model.UserID(otherUserID)).
		WithActive(true).
		Build()

	t.Run("複数スペースにまたがるアクティブなメンバーを一括取得できる", func(t *testing.T) {
		members, err := repo.ListActiveByUserAndSpaceIDs(
			context.Background(),
			model.UserID(userID),
			[]model.SpaceID{spaceID1, spaceID2, spaceID3},
		)
		if err != nil {
			t.Fatalf("ListActiveByUserAndSpaceIDs() error = %v", err)
		}

		// Only the active members of spaceID1 / spaceID2 are returned; the inactive one
		// (spaceID3) and the other user's member are excluded.
		//
		// [Ja] spaceID1 / spaceID2 のアクティブメンバーのみが返り、非アクティブ (spaceID3) と
		// 別ユーザーのメンバーは含まれない。
		gotIDs := make(map[model.SpaceMemberID]bool, len(members))
		for _, m := range members {
			gotIDs[m.ID] = true
			if m.UserID != model.UserID(userID) {
				t.Errorf("member.UserID = %v, want %v", m.UserID, userID)
			}
			if !m.Active {
				t.Errorf("member %v is inactive, want active only", m.ID)
			}
		}
		if len(members) != 2 {
			t.Fatalf("len(members) = %d, want 2", len(members))
		}
		if !gotIDs[memberID1] || !gotIDs[memberID2] {
			t.Errorf("members = %v, want to contain %v and %v", gotIDs, memberID1, memberID2)
		}
	})

	t.Run("指定したスペースIDに含まれないメンバーは返さない", func(t *testing.T) {
		members, err := repo.ListActiveByUserAndSpaceIDs(
			context.Background(),
			model.UserID(userID),
			[]model.SpaceID{spaceID1},
		)
		if err != nil {
			t.Fatalf("ListActiveByUserAndSpaceIDs() error = %v", err)
		}
		if len(members) != 1 {
			t.Fatalf("len(members) = %d, want 1", len(members))
		}
		if members[0].ID != memberID1 {
			t.Errorf("members[0].ID = %v, want %v", members[0].ID, memberID1)
		}
	})

	t.Run("空のスペースIDリストはnilを返す", func(t *testing.T) {
		members, err := repo.ListActiveByUserAndSpaceIDs(
			context.Background(),
			model.UserID(userID),
			[]model.SpaceID{},
		)
		if err != nil {
			t.Fatalf("ListActiveByUserAndSpaceIDs() error = %v", err)
		}
		if members != nil {
			t.Errorf("ListActiveByUserAndSpaceIDs() = %v, want nil", members)
		}
	})
}
