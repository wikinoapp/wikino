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
			t.Fatalf("FindOrCreate()のエラー = %v", err)
		}
		if editor == nil {
			t.Fatal("FindOrCreate()がnilを返した、期待値 = ページ編集者")
		}
		if editor.ID == "" {
			t.Error("editor.IDが空")
		}
		if editor.SpaceID != spaceID {
			t.Errorf("editor.SpaceID = %v、期待値 = %v", editor.SpaceID, spaceID)
		}
		if editor.PageID != pageID {
			t.Errorf("editor.PageID = %v、期待値 = %v", editor.PageID, pageID)
		}
		if editor.SpaceMemberID != spaceMemberID {
			t.Errorf("editor.SpaceMemberID = %v、期待値 = %v", editor.SpaceMemberID, spaceMemberID)
		}
		if editor.CreatedAt.IsZero() {
			t.Error("editor.CreatedAtがゼロ値")
		}
		if editor.UpdatedAt.IsZero() {
			t.Error("editor.UpdatedAtがゼロ値")
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
			t.Fatalf("FindOrCreate() (1回目の呼び出し) のエラー = %v", err)
		}

		editor2, err := repo.FindOrCreate(context.Background(), FindOrCreateInput{
			SpaceID:            spaceID,
			PageID:             pageID,
			SpaceMemberID:      spaceMemberID,
			LastPageModifiedAt: now.Add(time.Hour),
		})
		if err != nil {
			t.Fatalf("FindOrCreate() (2回目の呼び出し) のエラー = %v", err)
		}

		if editor1.ID != editor2.ID {
			t.Errorf("FindOrCreate()が異なるレコードを返した: ID = %v、%v", editor1.ID, editor2.ID)
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
			t.Fatalf("FindOrCreate() (member1) のエラー = %v", err)
		}

		editor2, err := repo.FindOrCreate(context.Background(), FindOrCreateInput{
			SpaceID:            spaceID,
			PageID:             pageID,
			SpaceMemberID:      spaceMemberID2,
			LastPageModifiedAt: now,
		})
		if err != nil {
			t.Fatalf("FindOrCreate() (member2) のエラー = %v", err)
		}

		if editor1.ID == editor2.ID {
			t.Errorf("FindOrCreate()がメンバーが異なるのに同じレコードを返した: ID = %v", editor1.ID)
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
		t.Fatalf("FindOrCreate()のエラー = %v", err)
	}

	t.Run("LastPageModifiedAtを更新できる", func(t *testing.T) {
		newTime := now.Add(time.Hour).Truncate(time.Microsecond)
		updated, err := repo.UpdateLastPageModifiedAt(context.Background(), UpdateLastPageModifiedAtInput{
			ID:                 editor.ID,
			SpaceID:            spaceID,
			LastPageModifiedAt: newTime,
		})
		if err != nil {
			t.Fatalf("UpdateLastPageModifiedAt()のエラー = %v", err)
		}
		if updated == nil {
			t.Fatal("UpdateLastPageModifiedAt()がnilを返した、期待値 = ページ編集者")
		}
		if updated.ID != editor.ID {
			t.Errorf("updated.ID = %v、期待値 = %v", updated.ID, editor.ID)
		}
		if !updated.LastPageModifiedAt.Equal(newTime) {
			t.Errorf("updated.LastPageModifiedAt = %v、期待値 = %v", updated.LastPageModifiedAt, newTime)
		}
		if !updated.UpdatedAt.After(editor.UpdatedAt) {
			t.Error("updated.UpdatedAtが元のUpdatedAtより後になっていない")
		}
	})
}
