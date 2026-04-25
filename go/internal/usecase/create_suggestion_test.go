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
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)

	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	suggestionRepo := repository.NewSuggestionRepository(q)
	suggestionPageRepo := repository.NewSuggestionPageRepository(q)
	suggestionPageRevisionRepo := repository.NewSuggestionPageRevisionRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	pageRepo := repository.NewPageRepository(q)
	pageRevisionRepo := repository.NewPageRevisionRepository(q)
	createValidator := validator.NewSuggestionCreateValidator(draftPageRepo, pageRepo)

	uc := NewCreateSuggestionUsecase(db, spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, suggestionRepo, suggestionPageRepo, suggestionPageRevisionRepo, draftPageRepo, pageRevisionRepo, createValidator)

	t.Run("正常系: 1つの下書きページから編集提案を作成できる", func(t *testing.T) {
		t.Parallel()

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

		createPageRevisionForTest(t, q, spaceID, spaceMemberID, pageID)

		draftPageID := testutil.NewDraftPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithPageID(pageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithTitle("提案タイトル").
			WithBody("提案ページ本文").
			WithBodyHTML("<p>提案ページ本文</p>").
			Build()

		output, err := uc.Execute(context.Background(), CreateSuggestionInput{
			SpaceIdentifier: "create-suggestion-1",
			TopicNumber:     1,
			UserID:          userID,
			Title:           "テスト編集提案",
			Body:            "この提案の説明",
			DraftPageIDs:    []model.DraftPageID{draftPageID},
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

		createPageRevisionForTest(t, q, spaceID, spaceMemberID, pageID)

		featuredID := testutil.NewAttachmentBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			Build()
		draftPageID := testutil.NewDraftPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithPageID(pageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithTitle("提案タイトル").
			WithBody("本文").
			WithBodyHTML("<p>本文</p>").
			WithLinkedPageIDs([]model.PageID{linkedPageID}).
			WithFeaturedImageAttachmentID(featuredID).
			Build()

		output, err := uc.Execute(context.Background(), CreateSuggestionInput{
			SpaceIdentifier: "create-suggestion-linked",
			TopicNumber:     1,
			UserID:          userID,
			Title:           "リンク付き提案",
			Body:            "",
			DraftPageIDs:    []model.DraftPageID{draftPageID},
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
		t.Parallel()

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

		createPageRevisionForTest(t, q, spaceID, spaceMemberID, page1ID)
		createPageRevisionForTest(t, q, spaceID, spaceMemberID, page2ID)

		draftPage1ID := testutil.NewDraftPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithPageID(page1ID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithTitle("提案ページ1").
			WithBody("本文1").
			WithBodyHTML("<p>本文1</p>").
			Build()
		draftPage2ID := testutil.NewDraftPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithPageID(page2ID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithTitle("提案ページ2").
			WithBody("本文2").
			WithBodyHTML("<p>本文2</p>").
			Build()

		output, err := uc.Execute(context.Background(), CreateSuggestionInput{
			SpaceIdentifier: "create-suggestion-2",
			TopicNumber:     1,
			UserID:          userID,
			Title:           "複数ページの提案",
			Body:            "",
			DraftPageIDs:    []model.DraftPageID{draftPage1ID, draftPage2ID},
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
		t.Parallel()

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

		createPageRevisionForTest(t, q, spaceID, spaceMemberID, pageID)

		draftPageID := testutil.NewDraftPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithPageID(pageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithTitle("提案タイトル").
			WithBody("提案ページ本文").
			WithBodyHTML("<p>提案ページ本文</p>").
			Build()

		output, err := uc.Execute(context.Background(), CreateSuggestionInput{
			SpaceIdentifier: "create-suggestion-spid",
			TopicNumber:     1,
			UserID:          userID,
			Title:           "テスト編集提案",
			Body:            "",
			DraftPageIDs:    []model.DraftPageID{draftPageID},
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

	t.Run("異常系: 存在しないスペースの場合AppErrorが返る", func(t *testing.T) {
		t.Parallel()

		_, err := uc.Execute(context.Background(), CreateSuggestionInput{
			SpaceIdentifier: "nonexistent-space-cs",
			TopicNumber:     1,
			UserID:          "dummy-user",
			Title:           "テスト",
			Body:            "",
			DraftPageIDs:    []model.DraftPageID{"dummy"},
		})

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatal("expected AppError, got nil")
		}
		if ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("Code = %d, want %d", ae.Code, model.AppErrCodeResourceNotFound)
		}
	})

	t.Run("異常系: 存在しないトピックの場合AppErrorが返る", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("create-sug-notopic").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("create-sug-notopic@example.com").
			WithAtname("createsugnotopic").
			Build()
		testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()

		_, err := uc.Execute(context.Background(), CreateSuggestionInput{
			SpaceIdentifier: "create-sug-notopic",
			TopicNumber:     999,
			UserID:          userID,
			Title:           "テスト",
			Body:            "",
			DraftPageIDs:    []model.DraftPageID{"dummy"},
		})

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatal("expected AppError, got nil")
		}
		if ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("Code = %d, want %d", ae.Code, model.AppErrCodeResourceNotFound)
		}
	})

	t.Run("異常系: 非メンバーの場合AppError(Forbidden)が返る", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("create-sug-noauth").
			Build()
		testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithNumber(1).
			WithName("General").
			Build()
		nonMemberID := testutil.NewUserBuilderDB(t, db).
			WithEmail("create-sug-noauth@example.com").
			WithAtname("createsugnoauth").
			Build()

		_, err := uc.Execute(context.Background(), CreateSuggestionInput{
			SpaceIdentifier: "create-sug-noauth",
			TopicNumber:     1,
			UserID:          nonMemberID,
			Title:           "テスト",
			Body:            "",
			DraftPageIDs:    []model.DraftPageID{"dummy"},
		})

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatal("expected AppError, got nil")
		}
		if ae.Code != model.AppErrCodeForbidden {
			t.Errorf("Code = %d, want %d", ae.Code, model.AppErrCodeForbidden)
		}
	})

	t.Run("異常系: タイトル未入力の場合ValidationErrorが返る", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("create-sug-notitle").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("create-sug-notitle@example.com").
			WithAtname("createsugnotitle").
			Build()
		testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithNumber(1).
			WithName("General").
			Build()

		_, err := uc.Execute(context.Background(), CreateSuggestionInput{
			SpaceIdentifier: "create-sug-notitle",
			TopicNumber:     1,
			UserID:          userID,
			Title:           "",
			Body:            "",
			DraftPageIDs:    []model.DraftPageID{"dummy"},
		})

		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatal("expected ValidationError, got nil")
		}
		if !ve.HasFieldError("title") {
			t.Error("expected title field error")
		}
	})

	t.Run("正常系: 新規ページ（リビジョンなし）の下書きで編集提案を作成できる", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("create-sug-norev").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("create-sug-norev@example.com").
			WithAtname("createsugnorev").
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

		draftPageID := testutil.NewDraftPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithPageID(pageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithTitle("タイトル").
			WithBody("本文").
			WithBodyHTML("<p>本文</p>").
			Build()

		output, err := uc.Execute(context.Background(), CreateSuggestionInput{
			SpaceIdentifier: "create-sug-norev",
			TopicNumber:     1,
			UserID:          userID,
			Title:           "テスト",
			Body:            "",
			DraftPageIDs:    []model.DraftPageID{draftPageID},
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil || output.Suggestion == nil {
			t.Fatal("output or Suggestion should not be nil")
		}

		// SuggestionPageのPageRevisionIDがnilであることを確認
		suggestionPageRepo := repository.NewSuggestionPageRepository(q)
		pages, err := suggestionPageRepo.ListBySuggestionID(context.Background(), output.Suggestion.ID, spaceID)
		if err != nil {
			t.Fatalf("ListBySuggestionID() error = %v", err)
		}
		if len(pages) != 1 {
			t.Fatalf("SuggestionPages count = %d, want 1", len(pages))
		}
		if pages[0].PageRevisionID != nil {
			t.Errorf("SuggestionPage.PageRevisionID = %v, want nil", pages[0].PageRevisionID)
		}
	})
}
