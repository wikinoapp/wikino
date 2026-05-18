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
	topicRepo := repository.NewTopicRepository(q)
	uc := NewGetHomeShowUsecase(spaceRepo, topicRepo)

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
		// Published page so PublishedPagesCount > 0.
		// [Ja] PublishedPagesCount が 1 以上になるように公開ページを 1 件用意する。
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicAID).
			WithNumber(1).
			WithTitle("A-published").
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

		gotByName := map[string]repository.JoinedTopicWithStats{}
		for _, s := range output.JoinedTopics {
			gotByName[s.Topic.Name] = s
		}
		a, ok := gotByName["Topic A"]
		if !ok {
			t.Fatal("Topic A not found in JoinedTopics")
		}
		if a.PublishedPagesCount != 1 {
			t.Errorf("Topic A PublishedPagesCount = %d, want 1", a.PublishedPagesCount)
		}
		if a.Topic.Space.ID != spaceID {
			t.Errorf("Topic A Space.ID = %v, want %v", a.Topic.Space.ID, spaceID)
		}
		if _, ok := gotByName["Topic B"]; !ok {
			t.Fatal("Topic B not found in JoinedTopics")
		}
	})
}
