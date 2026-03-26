package usecase

import (
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestUpdateSuggestionCommentUsecase_Execute(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)

	suggestionCommentRepo := repository.NewSuggestionCommentRepository(q)
	uc := NewUpdateSuggestionCommentUsecase(db, suggestionCommentRepo)

	t.Run("正常系: コメントの本文を更新できる", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("update-sc-1").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("update-sc-1@example.com").
			WithAtname("updatesc1").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("テストトピック").
			Build()
		suggestionID := testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithTitle("テスト提案").
			WithStatus(model.SuggestionStatusOpen).
			Build()
		commentID := testutil.NewSuggestionCommentBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithBody("更新前のコメント").
			Build()

		output, err := uc.Execute(ctx, UpdateSuggestionCommentInput{
			CommentID: commentID,
			SpaceID:   spaceID,
			Body:      "更新後のコメント",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.Comment.Body != "更新後のコメント" {
			t.Errorf("Body = %q, want %q", output.Comment.Body, "更新後のコメント")
		}
		if output.Comment.BodyHTML == "" {
			t.Error("BodyHTML should not be empty")
		}
	})

	t.Run("正常系: Markdown本文がHTMLに変換される", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("update-sc-2").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("update-sc-2@example.com").
			WithAtname("updatesc2").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("テストトピック2").
			Build()
		suggestionID := testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithTitle("テスト提案2").
			WithStatus(model.SuggestionStatusOpen).
			Build()
		commentID := testutil.NewSuggestionCommentBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithBody("元のコメント").
			Build()

		output, err := uc.Execute(ctx, UpdateSuggestionCommentInput{
			CommentID: commentID,
			SpaceID:   spaceID,
			Body:      "**太字** テスト",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if !strings.Contains(output.Comment.BodyHTML, "<strong>") {
			t.Errorf("BodyHTML should contain <strong> tag, got: %s", output.Comment.BodyHTML)
		}
	})

	t.Run("異なるスペースIDではコメントが更新されない", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("update-sc-3").
			Build()
		otherSpaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("update-sc-3-other").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("update-sc-3@example.com").
			WithAtname("updatesc3").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("テストトピック3").
			Build()
		suggestionID := testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithTitle("テスト提案3").
			WithStatus(model.SuggestionStatusOpen).
			Build()
		commentID := testutil.NewSuggestionCommentBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithBody("元のコメント").
			Build()

		output, err := uc.Execute(ctx, UpdateSuggestionCommentInput{
			CommentID: commentID,
			SpaceID:   otherSpaceID,
			Body:      "不正な更新",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.Comment != nil {
			t.Errorf("Comment should be nil when space_id does not match, got: %v", output.Comment)
		}
	})
}
