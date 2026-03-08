package repository

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestDraftPageRevisionRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewDraftPageRevisionRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("draftpagerev-create@example.com").
		WithAtname("draftpagerevcreate").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("draftpagerev-create").
		Build()

	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()

	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		Build()

	draftPageID := testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Draft Title").
		WithBody("Draft body").
		WithBodyHTML("<p>Draft body</p>").
		Build()

	t.Run("下書きページリビジョンを作成できる", func(t *testing.T) {
		revision, err := repo.Create(context.Background(), CreateDraftPageRevisionInput{
			DraftPageID:   draftPageID,
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
			Title:         "Revision Title",
			Body:          "revision body",
			BodyHTML:      "<p>revision body</p>",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if revision == nil {
			t.Fatal("Create() returned nil, want draft page revision")
		}
		if revision.ID == "" {
			t.Error("revision.ID should not be empty")
		}
		if revision.DraftPageID != draftPageID {
			t.Errorf("revision.DraftPageID = %v, want %v", revision.DraftPageID, draftPageID)
		}
		if revision.SpaceID != spaceID {
			t.Errorf("revision.SpaceID = %v, want %v", revision.SpaceID, spaceID)
		}
		if revision.SpaceMemberID != spaceMemberID {
			t.Errorf("revision.SpaceMemberID = %v, want %v", revision.SpaceMemberID, spaceMemberID)
		}
		if revision.Title != "Revision Title" {
			t.Errorf("revision.Title = %v, want 'Revision Title'", revision.Title)
		}
		if revision.Body != "revision body" {
			t.Errorf("revision.Body = %v, want 'revision body'", revision.Body)
		}
		if revision.BodyHTML != "<p>revision body</p>" {
			t.Errorf("revision.BodyHTML = %v, want '<p>revision body</p>'", revision.BodyHTML)
		}
		if revision.CreatedAt.IsZero() {
			t.Error("revision.CreatedAt should not be zero")
		}
	})

	t.Run("下書きページIDに紐づくリビジョンをすべて削除できる", func(t *testing.T) {
		// リビジョンを2つ作成
		_, err := repo.Create(context.Background(), CreateDraftPageRevisionInput{
			DraftPageID:   draftPageID,
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
			Title:         "Delete Test 1",
			Body:          "delete body 1",
			BodyHTML:      "<p>delete body 1</p>",
		})
		if err != nil {
			t.Fatalf("Create() first revision error = %v", err)
		}

		_, err = repo.Create(context.Background(), CreateDraftPageRevisionInput{
			DraftPageID:   draftPageID,
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
			Title:         "Delete Test 2",
			Body:          "delete body 2",
			BodyHTML:      "<p>delete body 2</p>",
		})
		if err != nil {
			t.Fatalf("Create() second revision error = %v", err)
		}

		// 削除
		err = repo.DeleteByDraftPageID(context.Background(), draftPageID, spaceID)
		if err != nil {
			t.Fatalf("DeleteByDraftPageID() error = %v", err)
		}
	})

	t.Run("同じ下書きに対して複数のリビジョンを作成できる", func(t *testing.T) {
		revision1, err := repo.Create(context.Background(), CreateDraftPageRevisionInput{
			DraftPageID:   draftPageID,
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
			Title:         "First Revision",
			Body:          "first body",
			BodyHTML:      "<p>first body</p>",
		})
		if err != nil {
			t.Fatalf("Create() first revision error = %v", err)
		}

		revision2, err := repo.Create(context.Background(), CreateDraftPageRevisionInput{
			DraftPageID:   draftPageID,
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
			Title:         "Second Revision",
			Body:          "second body",
			BodyHTML:      "<p>second body</p>",
		})
		if err != nil {
			t.Fatalf("Create() second revision error = %v", err)
		}

		if revision1.ID == revision2.ID {
			t.Errorf("revision1.ID and revision2.ID should be different, got %v", revision1.ID)
		}
		if revision1.Title != "First Revision" {
			t.Errorf("revision1.Title = %v, want 'First Revision'", revision1.Title)
		}
		if revision2.Title != "Second Revision" {
			t.Errorf("revision2.Title = %v, want 'Second Revision'", revision2.Title)
		}
	})
}
