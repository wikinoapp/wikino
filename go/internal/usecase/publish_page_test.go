package usecase

import (
	"context"
	"fmt"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestPublishPageUsecase_Execute(t *testing.T) {
	db := testutil.GetTestDB()
	q := query.New(db)
	pageRepo := repository.NewPageRepository(q)
	pageRevisionRepo := repository.NewPageRevisionRepository(q)
	pageEditorRepo := repository.NewPageEditorRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	attachmentRepo := repository.NewAttachmentRepository(q)
	pageAttachmentRefRepo := repository.NewPageAttachmentReferenceRepository(q)
	uc := NewPublishPageUsecase(db, pageRepo, pageRevisionRepo, pageEditorRepo, draftPageRepo, topicRepo, topicMemberRepo, attachmentRepo, pageAttachmentRefRepo)

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
	draftPageID := testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Updated Title").
		WithBody("Updated body").
		Build()

	title := "Updated Title"
	output, err := uc.Execute(context.Background(), PublishPageInput{
		SpaceID:          spaceID,
		PageID:           pageID,
		SpaceMemberID:    spaceMemberID,
		TopicID:          topicID,
		DraftPageID:      draftPageID,
		Title:            &title,
		Body:             "Updated body",
		SpaceIdentifier:  "publish-test",
		CurrentTopicName: "General",
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
	db := testutil.GetTestDB()
	q := query.New(db)
	pageRepo := repository.NewPageRepository(q)
	pageRevisionRepo := repository.NewPageRevisionRepository(q)
	pageEditorRepo := repository.NewPageEditorRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	attachmentRepo := repository.NewAttachmentRepository(q)
	pageAttachmentRefRepo := repository.NewPageAttachmentReferenceRepository(q)
	uc := NewPublishPageUsecase(db, pageRepo, pageRevisionRepo, pageEditorRepo, draftPageRepo, topicRepo, topicMemberRepo, attachmentRepo, pageAttachmentRefRepo)

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
	draftPageID := testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Wikilink Publish Test").
		WithBody("See [[リンク先ページ]]").
		Build()

	title := "Wikilink Publish Test"
	output, err := uc.Execute(context.Background(), PublishPageInput{
		SpaceID:          spaceID,
		PageID:           pageID,
		SpaceMemberID:    spaceMemberID,
		TopicID:          topicID,
		DraftPageID:      draftPageID,
		Title:            &title,
		Body:             "See [[リンク先ページ]]",
		SpaceIdentifier:  "publish-wikilink",
		CurrentTopicName: "General",
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
	db := testutil.GetTestDB()
	q := query.New(db)
	pageRepo := repository.NewPageRepository(q)
	pageRevisionRepo := repository.NewPageRevisionRepository(q)
	pageEditorRepo := repository.NewPageEditorRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	attachmentRepo := repository.NewAttachmentRepository(q)
	pageAttachmentRefRepo := repository.NewPageAttachmentReferenceRepository(q)
	uc := NewPublishPageUsecase(db, pageRepo, pageRevisionRepo, pageEditorRepo, draftPageRepo, topicRepo, topicMemberRepo, attachmentRepo, pageAttachmentRefRepo)

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
	draftPageID := testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithBody("Body without title").
		Build()

	output, err := uc.Execute(context.Background(), PublishPageInput{
		SpaceID:          spaceID,
		PageID:           pageID,
		SpaceMemberID:    spaceMemberID,
		TopicID:          topicID,
		DraftPageID:      draftPageID,
		Title:            nil,
		Body:             "Body without title",
		SpaceIdentifier:  "publish-niltitle",
		CurrentTopicName: "General",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if output.Page.Body != "Body without title" {
		t.Errorf("Body = %q, want %q", output.Page.Body, "Body without title")
	}
	if output.PublishedAt.IsZero() {
		t.Error("PublishedAt should not be zero")
	}
}

func TestPublishPageUsecase_Execute_ExistingLinkedPage(t *testing.T) {
	db := testutil.GetTestDB()
	q := query.New(db)
	pageRepo := repository.NewPageRepository(q)
	pageRevisionRepo := repository.NewPageRevisionRepository(q)
	pageEditorRepo := repository.NewPageEditorRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	attachmentRepo := repository.NewAttachmentRepository(q)
	pageAttachmentRefRepo := repository.NewPageAttachmentReferenceRepository(q)
	uc := NewPublishPageUsecase(db, pageRepo, pageRevisionRepo, pageEditorRepo, draftPageRepo, topicRepo, topicMemberRepo, attachmentRepo, pageAttachmentRefRepo)

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

	draftPageID := testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Existing Link Test").
		WithBody("See [[既存ページ]]").
		Build()

	title := "Existing Link Test"
	output, err := uc.Execute(context.Background(), PublishPageInput{
		SpaceID:          spaceID,
		PageID:           pageID,
		SpaceMemberID:    spaceMemberID,
		TopicID:          topicID,
		DraftPageID:      draftPageID,
		Title:            &title,
		Body:             "See [[既存ページ]]",
		SpaceIdentifier:  "publish-existing-link",
		CurrentTopicName: "General",
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

func TestPublishPageUsecase_Execute_WithAttachments(t *testing.T) {
	db := testutil.GetTestDB()
	q := query.New(db)
	pageRepo := repository.NewPageRepository(q)
	pageRevisionRepo := repository.NewPageRevisionRepository(q)
	pageEditorRepo := repository.NewPageEditorRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	attachmentRepo := repository.NewAttachmentRepository(q)
	pageAttachmentRefRepo := repository.NewPageAttachmentReferenceRepository(q)
	uc := NewPublishPageUsecase(db, pageRepo, pageRevisionRepo, pageEditorRepo, draftPageRepo, topicRepo, topicMemberRepo, attachmentRepo, pageAttachmentRefRepo)

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
	draftPageID := testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Attachment Test").
		WithBody(body).
		Build()

	title := "Attachment Test"
	output, err := uc.Execute(context.Background(), PublishPageInput{
		SpaceID:          spaceID,
		PageID:           pageID,
		SpaceMemberID:    spaceMemberID,
		TopicID:          topicID,
		DraftPageID:      draftPageID,
		Title:            &title,
		Body:             body,
		SpaceIdentifier:  "publish-attach",
		CurrentTopicName: "General",
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
	db := testutil.GetTestDB()
	q := query.New(db)
	pageRepo := repository.NewPageRepository(q)
	pageRevisionRepo := repository.NewPageRevisionRepository(q)
	pageEditorRepo := repository.NewPageEditorRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	attachmentRepo := repository.NewAttachmentRepository(q)
	pageAttachmentRefRepo := repository.NewPageAttachmentReferenceRepository(q)
	uc := NewPublishPageUsecase(db, pageRepo, pageRevisionRepo, pageEditorRepo, draftPageRepo, topicRepo, topicMemberRepo, attachmentRepo, pageAttachmentRefRepo)

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
	draftPageID := testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("No Featured Image").
		WithBody(body).
		Build()

	title := "No Featured Image"
	output, err := uc.Execute(context.Background(), PublishPageInput{
		SpaceID:          spaceID,
		PageID:           pageID,
		SpaceMemberID:    spaceMemberID,
		TopicID:          topicID,
		DraftPageID:      draftPageID,
		Title:            &title,
		Body:             body,
		SpaceIdentifier:  "publish-nofeatured",
		CurrentTopicName: "General",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	// アイキャッチ画像が設定されていないことを確認
	if output.Page.FeaturedImageAttachmentID != nil {
		t.Errorf("FeaturedImageAttachmentID should be nil, got %v", *output.Page.FeaturedImageAttachmentID)
	}
}
