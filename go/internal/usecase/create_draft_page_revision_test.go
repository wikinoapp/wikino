package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestCreateDraftPageRevisionUsecase_Execute(t *testing.T) {
	db := testutil.GetTestDB()
	q := query.New(db)
	draftPageRepo := repository.NewDraftPageRepository(q)
	draftPageRevisionRepo := repository.NewDraftPageRevisionRepository(q)
	uc := NewCreateDraftPageRevisionUsecase(db, draftPageRepo, draftPageRevisionRepo)

	// テストデータを作成
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("create-dpr").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("create-dpr@example.com").
		WithAtname("createdpr").
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
	_ = testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("下書きタイトル").
		WithBody("下書き本文").
		WithBodyHTML("<p>下書き本文</p>").
		Build()

	output, err := uc.Execute(context.Background(), CreateDraftPageRevisionInput{
		SpaceID:       spaceID,
		PageID:        pageID,
		SpaceMemberID: spaceMemberID,
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
	if output.DraftPageRevision.BodyHTML != "<p>下書き本文</p>" {
		t.Errorf("BodyHTML = %q, want %q", output.DraftPageRevision.BodyHTML, "<p>下書き本文</p>")
	}
	if output.DraftPageRevision.SpaceMemberID != spaceMemberID {
		t.Errorf("SpaceMemberID = %v, want %v", output.DraftPageRevision.SpaceMemberID, spaceMemberID)
	}
	if output.DraftPageRevision.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestCreateDraftPageRevisionUsecase_Execute_DraftPageNotFound(t *testing.T) {
	db := testutil.GetTestDB()
	q := query.New(db)
	draftPageRepo := repository.NewDraftPageRepository(q)
	draftPageRevisionRepo := repository.NewDraftPageRevisionRepository(q)
	uc := NewCreateDraftPageRevisionUsecase(db, draftPageRepo, draftPageRevisionRepo)

	// テストデータを作成（DraftPageは作成しない）
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("create-dpr-notfound").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("create-dpr-notfound@example.com").
		WithAtname("createdprnotfound").
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

	_, err := uc.Execute(context.Background(), CreateDraftPageRevisionInput{
		SpaceID:       spaceID,
		PageID:        pageID,
		SpaceMemberID: spaceMemberID,
	})
	if err == nil {
		t.Fatal("Execute() error = nil, want ErrDraftPageNotFound")
	}
	if err != ErrDraftPageNotFound {
		t.Errorf("Execute() error = %v, want ErrDraftPageNotFound", err)
	}
}
