package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestStartSuggestionPageEditUsecase_Execute(t *testing.T) {
	db := testutil.GetTestDB()
	q := query.New(db)

	suggestionPageRepo := repository.NewSuggestionPageRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	pageRepo := repository.NewPageRepository(q)

	uc := NewStartSuggestionPageEditUsecase(db, suggestionPageRepo, draftPageRepo, pageRepo)

	t.Run("正常系: 下書きが存在しない場合に新規作成してリダイレクト", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("start-edit-1").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("start-edit-1@example.com").
			WithAtname("startedit1").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("Topic").
			Build()
		pageID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("Test Page").
			Build()
		pageRevisionID := createPageRevisionViaRepo(t, q, spaceID, spaceMemberID, pageID)

		suggestionID := testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			Build()
		suggestionPageID := testutil.NewSuggestionPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithPageID(pageID).
			WithPageRevisionID(pageRevisionID).
			WithTitle("提案タイトル").
			WithBody("提案本文").
			WithBodyHTML("<p>提案本文</p>").
			Build()

		output, err := uc.Execute(context.Background(), StartSuggestionPageEditInput{
			SpaceID:          spaceID,
			SpaceMemberID:    spaceMemberID,
			SuggestionPageID: suggestionPageID,
			Force:            false,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output.Status != StartSuggestionPageEditRedirect {
			t.Errorf("Status = %d, want %d (Redirect)", output.Status, StartSuggestionPageEditRedirect)
		}
		if output.PageNumber != 1 {
			t.Errorf("PageNumber = %d, want 1", output.PageNumber)
		}

		// 下書きが作成されたことを確認
		draft, err := draftPageRepo.FindByPageAndMember(context.Background(), pageID, spaceMemberID, spaceID)
		if err != nil {
			t.Fatalf("FindByPageAndMember() error = %v", err)
		}
		if draft == nil {
			t.Fatal("DraftPage should have been created")
		}
		if draft.SuggestionPageID == nil || *draft.SuggestionPageID != suggestionPageID {
			t.Errorf("DraftPage.SuggestionPageID = %v, want %v", draft.SuggestionPageID, suggestionPageID)
		}
		if draft.Body != "提案本文" {
			t.Errorf("DraftPage.Body = %q, want %q", draft.Body, "提案本文")
		}
	})

	t.Run("正常系: 同じ編集提案ページにリンク済みの下書きがある場合はそのままリダイレクト", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("start-edit-2").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("start-edit-2@example.com").
			WithAtname("startedit2").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("Topic").
			Build()
		pageID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("Test Page").
			Build()
		pageRevisionID := createPageRevisionViaRepo(t, q, spaceID, spaceMemberID, pageID)

		suggestionID := testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			Build()
		suggestionPageID := testutil.NewSuggestionPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithPageID(pageID).
			WithPageRevisionID(pageRevisionID).
			Build()

		// 既にリンク済みの下書きを作成
		testutil.NewDraftPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithPageID(pageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithSuggestionPageID(suggestionPageID).
			Build()

		output, err := uc.Execute(context.Background(), StartSuggestionPageEditInput{
			SpaceID:          spaceID,
			SpaceMemberID:    spaceMemberID,
			SuggestionPageID: suggestionPageID,
			Force:            false,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output.Status != StartSuggestionPageEditRedirect {
			t.Errorf("Status = %d, want %d (Redirect)", output.Status, StartSuggestionPageEditRedirect)
		}
	})

	t.Run("正常系: 通常編集の下書きがある場合はConflictを返す", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("start-edit-3").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("start-edit-3@example.com").
			WithAtname("startedit3").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("Topic").
			Build()
		pageID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("Test Page").
			Build()
		pageRevisionID := createPageRevisionViaRepo(t, q, spaceID, spaceMemberID, pageID)

		suggestionID := testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			Build()
		suggestionPageID := testutil.NewSuggestionPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithPageID(pageID).
			WithPageRevisionID(pageRevisionID).
			Build()

		// 通常編集の下書き（suggestion_page_id = NULL）を作成
		testutil.NewDraftPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithPageID(pageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithBody("通常編集の下書き").
			Build()

		output, err := uc.Execute(context.Background(), StartSuggestionPageEditInput{
			SpaceID:          spaceID,
			SpaceMemberID:    spaceMemberID,
			SuggestionPageID: suggestionPageID,
			Force:            false,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output.Status != StartSuggestionPageEditConflict {
			t.Errorf("Status = %d, want %d (Conflict)", output.Status, StartSuggestionPageEditConflict)
		}
	})

	t.Run("正常系: Force=trueで通常編集の下書きを上書きしてリダイレクト", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("start-edit-4").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("start-edit-4@example.com").
			WithAtname("startedit4").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("Topic").
			Build()
		pageID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("Test Page").
			Build()
		pageRevisionID := createPageRevisionViaRepo(t, q, spaceID, spaceMemberID, pageID)

		suggestionID := testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			Build()
		suggestionPageID := testutil.NewSuggestionPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithPageID(pageID).
			WithPageRevisionID(pageRevisionID).
			WithBody("提案の本文").
			WithBodyHTML("<p>提案の本文</p>").
			Build()

		// 通常編集の下書きを作成
		testutil.NewDraftPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithPageID(pageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithBody("通常編集の下書き").
			Build()

		output, err := uc.Execute(context.Background(), StartSuggestionPageEditInput{
			SpaceID:          spaceID,
			SpaceMemberID:    spaceMemberID,
			SuggestionPageID: suggestionPageID,
			Force:            true,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output.Status != StartSuggestionPageEditRedirect {
			t.Errorf("Status = %d, want %d (Redirect)", output.Status, StartSuggestionPageEditRedirect)
		}

		// 下書きが上書きされたことを確認
		draft, err := draftPageRepo.FindByPageAndMember(context.Background(), pageID, spaceMemberID, spaceID)
		if err != nil {
			t.Fatalf("FindByPageAndMember() error = %v", err)
		}
		if draft == nil {
			t.Fatal("DraftPage should exist")
		}
		if draft.SuggestionPageID == nil || *draft.SuggestionPageID != suggestionPageID {
			t.Errorf("DraftPage.SuggestionPageID = %v, want %v", draft.SuggestionPageID, suggestionPageID)
		}
		if draft.Body != "提案の本文" {
			t.Errorf("DraftPage.Body = %q, want %q", draft.Body, "提案の本文")
		}
	})
}
