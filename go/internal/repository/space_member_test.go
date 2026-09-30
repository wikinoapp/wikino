package repository

import (
	"context"
	"slices"
	"testing"

	"github.com/lib/pq"

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
			t.Fatalf("FindActiveBySpaceAndUser()のエラー = %v", err)
		}
		if member == nil {
			t.Fatal("FindActiveBySpaceAndUser()がnilを返した、期待値 = メンバー")
		}
		if member.ID != spaceMemberID {
			t.Errorf("member.ID = %v、期待値 = %v", member.ID, spaceMemberID)
		}
		if member.SpaceID != spaceID {
			t.Errorf("member.SpaceID = %v、期待値 = %v", member.SpaceID, spaceID)
		}
		if member.UserID != model.UserID(userID) {
			t.Errorf("member.UserID = %v、期待値 = %v", member.UserID, userID)
		}
		if member.Role != model.SpaceRoleAdmin {
			t.Errorf("member.Role = %q、期待値 = %q", member.Role, model.SpaceRoleAdmin)
		}
		if !member.Active {
			t.Error("member.Active = false、期待値 = true")
		}
	})

	t.Run("存在しないスペースIDはnilを返す", func(t *testing.T) {
		member, err := repo.FindActiveBySpaceAndUser(context.Background(), "00000000-0000-0000-0000-000000000000", model.UserID(userID))
		if err != nil {
			t.Fatalf("FindActiveBySpaceAndUser()のエラー = %v", err)
		}
		if member != nil {
			t.Errorf("FindActiveBySpaceAndUser() = %v、期待値 = nil", member)
		}
	})

	t.Run("存在しないユーザーIDはnilを返す", func(t *testing.T) {
		member, err := repo.FindActiveBySpaceAndUser(context.Background(), spaceID, model.UserID("00000000-0000-0000-0000-000000000000"))
		if err != nil {
			t.Fatalf("FindActiveBySpaceAndUser()のエラー = %v", err)
		}
		if member != nil {
			t.Errorf("FindActiveBySpaceAndUser() = %v、期待値 = nil", member)
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
			t.Fatalf("FindActiveBySpaceAndUser()のエラー = %v", err)
		}
		if member != nil {
			t.Errorf("FindActiveBySpaceAndUser() = %v、期待値 = nil", member)
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

	// 対象ユーザーがアクティブに参加する2スペース。
	spaceID1 := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("list-active-space-1").
		WithName("List Active Space 1").
		Build()
	spaceID2 := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("list-active-space-2").
		WithName("List Active Space 2").
		Build()
	// 対象ユーザーが非アクティブに参加するスペース。
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
	// 非アクティブなメンバー (結果に含まれない想定)。
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID3).
		WithUserID(model.UserID(userID)).
		WithActive(false).
		Build()
	// 別ユーザーのメンバー (結果に含まれない想定)。
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
			t.Fatalf("ListActiveByUserAndSpaceIDs()のエラー = %v", err)
		}

		// spaceID1 / spaceID2のアクティブメンバーのみが返り、非アクティブ (spaceID3) と
		// 別ユーザーのメンバーは含まれない。
		gotIDs := make(map[model.SpaceMemberID]bool, len(members))
		for _, m := range members {
			gotIDs[m.ID] = true
			if m.UserID != model.UserID(userID) {
				t.Errorf("member.UserID = %v、期待値 = %v", m.UserID, userID)
			}
			if !m.Active {
				t.Errorf("非アクティブなメンバー%vが含まれている", m.ID)
			}
		}
		if len(members) != 2 {
			t.Fatalf("len(members) = %d、期待値 = 2", len(members))
		}
		if !gotIDs[memberID1] || !gotIDs[memberID2] {
			t.Errorf("members = %v、期待値 = %vと%vを含む", gotIDs, memberID1, memberID2)
		}
	})

	t.Run("指定したスペースIDに含まれないメンバーは返さない", func(t *testing.T) {
		members, err := repo.ListActiveByUserAndSpaceIDs(
			context.Background(),
			model.UserID(userID),
			[]model.SpaceID{spaceID1},
		)
		if err != nil {
			t.Fatalf("ListActiveByUserAndSpaceIDs()のエラー = %v", err)
		}
		if len(members) != 1 {
			t.Fatalf("len(members) = %d、期待値 = 1", len(members))
		}
		if members[0].ID != memberID1 {
			t.Errorf("members[0].ID = %v、期待値 = %v", members[0].ID, memberID1)
		}
	})

	t.Run("空のスペースIDリストはnilを返す", func(t *testing.T) {
		members, err := repo.ListActiveByUserAndSpaceIDs(
			context.Background(),
			model.UserID(userID),
			[]model.SpaceID{},
		)
		if err != nil {
			t.Fatalf("ListActiveByUserAndSpaceIDs()のエラー = %v", err)
		}
		if members != nil {
			t.Errorf("ListActiveByUserAndSpaceIDs() = %v、期待値 = nil", members)
		}
	})
}

func TestSpaceMemberRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewSpaceMemberRepository(testutil.QueriesWithTx(tx))

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("spacemember-create@example.com").
		WithAtname("spacemember_create").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("member-create-space").Build()

	created, err := repo.Create(context.Background(), CreateSpaceMemberInput{
		SpaceID: spaceID,
		UserID:  model.UserID(userID),
		Role:    model.SpaceRoleAdmin,
	})
	if err != nil {
		t.Fatalf("Create()のエラー = %v", err)
	}

	member, err := repo.FindActiveBySpaceAndUser(context.Background(), spaceID, model.UserID(userID))
	if err != nil {
		t.Fatalf("FindActiveBySpaceAndUser()のエラー = %v", err)
	}
	if member == nil {
		t.Fatal("作成したスペースメンバーがアクティブなメンバーとして見つからない")
	}
	if member.ID != created.ID {
		t.Errorf("member.ID = %v、期待値 = %v", member.ID, created.ID)
	}
	if member.Role != model.SpaceRoleAdmin {
		t.Errorf("member.Role = %q、期待値 = %q", member.Role, model.SpaceRoleAdmin)
	}
	// Rails版が判定に使うスコープも書かれていること
	var scopes []string
	if err := tx.QueryRowContext(context.Background(),
		`SELECT scopes FROM space_members WHERE id = $1 AND space_id = $2`,
		string(created.ID), string(spaceID),
	).Scan(pq.Array(&scopes)); err != nil {
		t.Fatalf("scopesの取得に失敗: %v", err)
	}
	if want := []string{"space:admin"}; !slices.Equal(scopes, want) {
		t.Errorf("scopes = %v、期待値 = %v", scopes, want)
	}
	if member.JoinedAt.IsZero() {
		t.Error("member.JoinedAtがゼロ値")
	}
}

func TestSpaceMemberRepository_Create_EditorWritesRoleScopesForRails(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewSpaceMemberRepository(testutil.QueriesWithTx(tx))

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("spacemember-create-editor@example.com").
		WithAtname("spacemember_editor").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("member-create-editor").Build()

	created, err := repo.Create(context.Background(), CreateSpaceMemberInput{
		SpaceID: spaceID,
		UserID:  model.UserID(userID),
		Role:    model.SpaceRoleEditor,
	})
	if err != nil {
		t.Fatalf("Create()のエラー = %v", err)
	}
	if created.Role != model.SpaceRoleEditor {
		t.Errorf("created.Role = %q、期待値 = %q", created.Role, model.SpaceRoleEditor)
	}

	var scopes []string
	if err := tx.QueryRowContext(context.Background(),
		`SELECT scopes FROM space_members WHERE id = $1 AND space_id = $2`,
		string(created.ID), string(spaceID),
	).Scan(pq.Array(&scopes)); err != nil {
		t.Fatalf("scopesの取得に失敗: %v", err)
	}
	if want := model.ScopesToStrings(model.SpaceRoleEditor.Scopes()); !slices.Equal(scopes, want) {
		t.Errorf("scopes = %v、期待値 = %v", scopes, want)
	}
}
