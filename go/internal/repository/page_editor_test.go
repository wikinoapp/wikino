package repository

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestPageEditorRepository_FindOrCreate(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewPageEditorRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("pageeditor-test@example.com").
		WithAtname("pageeditortest").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("pageeditor-test-space").
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

	now := time.Now()

	t.Run("存在しない場合は新規作成する", func(t *testing.T) {
		editor, err := repo.FindOrCreate(context.Background(), FindOrCreateInput{
			SpaceID:            spaceID,
			PageID:             pageID,
			SpaceMemberID:      spaceMemberID,
			LastPageModifiedAt: now,
		})
		if err != nil {
			t.Fatalf("FindOrCreate() error = %v", err)
		}
		if editor == nil {
			t.Fatal("FindOrCreate() returned nil, want page editor")
		}
		if editor.ID == "" {
			t.Error("editor.ID should not be empty")
		}
		if editor.SpaceID != spaceID {
			t.Errorf("editor.SpaceID = %v, want %v", editor.SpaceID, spaceID)
		}
		if editor.PageID != pageID {
			t.Errorf("editor.PageID = %v, want %v", editor.PageID, pageID)
		}
		if editor.SpaceMemberID != spaceMemberID {
			t.Errorf("editor.SpaceMemberID = %v, want %v", editor.SpaceMemberID, spaceMemberID)
		}
		if editor.CreatedAt.IsZero() {
			t.Error("editor.CreatedAt should not be zero")
		}
		if editor.UpdatedAt.IsZero() {
			t.Error("editor.UpdatedAt should not be zero")
		}
	})

	t.Run("既に存在する場合は既存のレコードを返す", func(t *testing.T) {
		editor1, err := repo.FindOrCreate(context.Background(), FindOrCreateInput{
			SpaceID:            spaceID,
			PageID:             pageID,
			SpaceMemberID:      spaceMemberID,
			LastPageModifiedAt: now,
		})
		if err != nil {
			t.Fatalf("FindOrCreate() first call error = %v", err)
		}

		editor2, err := repo.FindOrCreate(context.Background(), FindOrCreateInput{
			SpaceID:            spaceID,
			PageID:             pageID,
			SpaceMemberID:      spaceMemberID,
			LastPageModifiedAt: now.Add(time.Hour),
		})
		if err != nil {
			t.Fatalf("FindOrCreate() second call error = %v", err)
		}

		if editor1.ID != editor2.ID {
			t.Errorf("FindOrCreate() should return same record, got IDs %v and %v", editor1.ID, editor2.ID)
		}
	})

	t.Run("異なるスペースメンバーの場合は別のレコードを作成する", func(t *testing.T) {
		userID2 := testutil.NewUserBuilder(t, tx).
			WithEmail("pageeditor-test2@example.com").
			WithAtname("pageeditortest2").
			Build()

		spaceMemberID2 := testutil.NewSpaceMemberBuilder(t, tx).
			WithSpaceID(spaceID).
			WithUserID(userID2).
			Build()

		editor1, err := repo.FindOrCreate(context.Background(), FindOrCreateInput{
			SpaceID:            spaceID,
			PageID:             pageID,
			SpaceMemberID:      spaceMemberID,
			LastPageModifiedAt: now,
		})
		if err != nil {
			t.Fatalf("FindOrCreate() for member1 error = %v", err)
		}

		editor2, err := repo.FindOrCreate(context.Background(), FindOrCreateInput{
			SpaceID:            spaceID,
			PageID:             pageID,
			SpaceMemberID:      spaceMemberID2,
			LastPageModifiedAt: now,
		})
		if err != nil {
			t.Fatalf("FindOrCreate() for member2 error = %v", err)
		}

		if editor1.ID == editor2.ID {
			t.Errorf("FindOrCreate() should return different records for different members, got same ID %v", editor1.ID)
		}
	})
}

func TestPageEditorRepository_UpdateLastPageModifiedAt(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewPageEditorRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("pageeditor-update@example.com").
		WithAtname("pageeditorupdate").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("pageeditor-update-space").
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

	now := time.Now()

	// まずPageEditorを作成
	editor, err := repo.FindOrCreate(context.Background(), FindOrCreateInput{
		SpaceID:            spaceID,
		PageID:             pageID,
		SpaceMemberID:      spaceMemberID,
		LastPageModifiedAt: now,
	})
	if err != nil {
		t.Fatalf("FindOrCreate() error = %v", err)
	}

	t.Run("LastPageModifiedAtを更新できる", func(t *testing.T) {
		newTime := now.Add(time.Hour).Truncate(time.Microsecond)
		updated, err := repo.UpdateLastPageModifiedAt(context.Background(), UpdateLastPageModifiedAtInput{
			ID:                 editor.ID,
			SpaceID:            spaceID,
			LastPageModifiedAt: newTime,
		})
		if err != nil {
			t.Fatalf("UpdateLastPageModifiedAt() error = %v", err)
		}
		if updated == nil {
			t.Fatal("UpdateLastPageModifiedAt() returned nil, want page editor")
		}
		if updated.ID != editor.ID {
			t.Errorf("updated.ID = %v, want %v", updated.ID, editor.ID)
		}
		if !updated.LastPageModifiedAt.Equal(newTime) {
			t.Errorf("updated.LastPageModifiedAt = %v, want %v", updated.LastPageModifiedAt, newTime)
		}
		if !updated.UpdatedAt.After(editor.UpdatedAt) {
			t.Error("updated.UpdatedAt should be after original UpdatedAt")
		}
	})
}
