package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetSuggestionCommentUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)

	suggestionCommentRepo := repository.NewSuggestionCommentRepository(q)
	uc := NewGetSuggestionCommentUsecase(suggestionCommentRepo)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("get-sc-1").
		Build()
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("get-sc-1@example.com").
		WithAtname("getsc1").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithRole(0).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithName("テストトピック").
		Build()
	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithTitle("テスト提案").
		WithStatus(model.SuggestionStatusOpen).
		Build()
	testutil.NewSuggestionCommentBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithBody("テストコメント").
		Build()

	t.Run("正常系: コメント番号でコメントを取得できる", func(t *testing.T) {
		ctx := context.Background()
		output, err := uc.Execute(ctx, GetSuggestionCommentInput{
			SuggestionID:  suggestionID,
			CommentNumber: 1,
			SpaceID:       spaceID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output.Comment == nil {
			t.Fatal("Comment should not be nil")
		}
		if output.Comment.Body != "テストコメント" {
			t.Errorf("Comment.Body = %q, want %q", output.Comment.Body, "テストコメント")
		}
	})

	t.Run("存在しないコメント番号の場合はnilを返す", func(t *testing.T) {
		ctx := context.Background()
		output, err := uc.Execute(ctx, GetSuggestionCommentInput{
			SuggestionID:  suggestionID,
			CommentNumber: 999,
			SpaceID:       spaceID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output.Comment != nil {
			t.Errorf("Comment should be nil, got: %v", output.Comment)
		}
	})

	t.Run("異なるスペースIDではnilを返す", func(t *testing.T) {
		ctx := context.Background()
		otherSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("get-sc-other").
			Build()

		output, err := uc.Execute(ctx, GetSuggestionCommentInput{
			SuggestionID:  suggestionID,
			CommentNumber: 1,
			SpaceID:       otherSpaceID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output.Comment != nil {
			t.Errorf("Comment should be nil, got: %v", output.Comment)
		}
	})
}
