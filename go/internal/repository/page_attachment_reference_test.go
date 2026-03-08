package repository

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestPageAttachmentReferenceRepository_ListByPageID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewPageAttachmentReferenceRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("par-list@example.com").
		WithAtname("parlist").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("par-list-space").
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

	attachmentID1 := testutil.NewAttachmentBuilder(t, tx).WithSpaceID(spaceID).WithSpaceMemberID(spaceMemberID).Build()
	attachmentID2 := testutil.NewAttachmentBuilder(t, tx).WithSpaceID(spaceID).WithSpaceMemberID(spaceMemberID).Build()

	// テストデータを作成
	_, err := repo.CreateBatch(context.Background(), pageID, spaceID, []model.AttachmentID{attachmentID1, attachmentID2})
	if err != nil {
		t.Fatalf("CreateBatch() error = %v", err)
	}

	t.Run("ページIDに紐づく添付ファイル参照を取得できる", func(t *testing.T) {
		refs, err := repo.ListByPageID(context.Background(), pageID, spaceID)
		if err != nil {
			t.Fatalf("ListByPageID() error = %v", err)
		}
		if len(refs) != 2 {
			t.Fatalf("ListByPageID() returned %d refs, want 2", len(refs))
		}
	})

	t.Run("参照がないページIDは空スライスを返す", func(t *testing.T) {
		otherPageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(2).
			WithTitle("Other Page").
			Build()

		refs, err := repo.ListByPageID(context.Background(), otherPageID, spaceID)
		if err != nil {
			t.Fatalf("ListByPageID() error = %v", err)
		}
		if len(refs) != 0 {
			t.Errorf("ListByPageID() returned %d refs, want 0", len(refs))
		}
	})
}

func TestPageAttachmentReferenceRepository_CreateBatch(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewPageAttachmentReferenceRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("par-create@example.com").
		WithAtname("parcreate").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("par-create-space").
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

	t.Run("複数の添付ファイル参照を一括作成できる", func(t *testing.T) {
		attachmentID1 := testutil.NewAttachmentBuilder(t, tx).WithSpaceID(spaceID).WithSpaceMemberID(spaceMemberID).Build()
		attachmentID2 := testutil.NewAttachmentBuilder(t, tx).WithSpaceID(spaceID).WithSpaceMemberID(spaceMemberID).Build()

		refs, err := repo.CreateBatch(context.Background(), pageID, spaceID, []model.AttachmentID{attachmentID1, attachmentID2})
		if err != nil {
			t.Fatalf("CreateBatch() error = %v", err)
		}
		if len(refs) != 2 {
			t.Fatalf("CreateBatch() returned %d refs, want 2", len(refs))
		}
		for _, ref := range refs {
			if ref.ID == "" {
				t.Error("ref.ID should not be empty")
			}
			if ref.PageID != pageID {
				t.Errorf("ref.PageID = %v, want %v", ref.PageID, pageID)
			}
			if ref.CreatedAt.IsZero() {
				t.Error("ref.CreatedAt should not be zero")
			}
			if ref.UpdatedAt.IsZero() {
				t.Error("ref.UpdatedAt should not be zero")
			}
		}
		if refs[0].AttachmentID != attachmentID1 {
			t.Errorf("refs[0].AttachmentID = %v, want %v", refs[0].AttachmentID, attachmentID1)
		}
		if refs[1].AttachmentID != attachmentID2 {
			t.Errorf("refs[1].AttachmentID = %v, want %v", refs[1].AttachmentID, attachmentID2)
		}
	})

	t.Run("空のattachmentIDsリストの場合は空スライスを返す", func(t *testing.T) {
		refs, err := repo.CreateBatch(context.Background(), pageID, spaceID, []model.AttachmentID{})
		if err != nil {
			t.Fatalf("CreateBatch() error = %v", err)
		}
		if len(refs) != 0 {
			t.Errorf("CreateBatch() returned %d refs, want 0", len(refs))
		}
	})
}

func TestPageAttachmentReferenceRepository_DeleteByPageAndAttachmentIDs(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewPageAttachmentReferenceRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("par-delete@example.com").
		WithAtname("pardelete").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("par-delete-space").
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

	attachmentID1 := testutil.NewAttachmentBuilder(t, tx).WithSpaceID(spaceID).WithSpaceMemberID(spaceMemberID).Build()
	attachmentID2 := testutil.NewAttachmentBuilder(t, tx).WithSpaceID(spaceID).WithSpaceMemberID(spaceMemberID).Build()
	attachmentID3 := testutil.NewAttachmentBuilder(t, tx).WithSpaceID(spaceID).WithSpaceMemberID(spaceMemberID).Build()

	// テストデータを作成
	_, err := repo.CreateBatch(context.Background(), pageID, spaceID, []model.AttachmentID{attachmentID1, attachmentID2, attachmentID3})
	if err != nil {
		t.Fatalf("CreateBatch() error = %v", err)
	}

	t.Run("指定した添付ファイルIDの参照のみ削除される", func(t *testing.T) {
		err := repo.DeleteByPageAndAttachmentIDs(context.Background(), pageID, spaceID, []model.AttachmentID{attachmentID1, attachmentID2})
		if err != nil {
			t.Fatalf("DeleteByPageAndAttachmentIDs() error = %v", err)
		}

		refs, err := repo.ListByPageID(context.Background(), pageID, spaceID)
		if err != nil {
			t.Fatalf("ListByPageID() error = %v", err)
		}
		if len(refs) != 1 {
			t.Fatalf("ListByPageID() returned %d refs, want 1", len(refs))
		}
		if refs[0].AttachmentID != attachmentID3 {
			t.Errorf("remaining ref.AttachmentID = %v, want %v", refs[0].AttachmentID, attachmentID3)
		}
	})
}
