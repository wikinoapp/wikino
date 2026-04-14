package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetSuggestionListUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	suggestionRepo := repository.NewSuggestionRepository(q)
	userRepo := repository.NewUserRepository(q)
	uc := NewGetSuggestionListUsecase(spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, suggestionRepo, userRepo)

	// テストデータのセットアップ
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("sug-list@example.com").
		WithAtname("suglistuser").
		WithName("提案太郎").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sug-list-space").
		WithName("SugList Space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("提案トピック").
		Build()

	t.Run("編集提案がない場合は空のスライスが返る", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetSuggestionListInput{
			SpaceIdentifier: "sug-list-space",
			TopicNumber:     1,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if len(output.Suggestions) != 0 {
			t.Errorf("len(Suggestions) = %d, want 0", len(output.Suggestions))
		}
		if output.OpenCount != 0 {
			t.Errorf("OpenCount = %d, want 0", output.OpenCount)
		}
		if output.ClosedCount != 0 {
			t.Errorf("ClosedCount = %d, want 0", output.ClosedCount)
		}
		if output.Space == nil {
			t.Error("Space should not be nil")
		}
		if output.Topic == nil {
			t.Error("Topic should not be nil")
		}
	})

	t.Run("オープンステータスの編集提案が取得できる", func(t *testing.T) {
		testutil.NewSuggestionBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithTitle("オープン提案1").
			WithStatus(model.SuggestionStatusOpen).
			Build()

		testutil.NewSuggestionBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithTitle("下書き提案").
			WithStatus(model.SuggestionStatusDraft).
			Build()

		testutil.NewSuggestionBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithTitle("クローズ提案").
			WithStatus(model.SuggestionStatusClosed).
			Build()

		// オープン表示（下書き・オープン）
		output, err := uc.Execute(context.Background(), GetSuggestionListInput{
			SpaceIdentifier: "sug-list-space",
			TopicNumber:     1,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if len(output.Suggestions) != 2 {
			t.Errorf("len(Suggestions) = %d, want 2", len(output.Suggestions))
		}
		if output.OpenCount != 2 {
			t.Errorf("OpenCount = %d, want 2", output.OpenCount)
		}
		if output.ClosedCount != 1 {
			t.Errorf("ClosedCount = %d, want 1", output.ClosedCount)
		}
	})

	t.Run("作成者のユーザー情報がUserMapに含まれる", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetSuggestionListInput{
			SpaceIdentifier: "sug-list-space",
			TopicNumber:     1,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		user, ok := output.UserMap[spaceMemberID]
		if !ok {
			t.Fatal("UserMap should contain the creator's user info")
		}
		if user.Name != "提案太郎" {
			t.Errorf("user.Name = %q, want %q", user.Name, "提案太郎")
		}
	})

	t.Run("存在しないスペースの場合はnilが返る", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetSuggestionListInput{
			SpaceIdentifier: "nonexistent",
			TopicNumber:     1,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output != nil {
			t.Error("output should be nil for nonexistent space")
		}
	})

	t.Run("存在しないトピックの場合はnilが返る", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetSuggestionListInput{
			SpaceIdentifier: "sug-list-space",
			TopicNumber:     999,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output != nil {
			t.Error("output should be nil for nonexistent topic")
		}
	})

	t.Run("スペースメンバーが閲覧した場合はCanCreateSuggestionがtrue", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetSuggestionListInput{
			SpaceIdentifier: "sug-list-space",
			TopicNumber:     1,
			UserID:          &userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if !output.CanCreateSuggestion {
			t.Error("CanCreateSuggestion should be true for space member")
		}
	})

	t.Run("非メンバーのログインユーザーが閲覧した場合はCanCreateSuggestionがfalse", func(t *testing.T) {
		outsiderID := testutil.NewUserBuilder(t, tx).
			WithEmail("sug-list-outsider@example.com").
			WithAtname("suglistoutsider").
			WithName("よそ者").
			Build()

		output, err := uc.Execute(context.Background(), GetSuggestionListInput{
			SpaceIdentifier: "sug-list-space",
			TopicNumber:     1,
			UserID:          &outsiderID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.CanCreateSuggestion {
			t.Error("CanCreateSuggestion should be false for non-member")
		}
	})

	t.Run("未ログインで閲覧した場合はCanCreateSuggestionがfalse", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetSuggestionListInput{
			SpaceIdentifier: "sug-list-space",
			TopicNumber:     1,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.CanCreateSuggestion {
			t.Error("CanCreateSuggestion should be false for guest")
		}
	})
}
