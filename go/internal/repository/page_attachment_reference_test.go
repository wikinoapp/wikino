package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// createTestAttachment はテスト用の添付ファイルを作成し、IDを返す
func createTestAttachment(t *testing.T, tx *sql.Tx, spaceID model.SpaceID, spaceMemberID model.SpaceMemberID) string {
	t.Helper()

	now := time.Now()

	// active_storage_blobを作成
	var blobID string
	err := tx.QueryRowContext(
		context.Background(),
		`INSERT INTO active_storage_blobs (key, filename, content_type, service_name, byte_size, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		"test-key-"+now.Format("20060102150405.000000000"), "test.png", "image/png", "local", 1024, now,
	).Scan(&blobID)
	if err != nil {
		t.Fatalf("active_storage_blob作成に失敗: %v", err)
	}

	// active_storage_attachmentを作成
	var attachmentStorageID string
	err = tx.QueryRowContext(
		context.Background(),
		`INSERT INTO active_storage_attachments (name, record_type, record_id, blob_id, created_at)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id`,
		"file", "Space", string(spaceID), blobID, now,
	).Scan(&attachmentStorageID)
	if err != nil {
		t.Fatalf("active_storage_attachment作成に失敗: %v", err)
	}

	// attachmentを作成
	var attachmentID string
	err = tx.QueryRowContext(
		context.Background(),
		`INSERT INTO attachments (space_id, active_storage_attachment_id, attached_space_member_id, attached_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		string(spaceID), attachmentStorageID, string(spaceMemberID), now, now, now,
	).Scan(&attachmentID)
	if err != nil {
		t.Fatalf("attachment作成に失敗: %v", err)
	}

	return attachmentID
}

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

	attachmentID1 := createTestAttachment(t, tx, spaceID, spaceMemberID)
	attachmentID2 := createTestAttachment(t, tx, spaceID, spaceMemberID)

	// テストデータを作成
	_, err := repo.CreateBatch(context.Background(), pageID, []string{attachmentID1, attachmentID2})
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
		attachmentID1 := createTestAttachment(t, tx, spaceID, spaceMemberID)
		attachmentID2 := createTestAttachment(t, tx, spaceID, spaceMemberID)

		refs, err := repo.CreateBatch(context.Background(), pageID, []string{attachmentID1, attachmentID2})
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
		refs, err := repo.CreateBatch(context.Background(), pageID, []string{})
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

	attachmentID1 := createTestAttachment(t, tx, spaceID, spaceMemberID)
	attachmentID2 := createTestAttachment(t, tx, spaceID, spaceMemberID)
	attachmentID3 := createTestAttachment(t, tx, spaceID, spaceMemberID)

	// テストデータを作成
	_, err := repo.CreateBatch(context.Background(), pageID, []string{attachmentID1, attachmentID2, attachmentID3})
	if err != nil {
		t.Fatalf("CreateBatch() error = %v", err)
	}

	t.Run("指定した添付ファイルIDの参照のみ削除される", func(t *testing.T) {
		err := repo.DeleteByPageAndAttachmentIDs(context.Background(), pageID, []string{attachmentID1, attachmentID2})
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
