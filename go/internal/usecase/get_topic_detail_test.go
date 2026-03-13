package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetTopicDetailUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	pageRepo := repository.NewPageRepository(q)
	uc := NewGetTopicDetailUsecase(spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, pageRepo)

	// テストデータを作成
	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("gtd-owner@example.com").
		WithAtname("gtdowner").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("gtd-space").
		WithName("GTD Space").
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
		output, err := uc.Execute(context.Background(), GetTopicDetailInput{
			SpaceIdentifier: "nonexistent",
			TopicNumber:     1,
			Page:            1,
			PageLimit:       100,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output != nil {
			t.Error("output should be nil for non-existent space")
		}
	})

	t.Run("存在しないトピックでnilが返る", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetTopicDetailInput{
			SpaceIdentifier: "gtd-space",
			TopicNumber:     999,
			Page:            1,
			PageLimit:       100,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output != nil {
			t.Error("output should be nil for non-existent topic")
		}
	})

	t.Run("公開トピックを未ログインで取得できる", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetTopicDetailInput{
			SpaceIdentifier: "gtd-space",
			TopicNumber:     1,
			Page:            1,
			PageLimit:       100,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.Space.Name != "GTD Space" {
			t.Errorf("Space.Name = %q, want %q", output.Space.Name, "GTD Space")
		}
		if output.Topic.Name != "テストトピック" {
			t.Errorf("Topic.Name = %q, want %q", output.Topic.Name, "テストトピック")
		}
		if output.SpaceMember != nil {
			t.Error("SpaceMember should be nil for unauthenticated user")
		}
		if len(output.Pages) != 1 {
			t.Errorf("len(Pages) = %d, want 1", len(output.Pages))
		}
	})

	t.Run("ログインユーザーでスペースメンバーとトピックメンバーが取得できる", func(t *testing.T) {
		userID := ownerID
		output, err := uc.Execute(context.Background(), GetTopicDetailInput{
			SpaceIdentifier: "gtd-space",
			TopicNumber:     1,
			UserID:          &userID,
			Page:            1,
			PageLimit:       100,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.SpaceMember == nil {
			t.Fatal("SpaceMember should not be nil for authenticated user")
		}
		if output.SpaceMember.Role != model.SpaceMemberRoleOwner {
			t.Errorf("SpaceMember.Role = %d, want %d", output.SpaceMember.Role, model.SpaceMemberRoleOwner)
		}
		if output.TopicMember == nil {
			t.Fatal("TopicMember should not be nil for topic member")
		}
	})
}

func TestGetTopicDetailUsecase_Execute_非公開トピック(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	pageRepo := repository.NewPageRepository(q)
	uc := NewGetTopicDetailUsecase(spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, pageRepo)

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("gtd-priv-owner@example.com").
		WithAtname("gtdprivowner").
		Build()
	nonMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("gtd-priv-nonmember@example.com").
		WithAtname("gtdprivnonmember").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("gtd-priv").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		WithRole(0). // owner
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(nonMemberID).
		WithRole(1). // member
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("非公開トピック").
		WithVisibility(1). // private
		Build()

	t.Run("未ログインでnilが返る", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetTopicDetailInput{
			SpaceIdentifier: "gtd-priv",
			TopicNumber:     1,
			Page:            1,
			PageLimit:       100,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output != nil {
			t.Error("output should be nil for unauthenticated user on private topic")
		}
	})

	t.Run("スペースオーナーは閲覧できる", func(t *testing.T) {
		userID := ownerID
		output, err := uc.Execute(context.Background(), GetTopicDetailInput{
			SpaceIdentifier: "gtd-priv",
			TopicNumber:     1,
			UserID:          &userID,
			Page:            1,
			PageLimit:       100,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil for space owner")
		}
	})

	t.Run("非メンバーでnilが返る", func(t *testing.T) {
		userID := nonMemberID
		output, err := uc.Execute(context.Background(), GetTopicDetailInput{
			SpaceIdentifier: "gtd-priv",
			TopicNumber:     1,
			UserID:          &userID,
			Page:            1,
			PageLimit:       100,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output != nil {
			t.Error("output should be nil for non-member on private topic")
		}
	})
}
