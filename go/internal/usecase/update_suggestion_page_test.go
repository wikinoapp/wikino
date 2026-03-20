package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestUpdateSuggestionPageUsecase_Execute(t *testing.T) {
	db := testutil.GetTestDB()
	q := query.New(db)

	suggestionPageRepo := repository.NewSuggestionPageRepository(q)
	suggestionPageRevisionRepo := repository.NewSuggestionPageRevisionRepository(q)

	uc := NewUpdateSuggestionPageUsecase(db, suggestionPageRepo, suggestionPageRevisionRepo)

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

		updatedTitle := "更新されたタイトル"
		draftPage := &model.DraftPage{
			Title:    &updatedTitle,
			Body:     "更新された本文",
			BodyHTML: "<p>更新された本文</p>",
		}

		output, err := uc.Execute(context.Background(), UpdateSuggestionPageInput{
			SpaceID:          spaceID,
			SpaceMemberID:    spaceMemberID,
			SuggestionPageID: suggestionPageID,
			DraftPage:        draftPage,
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

		draftPage := &model.DraftPage{
			Title:    nil,
			Body:     "本文のみ更新",
			BodyHTML: "<p>本文のみ更新</p>",
		}

		output, err := uc.Execute(context.Background(), UpdateSuggestionPageInput{
			SpaceID:          spaceID,
			SpaceMemberID:    spaceMemberID,
			SuggestionPageID: suggestionPageID,
			DraftPage:        draftPage,
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
