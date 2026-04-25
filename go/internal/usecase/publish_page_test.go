package usecase

import (
	"context"
	"fmt"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestPublishPageUsecase_Execute(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	pageRepo := repository.NewPageRepository(q)
	pageRevisionRepo := repository.NewPageRevisionRepository(q)
	pageEditorRepo := repository.NewPageEditorRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	draftPageRevisionRepo := repository.NewDraftPageRevisionRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	attachmentRepo := repository.NewAttachmentRepository(q)
	pageAttachmentRefRepo := repository.NewPageAttachmentReferenceRepository(q)
	pageUpdateValidator := validator.NewPageUpdateValidator(pageRepo)
	uc := NewPublishPageUsecase(db, spaceRepo, spaceMemberRepo, pageRepo, pageRevisionRepo, pageEditorRepo, draftPageRepo, draftPageRevisionRepo, topicRepo, topicMemberRepo, attachmentRepo, pageAttachmentRefRepo, pageUpdateValidator)

	// テストデータを作成
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("publish-test").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("publish-test@example.com").
		WithAtname("publishtest").
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
		WithTitle("Test Page").
		Build()
	testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Updated Title").
		WithBody("Updated body").
		Build()

	output, err := uc.Execute(context.Background(), PublishPageInput{
		SpaceIdentifier: model.SpaceIdentifier("publish-test"),
		PageNumber:      1,
		UserID:          userID,
		Title:           "Updated Title",
		Body:            "Updated body",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if output == nil {
		t.Fatal("output should not be nil")
	}
	if output.Page == nil {
		t.Fatal("Page should not be nil")
	}
	if output.Page.Body != "Updated body" {
		t.Errorf("Body = %q, want %q", output.Page.Body, "Updated body")
	}
	if output.Page.Title == nil || *output.Page.Title != "Updated Title" {
		t.Errorf("Title = %v, want %q", output.Page.Title, "Updated Title")
	}
	if output.Page.BodyHTML == "" {
		t.Error("BodyHTML should not be empty")
	}
	if output.PublishedAt.IsZero() {
		t.Error("PublishedAt should not be zero")
	}
	if output.Page.PublishedAt == nil {
		t.Error("Page.PublishedAt should not be nil")
	}
}

func TestPublishPageUsecase_Execute_WithWikilinks(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	pageRepo := repository.NewPageRepository(q)
	pageRevisionRepo := repository.NewPageRevisionRepository(q)
	pageEditorRepo := repository.NewPageEditorRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	draftPageRevisionRepo := repository.NewDraftPageRevisionRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	attachmentRepo := repository.NewAttachmentRepository(q)
	pageAttachmentRefRepo := repository.NewPageAttachmentReferenceRepository(q)
	pageUpdateValidator := validator.NewPageUpdateValidator(pageRepo)
	uc := NewPublishPageUsecase(db, spaceRepo, spaceMemberRepo, pageRepo, pageRevisionRepo, pageEditorRepo, draftPageRepo, draftPageRevisionRepo, topicRepo, topicMemberRepo, attachmentRepo, pageAttachmentRefRepo, pageUpdateValidator)

	// テストデータを作成
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("publish-wikilink").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("publish-wikilink@example.com").
		WithAtname("publishwikilink").
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
		WithTitle("Test Page").
		Build()
	testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Wikilink Publish Test").
		WithBody("See [[リンク先ページ]]").
		Build()

	output, err := uc.Execute(context.Background(), PublishPageInput{
		SpaceIdentifier: model.SpaceIdentifier("publish-wikilink"),
		PageNumber:      1,
		UserID:          userID,
		Title:           "Wikilink Publish Test",
		Body:            "See [[リンク先ページ]]",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	// リンク先ページが自動作成され、LinkedPageIDsに含まれることを確認
	if len(output.Page.LinkedPageIDs) == 0 {
		t.Error("LinkedPageIDs should not be empty")
	}

	// bodyHTMLにリンクが含まれることを確認
	if output.Page.BodyHTML == "" {
		t.Error("BodyHTML should not be empty")
	}

	// PublishedAtが設定されていることを確認
	if output.Page.PublishedAt == nil {
		t.Error("Page.PublishedAt should not be nil")
	}
}

func TestPublishPageUsecase_Execute_NilTitle(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	pageRepo := repository.NewPageRepository(q)
	pageRevisionRepo := repository.NewPageRevisionRepository(q)
	pageEditorRepo := repository.NewPageEditorRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	draftPageRevisionRepo := repository.NewDraftPageRevisionRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	attachmentRepo := repository.NewAttachmentRepository(q)
	pageAttachmentRefRepo := repository.NewPageAttachmentReferenceRepository(q)
	pageUpdateValidator := validator.NewPageUpdateValidator(pageRepo)
	uc := NewPublishPageUsecase(db, spaceRepo, spaceMemberRepo, pageRepo, pageRevisionRepo, pageEditorRepo, draftPageRepo, draftPageRevisionRepo, topicRepo, topicMemberRepo, attachmentRepo, pageAttachmentRefRepo, pageUpdateValidator)

	// テストデータを作成
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("publish-niltitle").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("publish-niltitle@example.com").
		WithAtname("publishniltitle").
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
		WithTitle("Test Page").
		Build()
	testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithBody("Body without title").
		Build()

	_, err := uc.Execute(context.Background(), PublishPageInput{
		SpaceIdentifier: model.SpaceIdentifier("publish-niltitle"),
		PageNumber:      1,
		UserID:          userID,
		Title:           "",
		Body:            "Body without title",
	})
	if err == nil {
		t.Fatal("Execute() should return error for empty title")
	}

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatalf("expected ValidationError but got: %v", err)
	}
	if !ve.HasFieldError("title") {
		t.Error("expected title field error")
	}
}

func TestPublishPageUsecase_Execute_ExistingLinkedPage(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	pageRepo := repository.NewPageRepository(q)
	pageRevisionRepo := repository.NewPageRevisionRepository(q)
	pageEditorRepo := repository.NewPageEditorRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	draftPageRevisionRepo := repository.NewDraftPageRevisionRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	attachmentRepo := repository.NewAttachmentRepository(q)
	pageAttachmentRefRepo := repository.NewPageAttachmentReferenceRepository(q)
	pageUpdateValidator := validator.NewPageUpdateValidator(pageRepo)
	uc := NewPublishPageUsecase(db, spaceRepo, spaceMemberRepo, pageRepo, pageRevisionRepo, pageEditorRepo, draftPageRepo, draftPageRevisionRepo, topicRepo, topicMemberRepo, attachmentRepo, pageAttachmentRefRepo, pageUpdateValidator)

	// テストデータを作成
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("publish-existing-link").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("publish-existing-link@example.com").
		WithAtname("publishexistinglink").
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
		WithTitle("Test Page").
		Build()

	// リンク先ページを事前に作成
	existingPageID := testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("既存ページ").
		Build()

	testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Existing Link Test").
		WithBody("See [[既存ページ]]").
		Build()

	output, err := uc.Execute(context.Background(), PublishPageInput{
		SpaceIdentifier: model.SpaceIdentifier("publish-existing-link"),
		PageNumber:      1,
		UserID:          userID,
		Title:           "Existing Link Test",
		Body:            "See [[既存ページ]]",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	// LinkedPageIDsに既存ページのIDが含まれることを確認
	if len(output.Page.LinkedPageIDs) != 1 {
		t.Fatalf("LinkedPageIDs length = %d, want 1", len(output.Page.LinkedPageIDs))
	}
	if output.Page.LinkedPageIDs[0] != existingPageID {
		t.Errorf("LinkedPageIDs[0] = %v, want %v", output.Page.LinkedPageIDs[0], existingPageID)
	}
}

func TestPublishPageUsecase_Execute_WikilinkCreatesPageEditor(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	pageRepo := repository.NewPageRepository(q)
	pageRevisionRepo := repository.NewPageRevisionRepository(q)
	pageEditorRepo := repository.NewPageEditorRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	draftPageRevisionRepo := repository.NewDraftPageRevisionRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	attachmentRepo := repository.NewAttachmentRepository(q)
	pageAttachmentRefRepo := repository.NewPageAttachmentReferenceRepository(q)
	pageUpdateValidator := validator.NewPageUpdateValidator(pageRepo)
	uc := NewPublishPageUsecase(db, spaceRepo, spaceMemberRepo, pageRepo, pageRevisionRepo, pageEditorRepo, draftPageRepo, draftPageRevisionRepo, topicRepo, topicMemberRepo, attachmentRepo, pageAttachmentRefRepo, pageUpdateValidator)

	// テストデータを作成
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("publish-editor").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("publish-editor@example.com").
		WithAtname("publisheditor").
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
		WithTitle("Test Page").
		Build()
	testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Editor Publish Test").
		WithBody("See [[公開時自動作成ページ]]").
		Build()

	output, err := uc.Execute(context.Background(), PublishPageInput{
		SpaceIdentifier: model.SpaceIdentifier("publish-editor"),
		PageNumber:      1,
		UserID:          userID,
		Title:           "Editor Publish Test",
		Body:            "See [[公開時自動作成ページ]]",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	// 自動作成されたページのIDを取得
	if len(output.Page.LinkedPageIDs) == 0 {
		t.Fatal("LinkedPageIDs should not be empty")
	}
	linkedPageID := output.Page.LinkedPageIDs[0]

	// 自動作成されたページにpage_editorsが作成されていることを確認
	editor, err := pageEditorRepo.FindByPageAndSpaceMember(context.Background(), repository.FindByPageAndSpaceMemberInput{
		SpaceID:       spaceID,
		PageID:        linkedPageID,
		SpaceMemberID: spaceMemberID,
	})
	if err != nil {
		t.Fatalf("自動作成ページのpage_editorsが見つかりません: %v", err)
	}
	if editor.PageID != linkedPageID {
		t.Errorf("PageID = %v, want %v", editor.PageID, linkedPageID)
	}
	if editor.SpaceMemberID != spaceMemberID {
		t.Errorf("SpaceMemberID = %v, want %v", editor.SpaceMemberID, spaceMemberID)
	}
}

func TestPublishPageUsecase_Execute_WikilinkDiscardedPage(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	pageRepo := repository.NewPageRepository(q)
	pageRevisionRepo := repository.NewPageRevisionRepository(q)
	pageEditorRepo := repository.NewPageEditorRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	draftPageRevisionRepo := repository.NewDraftPageRevisionRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	attachmentRepo := repository.NewAttachmentRepository(q)
	pageAttachmentRefRepo := repository.NewPageAttachmentReferenceRepository(q)
	pageUpdateValidator := validator.NewPageUpdateValidator(pageRepo)
	uc := NewPublishPageUsecase(db, spaceRepo, spaceMemberRepo, pageRepo, pageRevisionRepo, pageEditorRepo, draftPageRepo, draftPageRevisionRepo, topicRepo, topicMemberRepo, attachmentRepo, pageAttachmentRefRepo, pageUpdateValidator)

	// テストデータを作成
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("publish-discarded").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("publish-discarded@example.com").
		WithAtname("publishdiscarded").
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
		WithTitle("Test Page").
		Build()

	// 廃棄済みページを作成
	discardedPageID := testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("廃棄済みページ").
		WithDiscarded().
		Build()

	testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Discarded Link Test").
		WithBody("See [[廃棄済みページ]]").
		Build()

	output, err := uc.Execute(context.Background(), PublishPageInput{
		SpaceIdentifier: model.SpaceIdentifier("publish-discarded"),
		PageNumber:      1,
		UserID:          userID,
		Title:           "Discarded Link Test",
		Body:            "See [[廃棄済みページ]]",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	// LinkedPageIDsに廃棄済みページのIDが含まれることを確認
	if len(output.Page.LinkedPageIDs) != 1 {
		t.Fatalf("LinkedPageIDs length = %d, want 1", len(output.Page.LinkedPageIDs))
	}
	if output.Page.LinkedPageIDs[0] != discardedPageID {
		t.Errorf("LinkedPageIDs[0] = %v, want %v", output.Page.LinkedPageIDs[0], discardedPageID)
	}
}

func TestPublishPageUsecase_Execute_WithAttachments(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	pageRepo := repository.NewPageRepository(q)
	pageRevisionRepo := repository.NewPageRevisionRepository(q)
	pageEditorRepo := repository.NewPageEditorRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	draftPageRevisionRepo := repository.NewDraftPageRevisionRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	attachmentRepo := repository.NewAttachmentRepository(q)
	pageAttachmentRefRepo := repository.NewPageAttachmentReferenceRepository(q)
	pageUpdateValidator := validator.NewPageUpdateValidator(pageRepo)
	uc := NewPublishPageUsecase(db, spaceRepo, spaceMemberRepo, pageRepo, pageRevisionRepo, pageEditorRepo, draftPageRepo, draftPageRevisionRepo, topicRepo, topicMemberRepo, attachmentRepo, pageAttachmentRefRepo, pageUpdateValidator)

	// テストデータを作成
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("publish-attach").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("publish-attach@example.com").
		WithAtname("publishattach").
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

	// 添付ファイルを作成
	attachmentID := testutil.NewAttachmentBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		WithFilename("image1.png").
		Build()

	pageID := testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Attachment Test").
		Build()

	body := fmt.Sprintf("![image](/attachments/%s)", attachmentID)
	testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Attachment Test").
		WithBody(body).
		Build()

	output, err := uc.Execute(context.Background(), PublishPageInput{
		SpaceIdentifier: model.SpaceIdentifier("publish-attach"),
		PageNumber:      1,
		UserID:          userID,
		Title:           "Attachment Test",
		Body:            body,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	// 添付ファイル参照が作成されていることを確認
	refs, err := pageAttachmentRefRepo.ListByPageID(context.Background(), pageID, spaceID)
	if err != nil {
		t.Fatalf("ListByPageID() error = %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("PageAttachmentReferences count = %d, want 1", len(refs))
	}
	if refs[0].AttachmentID != attachmentID {
		t.Errorf("AttachmentID = %v, want %v", refs[0].AttachmentID, attachmentID)
	}

	// アイキャッチ画像が設定されていることを確認（1行目が画像のため）
	if output.Page.FeaturedImageAttachmentID == nil {
		t.Error("FeaturedImageAttachmentID should not be nil")
	} else if *output.Page.FeaturedImageAttachmentID != attachmentID {
		t.Errorf("FeaturedImageAttachmentID = %v, want %v", *output.Page.FeaturedImageAttachmentID, attachmentID)
	}
}

func TestPublishPageUsecase_Execute_NoFeaturedImage(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	pageRepo := repository.NewPageRepository(q)
	pageRevisionRepo := repository.NewPageRevisionRepository(q)
	pageEditorRepo := repository.NewPageEditorRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	draftPageRevisionRepo := repository.NewDraftPageRevisionRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	attachmentRepo := repository.NewAttachmentRepository(q)
	pageAttachmentRefRepo := repository.NewPageAttachmentReferenceRepository(q)
	pageUpdateValidator := validator.NewPageUpdateValidator(pageRepo)
	uc := NewPublishPageUsecase(db, spaceRepo, spaceMemberRepo, pageRepo, pageRevisionRepo, pageEditorRepo, draftPageRepo, draftPageRevisionRepo, topicRepo, topicMemberRepo, attachmentRepo, pageAttachmentRefRepo, pageUpdateValidator)

	// テストデータを作成
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("publish-nofeatured").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("publish-nofeatured@example.com").
		WithAtname("publishnofeatured").
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
		WithTitle("No Featured Image").
		Build()

	// 1行目がテキストのみの本文（アイキャッチ画像なし）
	body := "This is plain text\nSome more content"
	testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("No Featured Image").
		WithBody(body).
		Build()

	output, err := uc.Execute(context.Background(), PublishPageInput{
		SpaceIdentifier: model.SpaceIdentifier("publish-nofeatured"),
		PageNumber:      1,
		UserID:          userID,
		Title:           "No Featured Image",
		Body:            body,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	// アイキャッチ画像が設定されていないことを確認
	if output.Page.FeaturedImageAttachmentID != nil {
		t.Errorf("FeaturedImageAttachmentID should be nil, got %v", *output.Page.FeaturedImageAttachmentID)
	}
}

func TestPublishPageUsecase_Execute_WithoutDraftPage(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	pageRepo := repository.NewPageRepository(q)
	pageRevisionRepo := repository.NewPageRevisionRepository(q)
	pageEditorRepo := repository.NewPageEditorRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	draftPageRevisionRepo := repository.NewDraftPageRevisionRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	attachmentRepo := repository.NewAttachmentRepository(q)
	pageAttachmentRefRepo := repository.NewPageAttachmentReferenceRepository(q)
	pageUpdateValidator := validator.NewPageUpdateValidator(pageRepo)
	uc := NewPublishPageUsecase(db, spaceRepo, spaceMemberRepo, pageRepo, pageRevisionRepo, pageEditorRepo, draftPageRepo, draftPageRevisionRepo, topicRepo, topicMemberRepo, attachmentRepo, pageAttachmentRefRepo, pageUpdateValidator)

	// テストデータを作成
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("publish-nodraft").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("publish-nodraft@example.com").
		WithAtname("publishnodraft").
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
	testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		Build()

	output, err := uc.Execute(context.Background(), PublishPageInput{
		SpaceIdentifier: model.SpaceIdentifier("publish-nodraft"),
		PageNumber:      1,
		UserID:          userID,
		Title:           "Updated Without Draft",
		Body:            "Published directly",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if output == nil {
		t.Fatal("output should not be nil")
	}
	if output.Page.Body != "Published directly" {
		t.Errorf("Body = %q, want %q", output.Page.Body, "Published directly")
	}
	if output.Page.Title == nil || *output.Page.Title != "Updated Without Draft" {
		t.Errorf("Title = %v, want %q", output.Page.Title, "Updated Without Draft")
	}
	if output.PublishedAt.IsZero() {
		t.Error("PublishedAt should not be zero")
	}
}

func TestPublishPageUsecase_Execute_DiscardUnpublishedConflictingPage(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	pageRepo := repository.NewPageRepository(q)
	pageRevisionRepo := repository.NewPageRevisionRepository(q)
	pageEditorRepo := repository.NewPageEditorRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	draftPageRevisionRepo := repository.NewDraftPageRevisionRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	attachmentRepo := repository.NewAttachmentRepository(q)
	pageAttachmentRefRepo := repository.NewPageAttachmentReferenceRepository(q)
	pageUpdateValidator := validator.NewPageUpdateValidator(pageRepo)
	uc := NewPublishPageUsecase(db, spaceRepo, spaceMemberRepo, pageRepo, pageRevisionRepo, pageEditorRepo, draftPageRepo, draftPageRevisionRepo, topicRepo, topicMemberRepo, attachmentRepo, pageAttachmentRefRepo, pageUpdateValidator)

	// テストデータを作成
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("publish-discard").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("publish-discard@example.com").
		WithAtname("publishdiscard").
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

	// リネーム対象のページ
	testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Original Title").
		Build()

	// 未公開かつ本文が空の競合ページ（Wikiリンクの自動保存で作成されたもの）
	conflictingPageID := testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("New Title").
		WithBody("").
		WithUnpublished().
		Build()

	output, err := uc.Execute(context.Background(), PublishPageInput{
		SpaceIdentifier: model.SpaceIdentifier("publish-discard"),
		PageNumber:      1,
		UserID:          userID,
		Title:           "New Title",
		Body:            "Updated body",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if output == nil {
		t.Fatal("output should not be nil")
	}
	if output.Page.Title == nil || *output.Page.Title != "New Title" {
		t.Errorf("Title = %v, want %q", output.Page.Title, "New Title")
	}

	// 競合ページが論理削除されていることを確認
	discardedPage, err := pageRepo.FindByTopicAndTitle(context.Background(), topicID, conflictingPageID.String(), spaceID)
	if err != nil {
		t.Fatalf("FindByTopicAndTitle() error = %v", err)
	}
	if discardedPage == nil {
		t.Fatal("discarded page should exist (with title changed to its ID)")
	}
	if discardedPage.DiscardedAt == nil {
		t.Error("discarded page should have DiscardedAt set")
	}
}

func TestPublishPageUsecase_Execute_WithDraftPageRevisions(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	pageRepo := repository.NewPageRepository(q)
	pageRevisionRepo := repository.NewPageRevisionRepository(q)
	pageEditorRepo := repository.NewPageEditorRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	draftPageRevisionRepo := repository.NewDraftPageRevisionRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	attachmentRepo := repository.NewAttachmentRepository(q)
	pageAttachmentRefRepo := repository.NewPageAttachmentReferenceRepository(q)
	pageUpdateValidator := validator.NewPageUpdateValidator(pageRepo)
	uc := NewPublishPageUsecase(db, spaceRepo, spaceMemberRepo, pageRepo, pageRevisionRepo, pageEditorRepo, draftPageRepo, draftPageRevisionRepo, topicRepo, topicMemberRepo, attachmentRepo, pageAttachmentRefRepo, pageUpdateValidator)

	// テストデータを作成
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("publish-draftrev").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("publish-draftrev@example.com").
		WithAtname("publishdraftrev").
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
		WithTitle("Test Page").
		Build()
	draftPageID := testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Draft With Revisions").
		WithBody("Draft body with revisions").
		Build()

	// 下書きリビジョンを作成（外部キー制約の再現）
	testutil.NewDraftPageRevisionBuilderDB(t, db).
		WithDraftPageID(draftPageID).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	testutil.NewDraftPageRevisionBuilderDB(t, db).
		WithDraftPageID(draftPageID).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		Build()

	output, err := uc.Execute(context.Background(), PublishPageInput{
		SpaceIdentifier: model.SpaceIdentifier("publish-draftrev"),
		PageNumber:      1,
		UserID:          userID,
		Title:           "Draft With Revisions",
		Body:            "Draft body with revisions",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if output == nil {
		t.Fatal("output should not be nil")
	}
	if output.Page.Body != "Draft body with revisions" {
		t.Errorf("Body = %q, want %q", output.Page.Body, "Draft body with revisions")
	}
	if output.PublishedAt.IsZero() {
		t.Error("PublishedAt should not be zero")
	}
}
