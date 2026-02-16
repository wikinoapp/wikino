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
		WithRole(0). // owner
		WithActive(true).
		Build()

	t.Run("アクティブなスペースメンバーを取得できる", func(t *testing.T) {
		member, err := repo.FindActiveBySpaceAndUser(context.Background(), spaceID, userID)
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
		if member.UserID != userID {
			t.Errorf("member.UserID = %v, want %v", member.UserID, userID)
		}
		if member.Role != model.SpaceMemberRoleOwner {
			t.Errorf("member.Role = %v, want SpaceMemberRoleOwner", member.Role)
		}
		if !member.Active {
			t.Error("member.Active = false, want true")
		}
	})

	t.Run("存在しないスペースIDはnilを返す", func(t *testing.T) {
		member, err := repo.FindActiveBySpaceAndUser(context.Background(), "00000000-0000-0000-0000-000000000000", userID)
		if err != nil {
			t.Fatalf("FindActiveBySpaceAndUser() error = %v", err)
		}
		if member != nil {
			t.Errorf("FindActiveBySpaceAndUser() = %v, want nil", member)
		}
	})

	t.Run("存在しないユーザーIDはnilを返す", func(t *testing.T) {
		member, err := repo.FindActiveBySpaceAndUser(context.Background(), spaceID, "00000000-0000-0000-0000-000000000000")
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
			WithRole(1). // member
			WithActive(false).
			Build()

		member, err := repo.FindActiveBySpaceAndUser(context.Background(), spaceID, inactiveUserID)
		if err != nil {
			t.Fatalf("FindActiveBySpaceAndUser() error = %v", err)
		}
		if member != nil {
			t.Errorf("FindActiveBySpaceAndUser() = %v, want nil", member)
		}
	})
}
