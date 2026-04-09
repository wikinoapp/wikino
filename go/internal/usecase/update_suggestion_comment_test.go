package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestUpdateSuggestionCommentUsecase_Execute(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)

	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	suggestionRepo := repository.NewSuggestionRepository(q)
	suggestionCommentRepo := repository.NewSuggestionCommentRepository(q)
	updateValidator := validator.NewSuggestionCommentUpdateValidator()

	uc := NewUpdateSuggestionCommentUsecase(
		db, spaceRepo, spaceMemberRepo, topicMemberRepo,
		suggestionRepo, suggestionCommentRepo, updateValidator,
	)

	t.Run("正常系: コメントの本文を更新できる", func(t *testing.T) {
		t.Parallel()

		ctx := i18n.SetLocale(context.Background(), i18n.LangJa)

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("update-sc-uc-1").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("update-sc-uc-1@example.com").
			WithAtname("updatescuc1").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("テストトピック").
			Build()
		testutil.NewTopicMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			Build()
		suggestionID := testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithTitle("テスト提案").
			WithStatus(model.SuggestionStatusOpen).
			Build()
		testutil.NewSuggestionCommentBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithBody("更新前のコメント").
			Build()

		output, err := uc.Execute(ctx, UpdateSuggestionCommentInput{
			SpaceIdentifier:  "update-sc-uc-1",
			SuggestionNumber: 1,
			CommentNumber:    1,
			UserID:           userID,
			Body:             "更新後のコメント",
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
	})

	t.Run("異常系: 存在しないスペースでAppErrorが返る", func(t *testing.T) {
		t.Parallel()

		ctx := i18n.SetLocale(context.Background(), i18n.LangJa)

		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("update-sc-uc-3@example.com").
			WithAtname("updatescuc3").
			Build()

		_, err := uc.Execute(ctx, UpdateSuggestionCommentInput{
			SpaceIdentifier:  "nonexistent-space",
			SuggestionNumber: 1,
			CommentNumber:    1,
			UserID:           userID,
			Body:             "コメント",
		})

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatal("expected AppError, got nil")
		}
		if ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("Code = %v, want %v", ae.Code, model.AppErrCodeResourceNotFound)
		}
	})

	t.Run("異常系: スペースメンバーでない場合AppErrorが返る", func(t *testing.T) {
		t.Parallel()

		ctx := i18n.SetLocale(context.Background(), i18n.LangJa)

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("update-sc-uc-4").
			Build()
		ownerUserID := testutil.NewUserBuilderDB(t, db).
			WithEmail("update-sc-uc-4-owner@example.com").
			WithAtname("updatescuc4owner").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(ownerUserID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("テストトピック4").
			Build()
		suggestionID := testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithTitle("テスト提案4").
			WithStatus(model.SuggestionStatusOpen).
			Build()
		testutil.NewSuggestionCommentBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithBody("元のコメント").
			Build()

		nonMemberUserID := testutil.NewUserBuilderDB(t, db).
			WithEmail("update-sc-uc-4-nonmember@example.com").
			WithAtname("updatescuc4nm").
			Build()

		_, err := uc.Execute(ctx, UpdateSuggestionCommentInput{
			SpaceIdentifier:  "update-sc-uc-4",
			SuggestionNumber: 1,
			CommentNumber:    1,
			UserID:           nonMemberUserID,
			Body:             "不正な更新",
		})

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatal("expected AppError, got nil")
		}
		if ae.Code != model.AppErrCodeForbidden {
			t.Errorf("Code = %v, want %v", ae.Code, model.AppErrCodeForbidden)
		}
	})

	t.Run("異常系: 本文が空の場合バリデーションエラーが返る", func(t *testing.T) {
		t.Parallel()

		ctx := i18n.SetLocale(context.Background(), i18n.LangJa)

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("update-sc-uc-5").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("update-sc-uc-5@example.com").
			WithAtname("updatescuc5").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("テストトピック5").
			Build()
		testutil.NewTopicMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			Build()
		suggestionID := testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithTitle("テスト提案5").
			WithStatus(model.SuggestionStatusOpen).
			Build()
		testutil.NewSuggestionCommentBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithBody("元のコメント").
			Build()

		_, err := uc.Execute(ctx, UpdateSuggestionCommentInput{
			SpaceIdentifier:  "update-sc-uc-5",
			SuggestionNumber: 1,
			CommentNumber:    1,
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
