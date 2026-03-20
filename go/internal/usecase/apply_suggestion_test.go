package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestApplySuggestionUsecase_Execute(t *testing.T) {
	db := testutil.GetTestDB()
	q := query.New(db)

	suggestionRepo := repository.NewSuggestionRepository(q)
	suggestionPageRepo := repository.NewSuggestionPageRepository(q)
	pageRepo := repository.NewPageRepository(q)
	pageRevisionRepo := repository.NewPageRevisionRepository(q)
	pageEditorRepo := repository.NewPageEditorRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	attachmentRepo := repository.NewAttachmentRepository(q)
	pageAttachmentRefRepo := repository.NewPageAttachmentReferenceRepository(q)

	uc := NewApplySuggestionUsecase(db, suggestionRepo, suggestionPageRepo, pageRepo, pageRevisionRepo, pageEditorRepo, topicMemberRepo, attachmentRepo, pageAttachmentRefRepo)

	t.Run("正常系: 1つのページの編集提案を反映できる", func(t *testing.T) {
		t.Parallel()
		// テストデータを作成
		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("apply-suggestion-1").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("apply-suggestion-1@example.com").
			WithAtname("applysuggestion1").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("General").
			Build()
		testutil.NewTopicMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithSpaceMemberID(spaceMemberID).
			Build()
		pageID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("Original Title").
			WithBody("Original body").
			Build()

		// ページリビジョンを作成
		pageRevisionID := testutil.NewPageRevisionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			WithPageID(pageID).
			Build()

		// オープンステータスの編集提案を作成
		suggestionID := testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithStatus(model.SuggestionStatusOpen).
			Build()

		// 編集提案ページを作成
		testutil.NewSuggestionPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithPageID(pageID).
			WithPageRevisionID(pageRevisionID).
			WithTitle("提案タイトル").
			WithBody("提案本文").
			WithBodyHTML("<p>提案本文</p>").
			Build()

		// 反映を実行
		output, err := uc.Execute(context.Background(), ApplySuggestionInput{
			SuggestionID:  suggestionID,
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
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

		// ステータスが反映済みになっていることを確認
		if output.Suggestion.Status != model.SuggestionStatusApplied {
			t.Errorf("Status = %d, want %d", output.Suggestion.Status, model.SuggestionStatusApplied)
		}

		// AppliedAtが設定されていることを確認
		if output.Suggestion.AppliedAt == nil {
			t.Error("AppliedAt should not be nil")
		}

		// ページが更新されていることを確認
		updatedPages, err := pageRepo.FindByIDs(context.Background(), []model.PageID{pageID}, spaceID)
		if err != nil {
			t.Fatalf("FindByIDs() error = %v", err)
		}
		if len(updatedPages) != 1 {
			t.Fatalf("updated pages count = %d, want 1", len(updatedPages))
		}
		updatedPage := updatedPages[0]
		if updatedPage.Body != "提案本文" {
			t.Errorf("Page.Body = %q, want %q", updatedPage.Body, "提案本文")
		}
		wantTitle := "提案タイトル"
		if updatedPage.Title == nil || *updatedPage.Title != wantTitle {
			t.Errorf("Page.Title = %v, want %q", updatedPage.Title, wantTitle)
		}

		// PageRevisionが作成されていることを確認
		latestRevision, err := pageRevisionRepo.FindLatestByPageID(context.Background(), pageID, spaceID)
		if err != nil {
			t.Fatalf("FindLatestByPageID() error = %v", err)
		}
		if latestRevision == nil {
			t.Fatal("latest revision should not be nil")
		}
		if latestRevision.Body != "提案本文" {
			t.Errorf("PageRevision.Body = %q, want %q", latestRevision.Body, "提案本文")
		}
	})

	t.Run("正常系: SuggestionPageのLinkedPageIDsとFeaturedImageAttachmentIDがPageに反映される", func(t *testing.T) {
		t.Parallel()
		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("apply-suggestion-linked").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("apply-suggestion-linked@example.com").
			WithAtname("applysuglinked").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("General").
			Build()
		testutil.NewTopicMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithSpaceMemberID(spaceMemberID).
			Build()
		pageID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("Original Title").
			Build()
		linkedPageID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(2).
			WithTitle("Linked Page").
			Build()
		featuredAttachmentID := testutil.NewAttachmentBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			Build()

		pageRevisionID := testutil.NewPageRevisionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			WithPageID(pageID).
			Build()

		suggestionID := testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithStatus(model.SuggestionStatusOpen).
			Build()

		testutil.NewSuggestionPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithPageID(pageID).
			WithPageRevisionID(pageRevisionID).
			WithTitle("提案タイトル").
			WithBody("提案本文").
			WithBodyHTML("<p>提案本文</p>").
			WithLinkedPageIDs([]model.PageID{linkedPageID}).
			WithFeaturedImageAttachmentID(featuredAttachmentID).
			Build()

		output, err := uc.Execute(context.Background(), ApplySuggestionInput{
			SuggestionID:  suggestionID,
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output.Suggestion.Status != model.SuggestionStatusApplied {
			t.Errorf("Status = %d, want %d", output.Suggestion.Status, model.SuggestionStatusApplied)
		}

		updatedPages, err := pageRepo.FindByIDs(context.Background(), []model.PageID{pageID}, spaceID)
		if err != nil {
			t.Fatalf("FindByIDs() error = %v", err)
		}
		if len(updatedPages) != 1 {
			t.Fatalf("updated pages count = %d, want 1", len(updatedPages))
		}
		updatedPage := updatedPages[0]

		if len(updatedPage.LinkedPageIDs) != 1 || updatedPage.LinkedPageIDs[0] != linkedPageID {
			t.Errorf("Page.LinkedPageIDs = %v, want [%v]", updatedPage.LinkedPageIDs, linkedPageID)
		}
		if updatedPage.FeaturedImageAttachmentID == nil || *updatedPage.FeaturedImageAttachmentID != featuredAttachmentID {
			t.Errorf("Page.FeaturedImageAttachmentID = %v, want %v", updatedPage.FeaturedImageAttachmentID, featuredAttachmentID)
		}
	})

	t.Run("正常系: 複数ページの編集提案を反映できる", func(t *testing.T) {
		t.Parallel()
		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("apply-suggestion-2").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("apply-suggestion-2@example.com").
			WithAtname("applysuggestion2").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("General").
			Build()
		testutil.NewTopicMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithSpaceMemberID(spaceMemberID).
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

		rev1ID := testutil.NewPageRevisionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			WithPageID(page1ID).
			Build()
		rev2ID := testutil.NewPageRevisionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			WithPageID(page2ID).
			Build()

		suggestionID := testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithStatus(model.SuggestionStatusOpen).
			Build()

		testutil.NewSuggestionPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithPageID(page1ID).
			WithPageRevisionID(rev1ID).
			WithTitle("提案ページ1").
			WithBody("提案本文1").
			WithBodyHTML("<p>提案本文1</p>").
			Build()
		testutil.NewSuggestionPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithPageID(page2ID).
			WithPageRevisionID(rev2ID).
			WithTitle("提案ページ2").
			WithBody("提案本文2").
			WithBodyHTML("<p>提案本文2</p>").
			Build()

		output, err := uc.Execute(context.Background(), ApplySuggestionInput{
			SuggestionID:  suggestionID,
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if output.Suggestion.Status != model.SuggestionStatusApplied {
			t.Errorf("Status = %d, want %d", output.Suggestion.Status, model.SuggestionStatusApplied)
		}

		// 両方のページが更新されていることを確認
		updatedPages, err := pageRepo.FindByIDs(context.Background(), []model.PageID{page1ID, page2ID}, spaceID)
		if err != nil {
			t.Fatalf("FindByIDs() error = %v", err)
		}
		if len(updatedPages) != 2 {
			t.Fatalf("updated pages count = %d, want 2", len(updatedPages))
		}

		// 各ページの内容が反映されていることを確認
		pageMap := make(map[model.PageID]*model.Page, len(updatedPages))
		for _, p := range updatedPages {
			pageMap[p.ID] = p
		}
		if pageMap[page1ID].Body != "提案本文1" {
			t.Errorf("Page1.Body = %q, want %q", pageMap[page1ID].Body, "提案本文1")
		}
		if pageMap[page2ID].Body != "提案本文2" {
			t.Errorf("Page2.Body = %q, want %q", pageMap[page2ID].Body, "提案本文2")
		}
	})

	t.Run("異常系: オープンステータス以外の編集提案は反映できない", func(t *testing.T) {
		t.Parallel()
		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("apply-suggestion-3").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("apply-suggestion-3@example.com").
			WithAtname("applysuggestion3").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("General").
			Build()

		// 下書きステータスの編集提案を作成
		suggestionID := testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithStatus(model.SuggestionStatusDraft).
			Build()

		_, err := uc.Execute(context.Background(), ApplySuggestionInput{
			SuggestionID:  suggestionID,
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
		})
		if err == nil {
			t.Error("expected error but got nil")
		}
	})

	t.Run("異常系: クローズ済みの編集提案は反映できない", func(t *testing.T) {
		t.Parallel()
		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("apply-suggestion-4").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("apply-suggestion-4@example.com").
			WithAtname("applysuggestion4").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("General").
			Build()

		suggestionID := testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithStatus(model.SuggestionStatusClosed).
			Build()

		_, err := uc.Execute(context.Background(), ApplySuggestionInput{
			SuggestionID:  suggestionID,
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
		})
		if err == nil {
			t.Error("expected error but got nil")
		}
	})

	t.Run("異常系: 反映済みの編集提案は再度反映できない", func(t *testing.T) {
		t.Parallel()
		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("apply-suggestion-6").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("apply-suggestion-6@example.com").
			WithAtname("applysuggestion6").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("General").
			Build()

		suggestionID := testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithStatus(model.SuggestionStatusApplied).
			Build()

		_, err := uc.Execute(context.Background(), ApplySuggestionInput{
			SuggestionID:  suggestionID,
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
		})
		if err == nil {
			t.Error("expected error but got nil")
		}
	})

	t.Run("異常系: 存在しない編集提案IDではエラーになる", func(t *testing.T) {
		t.Parallel()
		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("apply-suggestion-5").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("apply-suggestion-5@example.com").
			WithAtname("applysuggestion5").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()

		_, err := uc.Execute(context.Background(), ApplySuggestionInput{
			SuggestionID:  "nonexistent-id",
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
		})
		if err == nil {
			t.Error("expected error but got nil")
		}
	})
}
