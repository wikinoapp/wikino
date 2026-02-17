package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// createTestAttachmentForRepo はテスト用の添付ファイルを作成し、IDを返す
func createTestAttachmentForRepo(t *testing.T, tx *sql.Tx, spaceID model.SpaceID, spaceMemberID model.SpaceMemberID) model.AttachmentID {
	t.Helper()

	now := time.Now()

	var blobID string
	err := tx.QueryRowContext(
		context.Background(),
		`INSERT INTO active_storage_blobs (key, filename, content_type, service_name, byte_size, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		"test-key-"+now.Format("20060102150405.000000000"), "photo.jpg", "image/jpeg", "local", 2048, now,
	).Scan(&blobID)
	if err != nil {
		t.Fatalf("active_storage_blob作成に失敗: %v", err)
	}

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

	return model.AttachmentID(attachmentID)
}

func TestAttachmentRepository_ExistsByIDAndSpace(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewAttachmentRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("attach-exists@example.com").
		WithAtname("attachexists").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("attach-exists-space").
		Build()

	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()

	attachmentID := createTestAttachmentForRepo(t, tx, spaceID, spaceMemberID)

	t.Run("存在する添付ファイルはtrueを返す", func(t *testing.T) {
		exists, err := repo.ExistsByIDAndSpace(context.Background(), attachmentID, spaceID)
		if err != nil {
			t.Fatalf("ExistsByIDAndSpace() error = %v", err)
		}
		if !exists {
			t.Error("ExistsByIDAndSpace() = false, want true")
		}
	})

	t.Run("存在しないIDはfalseを返す", func(t *testing.T) {
		exists, err := repo.ExistsByIDAndSpace(context.Background(), model.AttachmentID("00000000-0000-0000-0000-000000000000"), spaceID)
		if err != nil {
			t.Fatalf("ExistsByIDAndSpace() error = %v", err)
		}
		if exists {
			t.Error("ExistsByIDAndSpace() = true, want false")
		}
	})

	t.Run("異なるスペースIDはfalseを返す", func(t *testing.T) {
		otherSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("attach-exists-other").
			Build()

		exists, err := repo.ExistsByIDAndSpace(context.Background(), attachmentID, otherSpaceID)
		if err != nil {
			t.Fatalf("ExistsByIDAndSpace() error = %v", err)
		}
		if exists {
			t.Error("ExistsByIDAndSpace() = true, want false")
		}
	})
}

func TestAttachmentRepository_FindByIDAndSpace(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewAttachmentRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("attach-find@example.com").
		WithAtname("attachfind").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("attach-find-space").
		Build()

	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()

	attachmentID := createTestAttachmentForRepo(t, tx, spaceID, spaceMemberID)

	t.Run("存在する添付ファイルを取得できる", func(t *testing.T) {
		attachment, err := repo.FindByIDAndSpace(context.Background(), attachmentID, spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace() error = %v", err)
		}
		if attachment == nil {
			t.Fatal("FindByIDAndSpace() returned nil, want attachment")
		}
		if attachment.ID != attachmentID {
			t.Errorf("attachment.ID = %v, want %v", attachment.ID, attachmentID)
		}
		if attachment.SpaceID != spaceID {
			t.Errorf("attachment.SpaceID = %v, want %v", attachment.SpaceID, spaceID)
		}
		if attachment.Filename != "photo.jpg" {
			t.Errorf("attachment.Filename = %v, want %v", attachment.Filename, "photo.jpg")
		}
	})

	t.Run("存在しないIDはnilを返す", func(t *testing.T) {
		attachment, err := repo.FindByIDAndSpace(context.Background(), model.AttachmentID("00000000-0000-0000-0000-000000000000"), spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace() error = %v", err)
		}
		if attachment != nil {
			t.Errorf("FindByIDAndSpace() = %v, want nil", attachment)
		}
	})

	t.Run("異なるスペースIDはnilを返す", func(t *testing.T) {
		otherSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("attach-find-other").
			Build()

		attachment, err := repo.FindByIDAndSpace(context.Background(), attachmentID, otherSpaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace() error = %v", err)
		}
		if attachment != nil {
			t.Errorf("FindByIDAndSpace() = %v, want nil", attachment)
		}
	})
}
