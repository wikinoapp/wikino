package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// createPageRevisionViaRepo はリポジトリ経由でページリビジョンを作成するヘルパー
func createPageRevisionViaRepo(t *testing.T, q *query.Queries, spaceID model.SpaceID, spaceMemberID model.SpaceMemberID, pageID model.PageID) model.PageRevisionID {
	t.Helper()

	repo := repository.NewPageRevisionRepository(q)
	rev, err := repo.Create(context.Background(), repository.CreatePageRevisionInput{
		SpaceID:       spaceID,
		SpaceMemberID: spaceMemberID,
		PageID:        pageID,
		Title:         "Revision Title",
		Body:          "Revision body",
		BodyHTML:      "<p>Revision body</p>",
	})
	if err != nil {
		t.Fatalf("ページリビジョン作成に失敗: %v", err)
	}
	return rev.ID
}

func TestCreateSuggestionUsecase_Execute(t *testing.T) {
	db := testutil.GetTestDB()
	q := query.New(db)

	suggestionRepo := repository.NewSuggestionRepository(q)
	suggestionPageRepo := repository.NewSuggestionPageRepository(q)
	suggestionPageRevisionRepo := repository.NewSuggestionPageRevisionRepository(q)
	pageRevisionRepo := repository.NewPageRevisionRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	pageRepo := repository.NewPageRepository(q)

	uc := NewCreateSuggestionUsecase(db, suggestionRepo, suggestionPageRepo, suggestionPageRevisionRepo, pageRevisionRepo, topicRepo, pageRepo)

	t.Run("正常系: 1つの下書きページから編集提案を作成できる", func(t *testing.T) {
		// テストデータを作成
		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("create-suggestion-1").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("create-suggestion-1@example.com").
			WithAtname("createsuggestion1").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("General").
			Build()
		pageID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("Test Page").
			Build()

		// ページリビジョンを作成（ベースリビジョンとして必要）
		createPageRevisionViaRepo(t, q, spaceID, spaceMemberID, pageID)

		draftTitle := "提案タイトル"
		output, err := uc.Execute(context.Background(), CreateSuggestionInput{
			SpaceID:          spaceID,
			TopicID:          topicID,
			SpaceMemberID:    spaceMemberID,
			SpaceIdentifier:  "create-suggestion-1",
			CurrentTopicName: "General",
			Title:            "テスト編集提案",
			Body:             "この提案の説明",
			DraftPages: []*model.DraftPage{
				{
					PageID:   pageID,
					Title:    &draftTitle,
					Body:     "提案ページ本文",
					BodyHTML: "<p>提案ページ本文</p>",
				},
			},
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.Suggestion == nil {
			t.Fatal("Suggestion should not be nil")
		}
		if output.Suggestion.Title != "テスト編集提案" {
			t.Errorf("Title = %q, want %q", output.Suggestion.Title, "テスト編集提案")
		}
		if output.Suggestion.Status != model.SuggestionStatusOpen {
			t.Errorf("Status = %d, want %d", output.Suggestion.Status, model.SuggestionStatusOpen)
		}
		if output.Suggestion.Number == 0 {
			t.Error("Number should not be 0")
		}
		if output.Suggestion.BodyHTML == "" {
			t.Error("BodyHTML should not be empty")
		}

		// SuggestionPageが作成されたことを確認
		suggestionPages, err := suggestionPageRepo.ListBySuggestionID(context.Background(), output.Suggestion.ID, spaceID)
		if err != nil {
			t.Fatalf("ListBySuggestionID() error = %v", err)
		}
		if len(suggestionPages) != 1 {
			t.Fatalf("SuggestionPages count = %d, want 1", len(suggestionPages))
		}
		if suggestionPages[0].PageID != pageID {
			t.Errorf("SuggestionPage.PageID = %v, want %v", suggestionPages[0].PageID, pageID)
		}
		if suggestionPages[0].Body != "提案ページ本文" {
			t.Errorf("SuggestionPage.Body = %q, want %q", suggestionPages[0].Body, "提案ページ本文")
		}

		// SuggestionPageRevisionが作成されたことを確認
		revisions, err := suggestionPageRevisionRepo.ListBySuggestionPageID(context.Background(), suggestionPages[0].ID, spaceID)
		if err != nil {
			t.Fatalf("ListBySuggestionPageID() error = %v", err)
		}
		if len(revisions) != 1 {
			t.Fatalf("SuggestionPageRevisions count = %d, want 1", len(revisions))
		}
		if revisions[0].Body != "提案ページ本文" {
			t.Errorf("SuggestionPageRevision.Body = %q, want %q", revisions[0].Body, "提案ページ本文")
		}
	})

	t.Run("正常系: 複数の下書きページから編集提案を作成できる", func(t *testing.T) {
		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("create-suggestion-2").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("create-suggestion-2@example.com").
			WithAtname("createsuggestion2").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("General").
			Build()
		page1ID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("Page 1").
			Build()
		page2ID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(2).
			WithTitle("Page 2").
			Build()

		createPageRevisionViaRepo(t, q, spaceID, spaceMemberID, page1ID)
		createPageRevisionViaRepo(t, q, spaceID, spaceMemberID, page2ID)

		draft1Title := "提案ページ1"
		draft2Title := "提案ページ2"
		output, err := uc.Execute(context.Background(), CreateSuggestionInput{
			SpaceID:          spaceID,
			TopicID:          topicID,
			SpaceMemberID:    spaceMemberID,
			SpaceIdentifier:  "create-suggestion-2",
			CurrentTopicName: "General",
			Title:            "複数ページの提案",
			Body:             "",
			DraftPages: []*model.DraftPage{
				{
					PageID:   page1ID,
					Title:    &draft1Title,
					Body:     "本文1",
					BodyHTML: "<p>本文1</p>",
				},
				{
					PageID:   page2ID,
					Title:    &draft2Title,
					Body:     "本文2",
					BodyHTML: "<p>本文2</p>",
				},
			},
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		suggestionPages, err := suggestionPageRepo.ListBySuggestionID(context.Background(), output.Suggestion.ID, spaceID)
		if err != nil {
			t.Fatalf("ListBySuggestionID() error = %v", err)
		}
		if len(suggestionPages) != 2 {
			t.Errorf("SuggestionPages count = %d, want 2", len(suggestionPages))
		}
	})

	t.Run("異常系: ページリビジョンが存在しない場合はエラー", func(t *testing.T) {
		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("create-suggestion-3").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("create-suggestion-3@example.com").
			WithAtname("createsuggestion3").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("General").
			Build()
		pageID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("No Revision Page").
			WithUnpublished().
			Build()

		draftTitle := "提案"
		_, err := uc.Execute(context.Background(), CreateSuggestionInput{
			SpaceID:          spaceID,
			TopicID:          topicID,
			SpaceMemberID:    spaceMemberID,
			SpaceIdentifier:  "create-suggestion-3",
			CurrentTopicName: "General",
			Title:            "テスト提案",
			Body:             "",
			DraftPages: []*model.DraftPage{
				{
					PageID:   pageID,
					Title:    &draftTitle,
					Body:     "本文",
					BodyHTML: "<p>本文</p>",
				},
			},
		})
		if err == nil {
			t.Error("expected error but got nil")
		}
	})
}
