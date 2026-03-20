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

	rev := createPageRevisionForTest(t, q, spaceID, spaceMemberID, pageID)
	return rev.ID
}

// createPageRevisionForTest はリポジトリ経由でページリビジョンを作成し、モデルを返すヘルパー
func createPageRevisionForTest(t *testing.T, q *query.Queries, spaceID model.SpaceID, spaceMemberID model.SpaceMemberID, pageID model.PageID) *model.PageRevision {
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
	return rev
}

func TestCreateSuggestionUsecase_Execute(t *testing.T) {
	db := testutil.GetTestDB()
	q := query.New(db)

	suggestionRepo := repository.NewSuggestionRepository(q)
	suggestionPageRepo := repository.NewSuggestionPageRepository(q)
	suggestionPageRevisionRepo := repository.NewSuggestionPageRevisionRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)

	uc := NewCreateSuggestionUsecase(db, suggestionRepo, suggestionPageRepo, suggestionPageRevisionRepo, draftPageRepo)

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
		pageRevision := createPageRevisionForTest(t, q, spaceID, spaceMemberID, pageID)

		draftTitle := "提案タイトル"
		output, err := uc.Execute(context.Background(), CreateSuggestionInput{
			SpaceID:       spaceID,
			TopicID:       topicID,
			SpaceMemberID: spaceMemberID,
			Title:         "テスト編集提案",
			Body:          "この提案の説明",
			BodyHTML:      "<p>この提案の説明</p>",
			DraftPages: []*model.DraftPage{
				{
					PageID:   pageID,
					Title:    &draftTitle,
					Body:     "提案ページ本文",
					BodyHTML: "<p>提案ページ本文</p>",
				},
			},
			PageRevisions: map[model.PageID]*model.PageRevision{
				pageID: pageRevision,
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

	t.Run("正常系: DraftPageのLinkedPageIDsとFeaturedImageAttachmentIDがSuggestionPageにコピーされる", func(t *testing.T) {
		t.Parallel()
		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("create-suggestion-linked").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("create-suggestion-linked@example.com").
			WithAtname("createsuglinked").
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
		linkedPageID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(2).
			WithTitle("Linked Page").
			Build()

		pageRevision := createPageRevisionForTest(t, q, spaceID, spaceMemberID, pageID)

		featuredID := testutil.NewAttachmentBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			Build()
		draftTitle := "提案タイトル"
		output, err := uc.Execute(context.Background(), CreateSuggestionInput{
			SpaceID:       spaceID,
			TopicID:       topicID,
			SpaceMemberID: spaceMemberID,
			Title:         "リンク付き提案",
			Body:          "",
			BodyHTML:      "",
			DraftPages: []*model.DraftPage{
				{
					PageID:                    pageID,
					Title:                     &draftTitle,
					Body:                      "本文",
					BodyHTML:                  "<p>本文</p>",
					LinkedPageIDs:             []model.PageID{linkedPageID},
					FeaturedImageAttachmentID: &featuredID,
				},
			},
			PageRevisions: map[model.PageID]*model.PageRevision{
				pageID: pageRevision,
			},
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		suggestionPages, err := suggestionPageRepo.ListBySuggestionID(context.Background(), output.Suggestion.ID, spaceID)
		if err != nil {
			t.Fatalf("ListBySuggestionID() error = %v", err)
		}
		if len(suggestionPages) != 1 {
			t.Fatalf("SuggestionPages count = %d, want 1", len(suggestionPages))
		}

		sp := suggestionPages[0]
		if len(sp.LinkedPageIDs) != 1 || sp.LinkedPageIDs[0] != linkedPageID {
			t.Errorf("SuggestionPage.LinkedPageIDs = %v, want [%v]", sp.LinkedPageIDs, linkedPageID)
		}
		if sp.FeaturedImageAttachmentID == nil || *sp.FeaturedImageAttachmentID != featuredID {
			t.Errorf("SuggestionPage.FeaturedImageAttachmentID = %v, want %v", sp.FeaturedImageAttachmentID, featuredID)
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

		page1Revision := createPageRevisionForTest(t, q, spaceID, spaceMemberID, page1ID)
		page2Revision := createPageRevisionForTest(t, q, spaceID, spaceMemberID, page2ID)

		draft1Title := "提案ページ1"
		draft2Title := "提案ページ2"
		output, err := uc.Execute(context.Background(), CreateSuggestionInput{
			SpaceID:       spaceID,
			TopicID:       topicID,
			SpaceMemberID: spaceMemberID,
			Title:         "複数ページの提案",
			Body:          "",
			BodyHTML:      "",
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
			PageRevisions: map[model.PageID]*model.PageRevision{
				page1ID: page1Revision,
				page2ID: page2Revision,
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

	t.Run("正常系: 編集提案作成後にDraftPageのsuggestion_page_idが設定される", func(t *testing.T) {
		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("create-suggestion-spid").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("create-suggestion-spid@example.com").
			WithAtname("createsugspid").
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

		pageRevision := createPageRevisionForTest(t, q, spaceID, spaceMemberID, pageID)

		// 実際のDraftPageをDBに作成
		draftPageID := testutil.NewDraftPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithPageID(pageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithTitle("提案タイトル").
			WithBody("提案ページ本文").
			WithBodyHTML("<p>提案ページ本文</p>").
			Build()

		draftTitle := "提案タイトル"
		output, err := uc.Execute(context.Background(), CreateSuggestionInput{
			SpaceID:       spaceID,
			TopicID:       topicID,
			SpaceMemberID: spaceMemberID,
			Title:         "テスト編集提案",
			Body:          "",
			BodyHTML:      "",
			DraftPages: []*model.DraftPage{
				{
					ID:       draftPageID,
					PageID:   pageID,
					Title:    &draftTitle,
					Body:     "提案ページ本文",
					BodyHTML: "<p>提案ページ本文</p>",
				},
			},
			PageRevisions: map[model.PageID]*model.PageRevision{
				pageID: pageRevision,
			},
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		// SuggestionPageのIDを取得
		suggestionPages, err := suggestionPageRepo.ListBySuggestionID(context.Background(), output.Suggestion.ID, spaceID)
		if err != nil {
			t.Fatalf("ListBySuggestionID() error = %v", err)
		}
		if len(suggestionPages) != 1 {
			t.Fatalf("SuggestionPages count = %d, want 1", len(suggestionPages))
		}

		// DraftPageのsuggestion_page_idが設定されていることを確認
		updatedDraftPage, err := draftPageRepo.FindByID(context.Background(), draftPageID, spaceID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if updatedDraftPage == nil {
			t.Fatal("updatedDraftPage should not be nil")
		}
		if updatedDraftPage.SuggestionPageID == nil {
			t.Fatal("DraftPage.SuggestionPageID should not be nil after suggestion creation")
		}
		if *updatedDraftPage.SuggestionPageID != suggestionPages[0].ID {
			t.Errorf("DraftPage.SuggestionPageID = %v, want %v", *updatedDraftPage.SuggestionPageID, suggestionPages[0].ID)
		}
	})
}
