package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetPageMoveDataUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	pageRepo := repository.NewPageRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	uc := NewGetPageMoveDataUsecase(spaceRepo, spaceMemberRepo, pageRepo, topicRepo, topicMemberRepo)

	// テストデータを作成
	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("gpmd-owner@example.com").
		WithAtname("gpmdowner").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("gpmd-space").
		WithName("GPMD Space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		WithRole(0). // owner
		Build()
	topicID1 := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("トピック1").
		WithVisibility(0). // public
		Build()
	topicID2 := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(2).
		WithName("トピック2").
		WithVisibility(0). // public
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID1).
		WithSpaceMemberID(spaceMemberID).
		WithRole(0). // admin
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID2).
		WithSpaceMemberID(spaceMemberID).
		WithRole(0). // admin
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID1).
		WithNumber(1).
		WithTitle("テストページ").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	t.Run("存在しないスペースでnilが返る", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetPageMoveDataInput{
			SpaceIdentifier: "nonexistent",
			PageNumber:      1,
			UserID:          ownerID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output != nil {
			t.Error("output should be nil for non-existent space")
		}
	})

	t.Run("スペースメンバーでないユーザーでnilが返る", func(t *testing.T) {
		nonMemberID := testutil.NewUserBuilder(t, tx).
			WithEmail("gpmd-nonmember@example.com").
			WithAtname("gpmdnonmember").
			Build()
		output, err := uc.Execute(context.Background(), GetPageMoveDataInput{
			SpaceIdentifier: "gpmd-space",
			PageNumber:      1,
			UserID:          nonMemberID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output != nil {
			t.Error("output should be nil for non-member user")
		}
	})

	t.Run("存在しないページでnilが返る", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetPageMoveDataInput{
			SpaceIdentifier: "gpmd-space",
			PageNumber:      999,
			UserID:          ownerID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output != nil {
			t.Error("output should be nil for non-existent page")
		}
	})

	t.Run("正常系: すべてのデータが取得できる", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetPageMoveDataInput{
			SpaceIdentifier: "gpmd-space",
			PageNumber:      1,
			UserID:          ownerID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.Space.Name != "GPMD Space" {
			t.Errorf("Space.Name = %q, want %q", output.Space.Name, "GPMD Space")
		}
		if output.SpaceMember == nil {
			t.Fatal("SpaceMember should not be nil")
		}
		if output.Page == nil {
			t.Fatal("Page should not be nil")
		}
		if output.TopicMember == nil {
			t.Fatal("TopicMember should not be nil")
		}
		if output.CurrentTopic == nil {
			t.Fatal("CurrentTopic should not be nil")
		}
		if output.CurrentTopic.Name != "トピック1" {
			t.Errorf("CurrentTopic.Name = %q, want %q", output.CurrentTopic.Name, "トピック1")
		}
		// AvailableTopicsは現在のトピックを除外するので、トピック2のみ
		if len(output.AvailableTopics) != 1 {
			t.Fatalf("AvailableTopics count = %d, want 1", len(output.AvailableTopics))
		}
		if output.AvailableTopics[0].Name != "トピック2" {
			t.Errorf("AvailableTopics[0].Name = %q, want %q", output.AvailableTopics[0].Name, "トピック2")
		}
	})

	t.Run("トピックメンバーでないユーザーでもデータが返る", func(t *testing.T) {
		// メンバーだがトピックメンバーではないユーザーを作成
		memberID := testutil.NewUserBuilder(t, tx).
			WithEmail("gpmd-member@example.com").
			WithAtname("gpmdmember").
			Build()
		testutil.NewSpaceMemberBuilder(t, tx).
			WithSpaceID(spaceID).
			WithUserID(memberID).
			WithRole(1). // member（ownerではない）
			Build()

		output, err := uc.Execute(context.Background(), GetPageMoveDataInput{
			SpaceIdentifier: "gpmd-space",
			PageNumber:      1,
			UserID:          memberID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.TopicMember != nil {
			t.Error("TopicMember should be nil for non-topic-member user")
		}
	})
}
