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

func TestMovePageUsecase_Execute(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	pageRepo := repository.NewPageRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	suggestionPageRepo := repository.NewSuggestionPageRepository(q)
	pageMoveValidator := validator.NewPageMoveCreateValidator(pageRepo, topicRepo, topicMemberRepo, suggestionPageRepo)
	uc := NewMovePageUsecase(db, spaceRepo, spaceMemberRepo, pageRepo, topicRepo, topicMemberRepo, draftPageRepo, pageMoveValidator)

	t.Run("正常系: ページを別のトピックに移動できる", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, i18n.LangJa)

		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("move-test-ok@example.com").
			WithAtname("move-test-ok").
			Build()
		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("move-test-ok").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		srcTopicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("Source Topic").
			WithNumber(1).
			Build()
		destTopicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("Dest Topic").
			WithNumber(2).
			Build()
		testutil.NewTopicMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(srcTopicID).
			WithSpaceMemberID(spaceMemberID).
			Build()
		pageID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(srcTopicID).
			WithNumber(1).
			WithTitle("Move Test Page").
			Build()

		// 移動対象ページに下書きを作成
		draftPageID := testutil.NewDraftPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithPageID(pageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(srcTopicID).
			WithBody("Draft body").
			Build()

		output, err := uc.Execute(ctx, MovePageInput{
			SpaceIdentifier: "move-test-ok",
			PageNumber:      1,
			UserID:          userID,
			DestTopicNumber: "2",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v, want nil", err)
		}
		if output == nil || output.Page == nil {
			t.Fatal("output.Page should not be nil")
		}
		if output.Page.SpaceID != spaceID {
			t.Errorf("Page.SpaceID = %v, want %v", output.Page.SpaceID, spaceID)
		}

		// 下書きのトピックも移動��に更新されていることを確認
		q := repository.NewDraftPageRepository(query.New(db))
		dp, err := q.FindByID(ctx, draftPageID, spaceID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if dp.TopicID != destTopicID {
			t.Errorf("DraftPage.TopicID = %v, want %v", dp.TopicID, destTopicID)
		}
	})

	t.Run("異常系: 存在しないスペースでAppErrorが返る", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, i18n.LangJa)

		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("move-test-nospace@example.com").
			WithAtname("move-test-nospace").
			Build()

		_, err := uc.Execute(ctx, MovePageInput{
			SpaceIdentifier: "nonexistent-space",
			PageNumber:      1,
			UserID:          userID,
			DestTopicNumber: "2",
		})

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatal("expected AppError but got nil")
		}
		if ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("AppError.Code = %v, want %v", ae.Code, model.AppErrCodeResourceNotFound)
		}
	})

	t.Run("異常系: 権限がないユーザーでAppErrorが返る", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, i18n.LangJa)

		// スペースメンバーでないユーザー
		nonMemberID := testutil.NewUserBuilderDB(t, db).
			WithEmail("move-test-forbidden@example.com").
			WithAtname("move-test-forbidden").
			Build()
		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("move-test-forbidden").
			Build()
		srcTopicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("Source").
			WithNumber(1).
			Build()
		testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(srcTopicID).
			WithNumber(1).
			WithTitle("Test").
			Build()

		_, err := uc.Execute(ctx, MovePageInput{
			SpaceIdentifier: "move-test-forbidden",
			PageNumber:      1,
			UserID:          nonMemberID,
			DestTopicNumber: "2",
		})

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatal("expected AppError but got nil")
		}
		if ae.Code != model.AppErrCodeForbidden {
			t.Errorf("AppError.Code = %v, want %v", ae.Code, model.AppErrCodeForbidden)
		}
	})

	t.Run("異常系: バリデーションエラーでValidationErrorが返る", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, i18n.LangJa)

		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("move-test-ve@example.com").
			WithAtname("move-test-ve").
			Build()
		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("move-test-ve").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		srcTopicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("Source").
			WithNumber(1).
			Build()
		testutil.NewTopicMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(srcTopicID).
			WithSpaceMemberID(spaceMemberID).
			Build()
		testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(srcTopicID).
			WithNumber(1).
			WithTitle("Test").
			Build()

		// 移動先トピックを空にしてバリデーションエラーを発生させる
		_, err := uc.Execute(ctx, MovePageInput{
			SpaceIdentifier: "move-test-ve",
			PageNumber:      1,
			UserID:          userID,
			DestTopicNumber: "",
		})

		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatal("expected ValidationError but got nil")
		}
		if !ve.HasFieldError("dest_topic") {
			t.Error("expected dest_topic field error")
		}
	})
}
