package usecase

import (
	"context"
	"database/sql"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func newManualSaveUC(db *sql.DB) *ManualSaveDraftPageUsecase {
	q := query.New(db)
	return NewManualSaveDraftPageUsecase(
		db,
		repository.NewDraftPageRepository(q),
		repository.NewDraftPageRevisionRepository(q),
		repository.NewPageRepository(q),
		repository.NewPageEditorRepository(q),
		repository.NewTopicRepository(q),
		repository.NewAttachmentRepository(q),
	)
}

func TestManualSaveDraftPageUsecase_Execute(t *testing.T) {
	db := testutil.GetTestDB()
	uc := newManualSaveUC(db)

	// テストデータを作成
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("manual-save").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("manual-save@example.com").
		WithAtname("manualsave").
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

	title := "下書きタイトル"
	output, err := uc.Execute(context.Background(), ManualSaveDraftPageInput{
		SpaceID:          spaceID,
		PageID:           pageID,
		SpaceMemberID:    spaceMemberID,
		TopicID:          topicID,
		Title:            &title,
		Body:             "下書き本文",
		SpaceIdentifier:  "manual-save",
		CurrentTopicName: "General",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if output == nil {
		t.Fatal("output should not be nil")
	}
	if output.DraftPageRevision == nil {
		t.Fatal("DraftPageRevision should not be nil")
	}
	if output.DraftPageRevision.Title != "下書きタイトル" {
		t.Errorf("Title = %q, want %q", output.DraftPageRevision.Title, "下書きタイトル")
	}
	if output.DraftPageRevision.Body != "下書き本文" {
		t.Errorf("Body = %q, want %q", output.DraftPageRevision.Body, "下書き本文")
	}
	if output.DraftPageRevision.SpaceMemberID != spaceMemberID {
		t.Errorf("SpaceMemberID = %v, want %v", output.DraftPageRevision.SpaceMemberID, spaceMemberID)
	}
	if output.DraftPageRevision.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestManualSaveDraftPageUsecase_Execute_WithoutDraftPage(t *testing.T) {
	db := testutil.GetTestDB()
	uc := newManualSaveUC(db)

	// テストデータを作成（DraftPageは作成しない）
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("manual-save-nodraft").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("manual-save-nodraft@example.com").
		WithAtname("manualsavenodraft").
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

	title := "新規下書き"
	output, err := uc.Execute(context.Background(), ManualSaveDraftPageInput{
		SpaceID:          spaceID,
		PageID:           pageID,
		SpaceMemberID:    spaceMemberID,
		TopicID:          topicID,
		Title:            &title,
		Body:             "新規下書き本文",
		SpaceIdentifier:  model.SpaceIdentifier("manual-save-nodraft"),
		CurrentTopicName: "General",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if output == nil {
		t.Fatal("output should not be nil")
	}
	if output.DraftPageRevision == nil {
		t.Fatal("DraftPageRevision should not be nil")
	}
	if output.DraftPageRevision.Title != "新規下書き" {
		t.Errorf("Title = %q, want %q", output.DraftPageRevision.Title, "新規下書き")
	}
	if output.DraftPageRevision.Body != "新規下書き本文" {
		t.Errorf("Body = %q, want %q", output.DraftPageRevision.Body, "新規下書き本文")
	}
}
