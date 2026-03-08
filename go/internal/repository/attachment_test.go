package repository

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

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

	attachmentID := testutil.NewAttachmentBuilder(t, tx).WithSpaceID(spaceID).WithSpaceMemberID(spaceMemberID).WithFilename("photo.jpg").WithContentType("image/jpeg").Build()

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

	attachmentID := testutil.NewAttachmentBuilder(t, tx).WithSpaceID(spaceID).WithSpaceMemberID(spaceMemberID).WithFilename("photo.jpg").WithContentType("image/jpeg").Build()

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

func TestAttachmentRepository_FindByIDsAndSpace(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewAttachmentRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("attach-findids@example.com").
		WithAtname("attachfindids").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("attach-findids-space").
		Build()

	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()

	attachmentID1 := testutil.NewAttachmentBuilder(t, tx).WithSpaceID(spaceID).WithSpaceMemberID(spaceMemberID).WithFilename("file1.png").Build()
	attachmentID2 := testutil.NewAttachmentBuilder(t, tx).WithSpaceID(spaceID).WithSpaceMemberID(spaceMemberID).WithFilename("file2.jpg").Build()

	t.Run("IDリストに含まれる添付ファイルを一括取得できる", func(t *testing.T) {
		attachments, err := repo.FindByIDsAndSpace(context.Background(), []model.AttachmentID{attachmentID1, attachmentID2}, spaceID)
		if err != nil {
			t.Fatalf("FindByIDsAndSpace() error = %v", err)
		}
		if len(attachments) != 2 {
			t.Fatalf("len(attachments) = %v, want 2", len(attachments))
		}
	})

	t.Run("空のIDリストは空のスライスを返す", func(t *testing.T) {
		attachments, err := repo.FindByIDsAndSpace(context.Background(), []model.AttachmentID{}, spaceID)
		if err != nil {
			t.Fatalf("FindByIDsAndSpace() error = %v", err)
		}
		if len(attachments) != 0 {
			t.Errorf("len(attachments) = %v, want 0", len(attachments))
		}
	})

	t.Run("異なるスペースIDの添付ファイルは取得されない", func(t *testing.T) {
		otherSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("attach-findids-other").
			Build()

		attachments, err := repo.FindByIDsAndSpace(context.Background(), []model.AttachmentID{attachmentID1, attachmentID2}, otherSpaceID)
		if err != nil {
			t.Fatalf("FindByIDsAndSpace() error = %v", err)
		}
		if len(attachments) != 0 {
			t.Errorf("len(attachments) = %v, want 0", len(attachments))
		}
	})
}
