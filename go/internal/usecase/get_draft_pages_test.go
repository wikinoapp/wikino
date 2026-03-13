package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetDraftPagesUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	draftPageRepo := repository.NewDraftPageRepository(q)
	uc := NewGetDraftPagesUsecase(draftPageRepo)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("ldp-owner@example.com").
		WithAtname("ldpowner").
		Build()

	t.Run("下書きがない場合は空のスライスが返る", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetDraftPagesInput{
			UserID: userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if len(output.DraftPages) != 0 {
			t.Errorf("len(DraftPages) = %d, want 0", len(output.DraftPages))
		}
	})

	t.Run("下書きがある場合はリストが返る", func(t *testing.T) {
		spaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("ldp-space").
			WithName("LDP Space").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(1).
			WithName("LDPトピック").
			Build()
		testutil.NewTopicMemberBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithSpaceMemberID(spaceMemberID).
			Build()
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("LDPページ").
			WithLinkedPageIDs([]model.PageID{}).
			Build()
		testutil.NewDraftPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithPageID(pageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithTitle("下書きタイトル").
			WithBody("下書き本文").
			Build()

		output, err := uc.Execute(context.Background(), GetDraftPagesInput{
			UserID: userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if len(output.DraftPages) != 1 {
			t.Errorf("len(DraftPages) = %d, want 1", len(output.DraftPages))
		}
	})
}
