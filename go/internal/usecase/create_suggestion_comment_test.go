package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestCreateSuggestionCommentUsecase_Execute(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)

	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	suggestionRepo := repository.NewSuggestionRepository(q)
	suggestionCommentRepo := repository.NewSuggestionCommentRepository(q)
	createValidator := validator.NewSuggestionCommentCreateValidator()

	uc := NewCreateSuggestionCommentUsecase(
		db, spaceRepo, spaceMemberRepo, topicMemberRepo, suggestionRepo, suggestionCommentRepo, createValidator,
	)

	t.Run("正常系: コメントが作成される", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("csc-ok@example.com").
			WithAtname("cscok").
			Build()
		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("csc-ok-sp").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("テストトピック").
			Build()
		testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithTitle("テスト提案").
			WithStatus(model.SuggestionStatusOpen).
			Build()

		output, err := uc.Execute(ctx, CreateSuggestionCommentInput{
			SpaceIdentifier:  "csc-ok-sp",
			SuggestionNumber: 1,
			UserID:           userID,
			Body:             "テストコメント",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil || output.Comment == nil {
			t.Fatal("output.Comment should not be nil")
		}
		if output.Comment.Body != "テストコメント" {
			t.Errorf("Body = %q, want %q", output.Comment.Body, "テストコメント")
		}
	})

	t.Run("異常系: 存在しないスペースでAppErrorが返る", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("csc-nosp@example.com").
			WithAtname("cscnosp").
			Build()

		_, err := uc.Execute(ctx, CreateSuggestionCommentInput{
			SpaceIdentifier:  "nonexistent",
			SuggestionNumber: 1,
			UserID:           userID,
			Body:             "コメント",
		})

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatal("expected AppError, got nil")
		}
		if ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("Code = %v, want AppErrCodeResourceNotFound", ae.Code)
		}
	})

	t.Run("異常系: スペースメンバーでないユーザーはAppError(Forbidden)が返る", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		ownerID := testutil.NewUserBuilderDB(t, db).
			WithEmail("csc-owner@example.com").
			WithAtname("cscowner").
			Build()
		nonMemberID := testutil.NewUserBuilderDB(t, db).
			WithEmail("csc-nonmember@example.com").
			WithAtname("cscnonmember").
			Build()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("csc-forbid-sp").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(ownerID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("制限トピック").
			Build()
		testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithTitle("制限提案").
			WithStatus(model.SuggestionStatusOpen).
			Build()

		_, err := uc.Execute(ctx, CreateSuggestionCommentInput{
			SpaceIdentifier:  "csc-forbid-sp",
			SuggestionNumber: 1,
			UserID:           nonMemberID,
			Body:             "コメント",
		})

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatal("expected AppError, got nil")
		}
		if ae.Code != model.AppErrCodeForbidden {
			t.Errorf("Code = %v, want AppErrCodeForbidden", ae.Code)
		}
	})

	t.Run("異常系: 本文が空の場合ValidationErrorが返る", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("csc-empty@example.com").
			WithAtname("cscempty").
			Build()
		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("csc-empty-sp").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("空コメントトピック").
			Build()
		testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithTitle("空コメント提案").
			WithStatus(model.SuggestionStatusOpen).
			Build()

		_, err := uc.Execute(ctx, CreateSuggestionCommentInput{
			SpaceIdentifier:  "csc-empty-sp",
			SuggestionNumber: 1,
			UserID:           userID,
			Body:             "",
		})

		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatal("expected ValidationError, got nil")
		}
		if !ve.HasFieldError("body") {
			t.Error("expected body field error")
		}
	})
}
