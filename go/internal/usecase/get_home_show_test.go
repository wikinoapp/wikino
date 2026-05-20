package usecase

import (
	"context"
	"fmt"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetHomeShowUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	spaceRepo := repository.NewSpaceRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	uc := NewGetHomeShowUsecase(spaceRepo, topicRepo, draftPageRepo)

	t.Run("参加中スペース・トピックが0件の場合は空のスライスが返る", func(t *testing.T) {
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
		if len(output.JoinedTopics) != 0 {
			t.Errorf("len(JoinedTopics) = %d, want 0", len(output.JoinedTopics))
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

	t.Run("参加中トピックが複数件ある場合は一覧が返る", func(t *testing.T) {
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("ghs-topics@example.com").
			WithAtname("ghstopics").
			Build()

		spaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("ghs-topics-space").
			WithName("GHS Topics Space").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()

		topicAID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(1).
			WithName("Topic A").
			Build()
		testutil.NewTopicMemberBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicAID).
			WithSpaceMemberID(spaceMemberID).
			Build()

		topicBID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(2).
			WithName("Topic B").
			Build()
		testutil.NewTopicMemberBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicBID).
			WithSpaceMemberID(spaceMemberID).
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
		if len(output.JoinedTopics) != 2 {
			t.Fatalf("len(JoinedTopics) = %d, want 2", len(output.JoinedTopics))
		}

		gotByName := map[string]*model.Topic{}
		for _, topic := range output.JoinedTopics {
			gotByName[topic.Name] = topic
		}
		a, ok := gotByName["Topic A"]
		if !ok {
			t.Fatal("Topic A not found in JoinedTopics")
		}
		if a.Space.ID != spaceID {
			t.Errorf("Topic A Space.ID = %v, want %v", a.Space.ID, spaceID)
		}
		if _, ok := gotByName["Topic B"]; !ok {
			t.Fatal("Topic B not found in JoinedTopics")
		}
	})

	t.Run("下書きが0件のときは空スライスを返す", func(t *testing.T) {
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("ghs-drafts-empty@example.com").
			WithAtname("ghsdraftsempty").
			Build()

		output, err := uc.Execute(context.Background(), GetHomeShowInput{
			UserID: userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if len(output.DraftPages) != 0 {
			t.Errorf("len(DraftPages) = %d, want 0", len(output.DraftPages))
		}
	})

	t.Run("下書きが上限を超えるとhomeDraftPagesLimit件に制限される", func(t *testing.T) {
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("ghs-drafts-many@example.com").
			WithAtname("ghsdraftsmany").
			Build()

		spaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("ghs-drafts-many-space").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(1).
			WithName("General").
			Build()

		// Create 7 drafts (> homeDraftPagesLimit which is 5).
		// [Ja] 7 件作成 (上限 5 を超える)
		for i := int32(1); i <= 7; i++ {
			pageID := testutil.NewPageBuilder(t, tx).
				WithSpaceID(spaceID).
				WithTopicID(topicID).
				WithNumber(model.PageNumber(i)).
				WithTitle(fmt.Sprintf("Page %d", i)).
				Build()
			testutil.NewDraftPageBuilder(t, tx).
				WithSpaceID(spaceID).
				WithPageID(pageID).
				WithSpaceMemberID(spaceMemberID).
				WithTopicID(topicID).
				Build()
		}

		output, err := uc.Execute(context.Background(), GetHomeShowInput{
			UserID: userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if len(output.DraftPages) != 5 {
			t.Errorf("len(DraftPages) = %d, want 5 (capped at homeDraftPagesLimit)", len(output.DraftPages))
		}
	})
}
