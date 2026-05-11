package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetHomeShowUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	spaceRepo := repository.NewSpaceRepository(q)
	uc := NewGetHomeShowUsecase(spaceRepo)

	t.Run("参加中スペースが0件の場合は空のスライスが返る", func(t *testing.T) {
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("ghs-empty@example.com").
			WithAtname("ghsempty").
			Build()

		output, err := uc.Execute(context.Background(), GetHomeShowInput{
			UserID: userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if len(output.ActiveSpaces) != 0 {
			t.Errorf("len(ActiveSpaces) = %d, want 0", len(output.ActiveSpaces))
		}
	})

	t.Run("参加中スペースが複数件ある場合は一覧が返る", func(t *testing.T) {
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("ghs-multi@example.com").
			WithAtname("ghsmulti").
			Build()

		firstSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("ghs-space-1").
			WithName("GHS Space 1").
			Build()
		testutil.NewSpaceMemberBuilder(t, tx).
			WithSpaceID(firstSpaceID).
			WithUserID(userID).
			Build()

		secondSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("ghs-space-2").
			WithName("GHS Space 2").
			Build()
		testutil.NewSpaceMemberBuilder(t, tx).
			WithSpaceID(secondSpaceID).
			WithUserID(userID).
			Build()

		output, err := uc.Execute(context.Background(), GetHomeShowInput{
			UserID: userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if len(output.ActiveSpaces) != 2 {
			t.Fatalf("len(ActiveSpaces) = %d, want 2", len(output.ActiveSpaces))
		}

		gotIDs := map[string]bool{
			string(output.ActiveSpaces[0].ID): true,
			string(output.ActiveSpaces[1].ID): true,
		}
		if !gotIDs[string(firstSpaceID)] || !gotIDs[string(secondSpaceID)] {
			t.Errorf("ActiveSpaces IDs = %v, want %v and %v", gotIDs, firstSpaceID, secondSpaceID)
		}
	})
}
