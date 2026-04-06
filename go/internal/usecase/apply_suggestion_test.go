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

	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	suggestionRepo := repository.NewSuggestionRepository(q)
	suggestionPageRepo := repository.NewSuggestionPageRepository(q)
	pageRepo := repository.NewPageRepository(q)
	pageRevisionRepo := repository.NewPageRevisionRepository(q)
	pageEditorRepo := repository.NewPageEditorRepository(q)
	attachmentRepo := repository.NewAttachmentRepository(q)
	pageAttachmentRefRepo := repository.NewPageAttachmentReferenceRepository(q)

	draftPageRepo := repository.NewDraftPageRepository(q)
	uc := NewApplySuggestionUsecase(
		db, spaceRepo, spaceMemberRepo, topicMemberRepo,
		suggestionRepo, suggestionPageRepo, pageRepo, pageRevisionRepo,
		pageEditorRepo, attachmentRepo, pageAttachmentRefRepo, draftPageRepo,
	)

	t.Run("正常系: 1つのページの編集提案を反映できる", func(t *testing.T) {
		t.Parallel()

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

		suggestionPageID := testutil.NewSuggestionPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithPageID(pageID).
			WithPageRevisionID(pageRevisionID).
			WithTitle("提案タイトル").
			WithBody("提案本文").
			WithBodyHTML("<p>提案本文</p>").
			Build()

		// 編集提案ページにリンクされた下書きを作成
		draftPageID := testutil.NewDraftPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithPageID(pageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithBody("Draft body").
			WithSuggestionPageID(suggestionPageID).
			Build()

		output, err := uc.Execute(context.Background(), ApplySuggestionInput{
			SpaceIdentifier:  "apply-suggestion-1",
			SuggestionNumber: 1,
			UserID:           userID,
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

		if output.Suggestion.Status != model.SuggestionStatusApplied {
			t.Errorf("Status = %d, want %d", output.Suggestion.Status, model.SuggestionStatusApplied)
		}

		if output.Suggestion.AppliedAt == nil {
			t.Error("AppliedAt should not be nil")
		}

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

		// 下書きのsuggestion_page_idがクリアされていることを確認
		dp, err := draftPageRepo.FindByID(context.Background(), draftPageID, spaceID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if dp != nil && dp.SuggestionPageID != nil {
			t.Error("DraftPage.SuggestionPageID should be nil after apply")
		}

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
			SpaceIdentifier:  "apply-suggestion-linked",
			SuggestionNumber: 1,
			UserID:           userID,
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
			SpaceIdentifier:  "apply-suggestion-2",
			SuggestionNumber: 1,
			UserID:           userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if output.Suggestion.Status != model.SuggestionStatusApplied {
			t.Errorf("Status = %d, want %d", output.Suggestion.Status, model.SuggestionStatusApplied)
		}

		updatedPages, err := pageRepo.FindByIDs(context.Background(), []model.PageID{page1ID, page2ID}, spaceID)
		if err != nil {
			t.Fatalf("FindByIDs() error = %v", err)
		}
		if len(updatedPages) != 2 {
			t.Fatalf("updated pages count = %d, want 2", len(updatedPages))
		}

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

	t.Run("正常系: 反映後にPageEditorが作成されLastPageModifiedAtが更新される", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("apply-sug-editor").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("apply-sug-editor@example.com").
			WithAtname("applysugeditor").
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
			WithTitle("Editor Test Page").
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
			WithTitle("エディターテスト").
			WithBody("エディターテスト本文").
			WithBodyHTML("<p>エディターテスト本文</p>").
			Build()

		output, err := uc.Execute(context.Background(), ApplySuggestionInput{
			SpaceIdentifier:  "apply-sug-editor",
			SuggestionNumber: 1,
			UserID:           userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output.Suggestion.Status != model.SuggestionStatusApplied {
			t.Errorf("Status = %d, want %d", output.Suggestion.Status, model.SuggestionStatusApplied)
		}

		// PageEditorが作成されていることを確認
		pe, err := pageEditorRepo.FindByPageAndSpaceMember(context.Background(), repository.FindByPageAndSpaceMemberInput{
			SpaceID:       spaceID,
			PageID:        pageID,
			SpaceMemberID: spaceMemberID,
		})
		if err != nil {
			t.Fatalf("FindByPageAndSpaceMember() error = %v", err)
		}
		if pe == nil {
			t.Fatal("PageEditor should exist after apply")
		}
		if pe.LastPageModifiedAt.IsZero() {
			t.Error("PageEditor.LastPageModifiedAt should not be zero")
		}
	})

	t.Run("正常系: 反映後にTopicMemberのLastPageModifiedAtが更新される", func(t *testing.T) {
		t.Parallel()

		topicMemberRepo := repository.NewTopicMemberRepository(q)

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("apply-sug-tm").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("apply-sug-tm@example.com").
			WithAtname("applysugtm").
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
			WithTitle("TM Test Page").
			Build()
		pageRevisionID := testutil.NewPageRevisionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			WithPageID(pageID).
			Build()

		// 反映前のTopicMemberを取得
		tmBefore, err := topicMemberRepo.FindBySpaceMemberAndTopic(context.Background(), spaceID, spaceMemberID, topicID)
		if err != nil {
			t.Fatalf("FindBySpaceMemberAndTopic() error = %v", err)
		}

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
			WithTitle("TMテスト").
			WithBody("TMテスト本文").
			WithBodyHTML("<p>TMテスト本文</p>").
			Build()

		_, err = uc.Execute(context.Background(), ApplySuggestionInput{
			SpaceIdentifier:  "apply-sug-tm",
			SuggestionNumber: 1,
			UserID:           userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		// 反映後のTopicMemberを取得
		tmAfter, err := topicMemberRepo.FindBySpaceMemberAndTopic(context.Background(), spaceID, spaceMemberID, topicID)
		if err != nil {
			t.Fatalf("FindBySpaceMemberAndTopic() error = %v", err)
		}
		if tmAfter.LastPageModifiedAt == nil {
			t.Fatal("TopicMember.LastPageModifiedAt should not be nil after apply")
		}
		if tmBefore.LastPageModifiedAt != nil && !tmAfter.LastPageModifiedAt.After(*tmBefore.LastPageModifiedAt) {
			t.Error("TopicMember.LastPageModifiedAt should be updated after apply")
		}
	})

	t.Run("正常系: 新規ページ（リビジョンなし）を含む編集提案を反映できる", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("apply-sug-newpage").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("apply-sug-newpage@example.com").
			WithAtname("applysugnewpage").
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

		// published_atがnilのページ（未公開）を作成
		pageID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("Unpublished Page").
			WithUnpublished().
			Build()

		suggestionID := testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithStatus(model.SuggestionStatusOpen).
			Build()

		// page_revision_idがnilのSuggestionPage
		testutil.NewSuggestionPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithPageID(pageID).
			WithTitle("新規ページタイトル").
			WithBody("新規ページ本文").
			WithBodyHTML("<p>新規ページ本文</p>").
			Build()

		output, err := uc.Execute(context.Background(), ApplySuggestionInput{
			SpaceIdentifier:  "apply-sug-newpage",
			SuggestionNumber: 1,
			UserID:           userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output.Suggestion.Status != model.SuggestionStatusApplied {
			t.Errorf("Status = %d, want %d", output.Suggestion.Status, model.SuggestionStatusApplied)
		}

		// ページ内容が更新されていることを確認
		updatedPages, err := pageRepo.FindByIDs(context.Background(), []model.PageID{pageID}, spaceID)
		if err != nil {
			t.Fatalf("FindByIDs() error = %v", err)
		}
		if len(updatedPages) != 1 {
			t.Fatalf("updated pages count = %d, want 1", len(updatedPages))
		}
		updatedPage := updatedPages[0]
		if updatedPage.Body != "新規ページ本文" {
			t.Errorf("Page.Body = %q, want %q", updatedPage.Body, "新規ページ本文")
		}
		wantTitle := "新規ページタイトル"
		if updatedPage.Title == nil || *updatedPage.Title != wantTitle {
			t.Errorf("Page.Title = %v, want %q", updatedPage.Title, wantTitle)
		}

		// published_atが設定されていることを確認
		if updatedPage.PublishedAt == nil {
			t.Error("Page.PublishedAt should not be nil after apply")
		}

		// ページリビジョンが作成されていることを確認
		latestRevision, err := pageRevisionRepo.FindLatestByPageID(context.Background(), pageID, spaceID)
		if err != nil {
			t.Fatalf("FindLatestByPageID() error = %v", err)
		}
		if latestRevision == nil {
			t.Fatal("latest revision should not be nil")
		}
		if latestRevision.Body != "新規ページ本文" {
			t.Errorf("PageRevision.Body = %q, want %q", latestRevision.Body, "新規ページ本文")
		}
	})

	t.Run("正常系: ベースリビジョン乖離時でも上書き反映できる", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("apply-sug-diverge").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("apply-sug-diverge@example.com").
			WithAtname("applysugdiverge").
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

		// ページ作成 → リビジョンv1作成
		pageID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("Original v1").
			WithBody("Original body v1").
			Build()
		revV1ID := testutil.NewPageRevisionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			WithPageID(pageID).
			Build()

		// 編集提案作成（v1がベース）
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
			WithPageRevisionID(revV1ID).
			WithTitle("提案によるタイトル").
			WithBody("提案による本文").
			WithBodyHTML("<p>提案による本文</p>").
			Build()

		// ページを直接更新（v2） - ベースリビジョンが乖離する
		_, err := db.Exec(
			`UPDATE pages SET title = $1, body = $2, body_html = $3 WHERE id = $4 AND space_id = $5`,
			"Updated v2", "Updated body v2", "<p>Updated body v2</p>", string(pageID), string(spaceID),
		)
		if err != nil {
			t.Fatalf("ページの直接更新に失敗: %v", err)
		}
		// リビジョンv2を作成
		testutil.NewPageRevisionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			WithPageID(pageID).
			Build()

		// 編集提案を反映（ベースリビジョンが乖離しているが上書き反映される）
		output, err := uc.Execute(context.Background(), ApplySuggestionInput{
			SpaceIdentifier:  "apply-sug-diverge",
			SuggestionNumber: 1,
			UserID:           userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output.Suggestion.Status != model.SuggestionStatusApplied {
			t.Errorf("Status = %d, want %d", output.Suggestion.Status, model.SuggestionStatusApplied)
		}

		// ページ内容が編集提案の内容で上書きされていることを確認（v2ではなく提案内容）
		updatedPages, err := pageRepo.FindByIDs(context.Background(), []model.PageID{pageID}, spaceID)
		if err != nil {
			t.Fatalf("FindByIDs() error = %v", err)
		}
		if len(updatedPages) != 1 {
			t.Fatalf("updated pages count = %d, want 1", len(updatedPages))
		}
		updatedPage := updatedPages[0]
		if updatedPage.Body != "提案による本文" {
			t.Errorf("Page.Body = %q, want %q", updatedPage.Body, "提案による本文")
		}
		wantTitle := "提案によるタイトル"
		if updatedPage.Title == nil || *updatedPage.Title != wantTitle {
			t.Errorf("Page.Title = %v, want %q", updatedPage.Title, wantTitle)
		}
	})

	t.Run("認可エラー: スペースメンバーでないユーザーはForbiddenエラーが返る", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("apply-sug-auth-1").
			Build()
		ownerID := testutil.NewUserBuilderDB(t, db).
			WithEmail("apply-sug-auth-owner@example.com").
			WithAtname("applysugauthown").
			Build()
		nonMemberID := testutil.NewUserBuilderDB(t, db).
			WithEmail("apply-sug-auth-non@example.com").
			WithAtname("applysugauthnon").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(ownerID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("General").
			Build()
		testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithStatus(model.SuggestionStatusOpen).
			Build()

		_, err := uc.Execute(context.Background(), ApplySuggestionInput{
			SpaceIdentifier:  "apply-sug-auth-1",
			SuggestionNumber: 1,
			UserID:           nonMemberID,
		})
		if err == nil {
			t.Fatal("expected error but got nil")
		}

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatalf("expected AppError but got %T: %v", err, err)
		}
		if ae.Code != model.AppErrCodeForbidden {
			t.Errorf("error code = %d, want %d", ae.Code, model.AppErrCodeForbidden)
		}
	})

	t.Run("ステータスエラー: クローズ済みの編集提案はConflictエラーが返る", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("apply-sug-closed").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("apply-sug-closed@example.com").
			WithAtname("applysugclosed").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("General").
			Build()
		testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithStatus(model.SuggestionStatusClosed).
			Build()

		_, err := uc.Execute(context.Background(), ApplySuggestionInput{
			SpaceIdentifier:  "apply-sug-closed",
			SuggestionNumber: 1,
			UserID:           userID,
		})
		if err == nil {
			t.Fatal("expected error but got nil")
		}

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatalf("expected AppError but got %T: %v", err, err)
		}
		if ae.Code != model.AppErrCodeConflict {
			t.Errorf("error code = %d, want %d", ae.Code, model.AppErrCodeConflict)
		}
	})

	t.Run("べき等性: 反映済みの編集提案は成功出力を返す", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("apply-sug-idem").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("apply-sug-idem@example.com").
			WithAtname("applysugidem").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("General").
			Build()
		testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithStatus(model.SuggestionStatusApplied).
			Build()

		output, err := uc.Execute(context.Background(), ApplySuggestionInput{
			SpaceIdentifier:  "apply-sug-idem",
			SuggestionNumber: 1,
			UserID:           userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.Suggestion.Status != model.SuggestionStatusApplied {
			t.Errorf("Status = %d, want %d", output.Suggestion.Status, model.SuggestionStatusApplied)
		}
	})
}
