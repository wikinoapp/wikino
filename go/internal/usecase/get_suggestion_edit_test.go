package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetSuggestionEditUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	suggestionRepo := repository.NewSuggestionRepository(q)
	userRepo := repository.NewUserRepository(q)
	uc := NewGetSuggestionEditUsecase(spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, suggestionRepo, userRepo)

	// テストデータのセットアップ
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("sug-edit-uc@example.com").
		WithAtname("sugedituc").
		WithName("編集太郎").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sug-edit-uc-space").
		WithName("SugEdit Space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("編集トピック").
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()

	testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithTitle("編集テスト提案").
		WithBody("提案の本文").
		WithBodyHTML("<p>提案の本文</p>").
		WithStatus(model.SuggestionStatusOpen).
		Build()

	t.Run("編集提案の編集データが取得できる", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetSuggestionEditInput{
			SpaceIdentifier:  "sug-edit-uc-space",
			SuggestionNumber: 1,
			UserID:           userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.Suggestion == nil {
			t.Fatal("Suggestion should not be nil")
		}
		if output.Suggestion.Title != "編集テスト提案" {
			t.Errorf("Suggestion.Title = %q, want %q", output.Suggestion.Title, "編集テスト提案")
		}
		if output.Space == nil {
			t.Fatal("Space should not be nil")
		}
		if output.Topic == nil {
			t.Fatal("Topic should not be nil")
		}
		if output.Topic.Name != "編集トピック" {
			t.Errorf("Topic.Name = %q, want %q", output.Topic.Name, "編集トピック")
		}
	})

	t.Run("作成者のユーザー情報がUserMapに含まれる", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetSuggestionEditInput{
			SpaceIdentifier:  "sug-edit-uc-space",
			SuggestionNumber: 1,
			UserID:           userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		user, ok := output.UserMap[spaceMemberID]
		if !ok {
			t.Fatal("UserMap should contain the creator's user info")
		}
		if user.Name != "編集太郎" {
			t.Errorf("user.Name = %q, want %q", user.Name, "編集太郎")
		}
	})

	t.Run("スペースオーナーは編集提案の更新権限を持つ", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetSuggestionEditInput{
			SpaceIdentifier:  "sug-edit-uc-space",
			SuggestionNumber: 1,
			UserID:           userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if !output.CanUpdateSuggestion {
			t.Error("CanUpdateSuggestion should be true for space owner")
		}
		if !output.CanUpdateSuggestionComment {
			t.Error("CanUpdateSuggestionComment should be true for space owner")
		}
	})

	t.Run("存在しないスペースの場合はnilが返る", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetSuggestionEditInput{
			SpaceIdentifier:  "nonexistent",
			SuggestionNumber: 1,
			UserID:           userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output != nil {
			t.Error("output should be nil for nonexistent space")
		}
	})

	t.Run("存在しない編集提案番号の場合はnilが返る", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetSuggestionEditInput{
			SpaceIdentifier:  "sug-edit-uc-space",
			SuggestionNumber: 999,
			UserID:           userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output != nil {
			t.Error("output should be nil for nonexistent suggestion number")
		}
	})

	t.Run("スペースメンバーでないユーザーの場合はnilが返る", func(t *testing.T) {
		nonMemberUserID := testutil.NewUserBuilder(t, tx).
			WithEmail("sug-edit-uc-nonmember@example.com").
			WithAtname("sugeditucnonmember").
			Build()

		output, err := uc.Execute(context.Background(), GetSuggestionEditInput{
			SpaceIdentifier:  "sug-edit-uc-space",
			SuggestionNumber: 1,
			UserID:           nonMemberUserID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output != nil {
			t.Error("output should be nil for non-member user")
		}
	})
}

func TestGetSuggestionEditUsecase_Execute_非公開トピック(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	suggestionRepo := repository.NewSuggestionRepository(q)
	userRepo := repository.NewUserRepository(q)
	uc := NewGetSuggestionEditUsecase(spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, suggestionRepo, userRepo)

	// オーナーユーザー
	ownerUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("sug-edit-priv-owner@example.com").
		WithAtname("sugeditprivowner").
		Build()

	// トピックメンバーユーザー
	memberUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("sug-edit-priv-member@example.com").
		WithAtname("sugeditprivmember").
		Build()

	// トピックメンバーでないユーザー
	nonMemberUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("sug-edit-priv-nonmember@example.com").
		WithAtname("sugeditprivnonmember").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sug-edit-priv").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerUserID).
		WithRole(int32(model.SpaceMemberRoleOwner)).
		Build()
	memberSpaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(memberUserID).
		WithRole(int32(model.SpaceMemberRoleMember)).
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(nonMemberUserID).
		WithRole(int32(model.SpaceMemberRoleMember)).
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(int32(model.TopicVisibilityPrivate)).
		Build()

	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(memberSpaceMemberID).
		Build()

	testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(memberSpaceMemberID).
		WithTitle("非公開トピックの提案").
		WithStatus(model.SuggestionStatusOpen).
		Build()

	t.Run("スペースオーナーはアクセスできる", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetSuggestionEditInput{
			SpaceIdentifier:  "sug-edit-priv",
			SuggestionNumber: 1,
			UserID:           ownerUserID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil for space owner")
		}
	})

	t.Run("トピックメンバーはアクセスできる", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetSuggestionEditInput{
			SpaceIdentifier:  "sug-edit-priv",
			SuggestionNumber: 1,
			UserID:           memberUserID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil for topic member")
		}
	})

	t.Run("トピックメンバーでないスペースメンバーはnilが返る", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetSuggestionEditInput{
			SpaceIdentifier:  "sug-edit-priv",
			SuggestionNumber: 1,
			UserID:           nonMemberUserID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output != nil {
			t.Error("output should be nil for non-topic-member on private topic")
		}
	})
}
