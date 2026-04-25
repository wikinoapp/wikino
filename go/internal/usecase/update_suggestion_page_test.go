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

func TestUpdateSuggestionPageUsecase_Execute(t *testing.T) {
	db := testutil.GetTestDB()
	q := query.New(db)

	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	suggestionRepo := repository.NewSuggestionRepository(q)
	suggestionPageRepo := repository.NewSuggestionPageRepository(q)
	suggestionPageRevisionRepo := repository.NewSuggestionPageRevisionRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)

	updateValidator := validator.NewSuggestionPageUpdateValidator(draftPageRepo)

	uc := NewUpdateSuggestionPageUsecase(
		db, spaceRepo, spaceMemberRepo, topicMemberRepo,
		suggestionRepo, suggestionPageRepo, suggestionPageRevisionRepo, updateValidator,
	)

	t.Run("正常系: SuggestionPageのコンテンツが更新されSuggestionPageRevisionが作成される", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("upd-sp-uc-ok").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("upd-sp-uc-ok@example.com").
			WithAtname("updspucok").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("Topic").
			Build()
		suggestionID := testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithStatus(model.SuggestionStatusOpen).
			Build()

		pageID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("Original Title").
			Build()
		pageRevisionID := testutil.NewPageRevisionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithPageID(pageID).
			WithSpaceMemberID(spaceMemberID).
			Build()
		suggestionPageID := testutil.NewSuggestionPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithPageID(pageID).
			WithPageRevisionID(pageRevisionID).
			WithBody("元の提案本文").
			Build()

		testutil.NewDraftPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithPageID(pageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithSuggestionPageID(suggestionPageID).
			WithBody("更新された本文").
			WithBodyHTML("<p>更新された本文</p>").
			WithTitle("更新されたタイトル").
			Build()

		output, err := uc.Execute(context.Background(), UpdateSuggestionPageInput{
			SpaceIdentifier:  "upd-sp-uc-ok",
			SuggestionNumber: 1,
			SuggestionPageID: suggestionPageID,
			UserID:           userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.SuggestionPage == nil {
			t.Fatal("SuggestionPage should not be nil")
		}

		// SuggestionPageのコンテンツが更新されていることを確認
		updatedSP, err := suggestionPageRepo.FindByID(context.Background(), suggestionPageID, spaceID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if updatedSP.Body != "更新された本文" {
			t.Errorf("SuggestionPage.Body = %q, want %q", updatedSP.Body, "更新された本文")
		}
		if updatedSP.BodyHTML != "<p>更新された本文</p>" {
			t.Errorf("SuggestionPage.BodyHTML = %q, want %q", updatedSP.BodyHTML, "<p>更新された本文</p>")
		}
		if updatedSP.Title == nil || *updatedSP.Title != "更新されたタイトル" {
			t.Errorf("SuggestionPage.Title = %v, want %q", updatedSP.Title, "更新されたタイトル")
		}

		// SuggestionPageRevisionが作成されていることを確認
		revisions, err := suggestionPageRevisionRepo.ListBySuggestionPageID(context.Background(), suggestionPageID, spaceID)
		if err != nil {
			t.Fatalf("ListBySuggestionPageID() error = %v", err)
		}
		if len(revisions) == 0 {
			t.Fatal("SuggestionPageRevisionが作成されていません")
		}
		latestRevision := revisions[len(revisions)-1]
		if latestRevision.Body != "更新された本文" {
			t.Errorf("Revision.Body = %q, want %q", latestRevision.Body, "更新された本文")
		}
		if latestRevision.EditorSpaceMemberID != spaceMemberID {
			t.Errorf("Revision.EditorSpaceMemberID = %v, want %v", latestRevision.EditorSpaceMemberID, spaceMemberID)
		}
	})

	t.Run("異常系: 存在しないスペースの場合はAppErrorが返る", func(t *testing.T) {
		t.Parallel()

		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("upd-sp-uc-nospace@example.com").
			WithAtname("updspucnosp").
			Build()

		_, err := uc.Execute(context.Background(), UpdateSuggestionPageInput{
			SpaceIdentifier:  "nonexistent-space",
			SuggestionNumber: 1,
			SuggestionPageID: "test-sp-id",
			UserID:           userID,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatalf("expected AppError, got %T: %v", err, err)
		}
		if ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("AppError.Code = %v, want %v", ae.Code, model.AppErrCodeResourceNotFound)
		}
	})

	t.Run("異常系: スペースメンバーでない場合はAppError(Forbidden)が返る", func(t *testing.T) {
		t.Parallel()

		nonMemberID := testutil.NewUserBuilderDB(t, db).
			WithEmail("upd-sp-uc-nonmem@example.com").
			WithAtname("updspucnonm").
			Build()
		ownerID := testutil.NewUserBuilderDB(t, db).
			WithEmail("upd-sp-uc-owner2@example.com").
			WithAtname("updspucown2").
			Build()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("upd-sp-uc-forbid").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(ownerID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("Topic").
			Build()
		suggestionID := testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithStatus(model.SuggestionStatusOpen).
			Build()

		pageID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			Build()
		pageRevisionID := testutil.NewPageRevisionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithPageID(pageID).
			WithSpaceMemberID(spaceMemberID).
			Build()
		suggestionPageID := testutil.NewSuggestionPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithPageID(pageID).
			WithPageRevisionID(pageRevisionID).
			Build()

		_, err := uc.Execute(context.Background(), UpdateSuggestionPageInput{
			SpaceIdentifier:  "upd-sp-uc-forbid",
			SuggestionNumber: 1,
			SuggestionPageID: suggestionPageID,
			UserID:           nonMemberID,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatalf("expected AppError, got %T: %v", err, err)
		}
		if ae.Code != model.AppErrCodeForbidden {
			t.Errorf("AppError.Code = %v, want %v", ae.Code, model.AppErrCodeForbidden)
		}
	})

	t.Run("異常系: クローズ済みの編集提案はAppError(Forbidden)が返る", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("upd-sp-uc-closed").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("upd-sp-uc-closed@example.com").
			WithAtname("updspucclosed").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("Topic").
			Build()
		suggestionID := testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithStatus(model.SuggestionStatusClosed).
			Build()

		pageID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			Build()
		pageRevisionID := testutil.NewPageRevisionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithPageID(pageID).
			WithSpaceMemberID(spaceMemberID).
			Build()
		suggestionPageID := testutil.NewSuggestionPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithPageID(pageID).
			WithPageRevisionID(pageRevisionID).
			Build()

		_, err := uc.Execute(context.Background(), UpdateSuggestionPageInput{
			SpaceIdentifier:  "upd-sp-uc-closed",
			SuggestionNumber: 1,
			SuggestionPageID: suggestionPageID,
			UserID:           userID,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatalf("expected AppError, got %T: %v", err, err)
		}
		if ae.Code != model.AppErrCodeForbidden {
			t.Errorf("AppError.Code = %v, want %v", ae.Code, model.AppErrCodeForbidden)
		}
	})

	t.Run("正常系: タイトルがnilの場合もSuggestionPageが更新される", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("upd-sp-uc-niltitle").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("upd-sp-uc-niltitle@example.com").
			WithAtname("updspucnilt").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("Topic").
			Build()
		suggestionID := testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithStatus(model.SuggestionStatusOpen).
			Build()

		pageID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			Build()
		pageRevisionID := testutil.NewPageRevisionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithPageID(pageID).
			WithSpaceMemberID(spaceMemberID).
			Build()
		suggestionPageID := testutil.NewSuggestionPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithPageID(pageID).
			WithPageRevisionID(pageRevisionID).
			Build()

		testutil.NewDraftPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithPageID(pageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithSuggestionPageID(suggestionPageID).
			WithBody("本文のみ更新").
			WithBodyHTML("<p>本文のみ更新</p>").
			Build()

		output, err := uc.Execute(context.Background(), UpdateSuggestionPageInput{
			SpaceIdentifier:  "upd-sp-uc-niltitle",
			SuggestionNumber: 1,
			SuggestionPageID: suggestionPageID,
			UserID:           userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}

		updatedSP, err := suggestionPageRepo.FindByID(context.Background(), suggestionPageID, spaceID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if updatedSP.Body != "本文のみ更新" {
			t.Errorf("SuggestionPage.Body = %q, want %q", updatedSP.Body, "本文のみ更新")
		}
	})
}
