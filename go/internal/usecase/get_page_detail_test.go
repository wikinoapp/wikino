package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetPageDetailUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	pageRepo := repository.NewPageRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	uc := NewGetPageDetailUsecase(spaceRepo, spaceMemberRepo, pageRepo, draftPageRepo, topicRepo, topicMemberRepo)

	// テストデータを作成
	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("gpd-owner@example.com").
		WithAtname("gpdowner").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("gpd-space").
		WithName("GPD Space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		WithRole(0). // owner
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("テストトピック").
		WithVisibility(0). // public
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		WithRole(0). // admin
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("テストページ").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	t.Run("存在しないスペースでnilが返る", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetPageDetailInput{
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
			WithEmail("gpd-nonmember@example.com").
			WithAtname("gpdnonmember").
			Build()
		output, err := uc.Execute(context.Background(), GetPageDetailInput{
			SpaceIdentifier: "gpd-space",
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
		output, err := uc.Execute(context.Background(), GetPageDetailInput{
			SpaceIdentifier: "gpd-space",
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
		output, err := uc.Execute(context.Background(), GetPageDetailInput{
			SpaceIdentifier: "gpd-space",
			PageNumber:      1,
			UserID:          ownerID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.Space.Name != "GPD Space" {
			t.Errorf("Space.Name = %q, want %q", output.Space.Name, "GPD Space")
		}
		if output.SpaceMember == nil {
			t.Fatal("SpaceMember should not be nil")
		}
		if output.Page == nil {
			t.Fatal("Page should not be nil")
		}
		if output.Topic == nil {
			t.Fatal("Topic should not be nil")
		}
		if output.Topic.Name != "テストトピック" {
			t.Errorf("Topic.Name = %q, want %q", output.Topic.Name, "テストトピック")
		}
		if output.TopicMember == nil {
			t.Fatal("TopicMember should not be nil")
		}
		if output.DraftPage != nil {
			t.Error("DraftPage should be nil when no draft exists")
		}
	})

	t.Run("正常系: DraftPageが存在する場合も取得できる", func(t *testing.T) {
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(2).
			WithTitle("下書きありページ").
			WithLinkedPageIDs([]model.PageID{}).
			Build()
		testutil.NewDraftPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithPageID(pageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithTitle("下書きタイトル").
			Build()

		output, err := uc.Execute(context.Background(), GetPageDetailInput{
			SpaceIdentifier: "gpd-space",
			PageNumber:      2,
			UserID:          ownerID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.DraftPage == nil {
			t.Fatal("DraftPage should not be nil when draft exists")
		}
	})
}
