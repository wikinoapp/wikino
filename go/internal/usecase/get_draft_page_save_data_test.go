package usecase

import (
	"context"
	"fmt"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetDraftPageSaveDataUsecase_Execute(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	attachmentRepo := repository.NewAttachmentRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	uc := NewGetDraftPageSaveDataUsecase(attachmentRepo, topicRepo)

	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("get-draft-save-data").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("get-draft-save-data@example.com").
		WithAtname("getdraftsavedata").
		Build()
	testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithName("General").
		Build()

	output, err := uc.Execute(context.Background(), GetDraftPageSaveDataInput{
		Body:             "Hello **world**",
		SpaceID:          spaceID,
		CurrentTopicName: "General",
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
	if output.WikilinkKeys != nil {
		t.Errorf("WikilinkKeys should be nil, got %v", output.WikilinkKeys)
	}
	if output.TopicMap != nil {
		t.Errorf("TopicMap should be nil, got %v", output.TopicMap)
	}
}

func TestGetDraftPageSaveDataUsecase_Execute_WithWikilinks(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	attachmentRepo := repository.NewAttachmentRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	uc := NewGetDraftPageSaveDataUsecase(attachmentRepo, topicRepo)

	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("get-draft-save-wiki").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("get-draft-save-wiki@example.com").
		WithAtname("getdraftsavewiki").
		Build()
	testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithName("General").
		Build()

	output, err := uc.Execute(context.Background(), GetDraftPageSaveDataInput{
		Body:             "See [[リンク先ページ]]",
		SpaceID:          spaceID,
		CurrentTopicName: "General",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if len(output.WikilinkKeys) != 1 {
		t.Fatalf("WikilinkKeys length = %d, want 1", len(output.WikilinkKeys))
	}
	if output.WikilinkKeys[0].TopicName != "General" {
		t.Errorf("WikilinkKeys[0].TopicName = %q, want %q", output.WikilinkKeys[0].TopicName, "General")
	}
	if output.WikilinkKeys[0].PageTitle != "リンク先ページ" {
		t.Errorf("WikilinkKeys[0].PageTitle = %q, want %q", output.WikilinkKeys[0].PageTitle, "リンク先ページ")
	}
	if output.TopicMap == nil {
		t.Fatal("TopicMap should not be nil")
	}
	if _, ok := output.TopicMap["General"]; !ok {
		t.Error("TopicMap should contain 'General'")
	}
}

func TestGetDraftPageSaveDataUsecase_Execute_WithAttachment(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	attachmentRepo := repository.NewAttachmentRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	uc := NewGetDraftPageSaveDataUsecase(attachmentRepo, topicRepo)

	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("get-draft-save-attach").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("get-draft-save-attach@example.com").
		WithAtname("getdraftsaveattach").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithName("General").
		Build()

	attachmentID := testutil.NewAttachmentBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		WithFilename("test.png").
		Build()

	body := fmt.Sprintf("![image](/attachments/%s)", attachmentID)
	output, err := uc.Execute(context.Background(), GetDraftPageSaveDataInput{
		Body:             body,
		SpaceID:          spaceID,
		CurrentTopicName: "General",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	if output.FeaturedImageAttachmentID == nil {
		t.Error("FeaturedImageAttachmentID should not be nil")
	} else if *output.FeaturedImageAttachmentID != attachmentID {
		t.Errorf("FeaturedImageAttachmentID = %v, want %v", *output.FeaturedImageAttachmentID, attachmentID)
	}
}
