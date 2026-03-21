package usecase

import (
	"context"
	"fmt"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetPagePublishDataUsecase_Execute(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	attachmentRepo := repository.NewAttachmentRepository(q)
	pageAttachmentRefRepo := repository.NewPageAttachmentReferenceRepository(q)
	uc := NewGetPagePublishDataUsecase(attachmentRepo, pageAttachmentRefRepo)

	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("get-publish-data").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("get-publish-data@example.com").
		WithAtname("getpublishdata").
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

	output, err := uc.Execute(context.Background(), GetPagePublishDataInput{
		Body:    "Hello **world**",
		PageID:  pageID,
		SpaceID: spaceID,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if output.BodyHTML == "" {
		t.Error("BodyHTML should not be empty")
	}
	if output.FeaturedImageAttachmentID != nil {
		t.Errorf("FeaturedImageAttachmentID should be nil, got %v", *output.FeaturedImageAttachmentID)
	}
	if len(output.AttachmentRefsToAdd) != 0 {
		t.Errorf("AttachmentRefsToAdd should be empty, got %v", output.AttachmentRefsToAdd)
	}
	if len(output.AttachmentRefsToRemove) != 0 {
		t.Errorf("AttachmentRefsToRemove should be empty, got %v", output.AttachmentRefsToRemove)
	}

	_ = spaceMemberID
}

func TestGetPagePublishDataUsecase_Execute_WithAttachment(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	attachmentRepo := repository.NewAttachmentRepository(q)
	pageAttachmentRefRepo := repository.NewPageAttachmentReferenceRepository(q)
	uc := NewGetPagePublishDataUsecase(attachmentRepo, pageAttachmentRefRepo)

	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("get-publish-attach").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("get-publish-attach@example.com").
		WithAtname("getpublishattach").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithName("General").
		Build()

	attachmentID := testutil.NewAttachmentBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		WithFilename("test.png").
		Build()

	pageID := testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		Build()

	body := fmt.Sprintf("![image](/attachments/%s)", attachmentID)
	output, err := uc.Execute(context.Background(), GetPagePublishDataInput{
		Body:    body,
		PageID:  pageID,
		SpaceID: spaceID,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	// アイキャッチ画像が設定されていることを確認
	if output.FeaturedImageAttachmentID == nil {
		t.Error("FeaturedImageAttachmentID should not be nil")
	} else if *output.FeaturedImageAttachmentID != attachmentID {
		t.Errorf("FeaturedImageAttachmentID = %v, want %v", *output.FeaturedImageAttachmentID, attachmentID)
	}

	// 添付ファイル参照の追加分に含まれていることを確認
	if len(output.AttachmentRefsToAdd) != 1 {
		t.Fatalf("AttachmentRefsToAdd length = %d, want 1", len(output.AttachmentRefsToAdd))
	}
	if output.AttachmentRefsToAdd[0] != attachmentID {
		t.Errorf("AttachmentRefsToAdd[0] = %v, want %v", output.AttachmentRefsToAdd[0], attachmentID)
	}
}

func TestGetPagePublishDataUsecase_Execute_NoFeaturedImage(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	attachmentRepo := repository.NewAttachmentRepository(q)
	pageAttachmentRefRepo := repository.NewPageAttachmentReferenceRepository(q)
	uc := NewGetPagePublishDataUsecase(attachmentRepo, pageAttachmentRefRepo)

	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("get-publish-nofeat").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("get-publish-nofeat@example.com").
		WithAtname("getpublishnofeat").
		Build()
	testutil.NewSpaceMemberBuilderDB(t, db).
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

	output, err := uc.Execute(context.Background(), GetPagePublishDataInput{
		Body:    "This is plain text\nSome more content",
		PageID:  pageID,
		SpaceID: spaceID,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	if output.FeaturedImageAttachmentID != nil {
		t.Errorf("FeaturedImageAttachmentID should be nil, got %v", *output.FeaturedImageAttachmentID)
	}
}
