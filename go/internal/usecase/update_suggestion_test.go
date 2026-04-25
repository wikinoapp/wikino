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

func TestUpdateSuggestionUsecase_Execute(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)

	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	suggestionRepo := repository.NewSuggestionRepository(q)
	updateValidator := validator.NewSuggestionUpdateValidator()

	uc := NewUpdateSuggestionUsecase(db, spaceRepo, spaceMemberRepo, topicMemberRepo, suggestionRepo, updateValidator)

	t.Run("正常系: タイトルと本文を更新できる", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("update-sug-1").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("update-sug-1@example.com").
			WithAtname("updatesug1").
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
			WithTitle("旧タイトル").
			WithStatus(model.SuggestionStatusOpen).
			Build()

		output, err := uc.Execute(ctx, UpdateSuggestionInput{
			SpaceIdentifier:  "update-sug-1",
			SuggestionNumber: 1,
			UserID:           userID,
			Title:            "新タイトル",
			Body:             "新本文",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if output.Suggestion.Title != "新タイトル" {
			t.Errorf("Title = %q, want %q", output.Suggestion.Title, "新タイトル")
		}
		if output.Suggestion.Body != "新本文" {
			t.Errorf("Body = %q, want %q", output.Suggestion.Body, "新本文")
		}
	})

	t.Run("正常系: 本文を空に更新できる", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("update-sug-2").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("update-sug-2@example.com").
			WithAtname("updatesug2").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("テストトピック2").
			Build()
		testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithTitle("テスト").
			WithStatus(model.SuggestionStatusOpen).
			Build()

		output, err := uc.Execute(ctx, UpdateSuggestionInput{
			SpaceIdentifier:  "update-sug-2",
			SuggestionNumber: 1,
			UserID:           userID,
			Title:            "更新タイトル",
			Body:             "",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if output.Suggestion.Title != "更新タイトル" {
			t.Errorf("Title = %q, want %q", output.Suggestion.Title, "更新タイトル")
		}
		if output.Suggestion.Body != "" {
			t.Errorf("Body = %q, want empty", output.Suggestion.Body)
		}
	})

	t.Run("異常系: 存在しないスペースでAppErrorが返る", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("update-sug-notfound@example.com").
			WithAtname("updatesugnotfound").
			Build()

		_, err := uc.Execute(ctx, UpdateSuggestionInput{
			SpaceIdentifier:  "nonexistent-space",
			SuggestionNumber: 1,
			UserID:           userID,
			Title:            "テスト",
			Body:             "",
		})

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatal("expected AppError, got nil")
		}
		if ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("Code = %d, want %d", ae.Code, model.AppErrCodeResourceNotFound)
		}
	})

	t.Run("異常系: 非メンバーはForbiddenエラーが返る", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("update-sug-forbidden").
			Build()
		ownerUserID := testutil.NewUserBuilderDB(t, db).
			WithEmail("update-sug-owner@example.com").
			WithAtname("updatesugowner").
			Build()
		ownerMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(ownerUserID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("テストトピック").
			Build()
		testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(ownerMemberID).
			WithTitle("テスト").
			WithStatus(model.SuggestionStatusOpen).
			Build()

		// 別のユーザー（スペースメンバーではない）
		nonMemberUserID := testutil.NewUserBuilderDB(t, db).
			WithEmail("update-sug-nonmember@example.com").
			WithAtname("updatesugnonmember").
			Build()

		_, err := uc.Execute(ctx, UpdateSuggestionInput{
			SpaceIdentifier:  "update-sug-forbidden",
			SuggestionNumber: 1,
			UserID:           nonMemberUserID,
			Title:            "テスト",
			Body:             "",
		})

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatal("expected AppError, got nil")
		}
		if ae.Code != model.AppErrCodeForbidden {
			t.Errorf("Code = %d, want %d", ae.Code, model.AppErrCodeForbidden)
		}
	})

	t.Run("異常系: タイトル空でValidationErrorが返る", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("update-sug-val").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("update-sug-val@example.com").
			WithAtname("updatesugval").
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
			WithTitle("テスト").
			WithStatus(model.SuggestionStatusOpen).
			Build()

		_, err := uc.Execute(ctx, UpdateSuggestionInput{
			SpaceIdentifier:  "update-sug-val",
			SuggestionNumber: 1,
			UserID:           userID,
			Title:            "",
			Body:             "本文",
		})

		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatal("expected ValidationError, got nil")
		}
		if !ve.HasFieldError("title") {
			t.Error("expected title field error")
		}
	})
}
