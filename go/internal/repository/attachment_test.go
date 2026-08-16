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

func TestAttachmentRepository_FindPubliclyReferencedBlobByID(t *testing.T) {
	t.Parallel()

	t.Run("公開トピックのページから参照されている添付は blob 情報を取得できる", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		repo := NewAttachmentRepository(q)
		parRepo := NewPageAttachmentReferenceRepository(q)

		spaceID, spaceMemberID := testutil.SetupSpaceWithMember(t, tx, "attach-pub-1")
		topicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(1).
			WithName("public").
			WithVisibility(int32(model.TopicVisibilityPublic)).
			Build()
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("Page 1").
			Build()
		attachmentID := testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			WithFilename("og.png").
			WithContentType("image/png").
			Build()
		if _, err := parRepo.CreateBatch(context.Background(), pageID, spaceID, []model.AttachmentID{attachmentID}); err != nil {
			t.Fatalf("CreateBatch() error = %v", err)
		}

		attachment, err := repo.FindPubliclyReferencedBlobByID(context.Background(), attachmentID)
		if err != nil {
			t.Fatalf("FindPubliclyReferencedBlobByID() error = %v", err)
		}
		if attachment == nil {
			t.Fatal("FindPubliclyReferencedBlobByID() returned nil, want attachment")
		}
		if attachment.ID != attachmentID {
			t.Errorf("attachment.ID = %v, want %v", attachment.ID, attachmentID)
		}
		if attachment.SpaceID != spaceID {
			t.Errorf("attachment.SpaceID = %v, want %v", attachment.SpaceID, spaceID)
		}
		if attachment.BlobKey == "" {
			t.Error("attachment.BlobKey should not be empty")
		}
		if attachment.ContentType != "image/png" {
			t.Errorf("attachment.ContentType = %v, want image/png", attachment.ContentType)
		}
	})

	t.Run("非公開トピックのページから参照されている添付は nil を返す", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		repo := NewAttachmentRepository(q)
		parRepo := NewPageAttachmentReferenceRepository(q)

		spaceID, spaceMemberID := testutil.SetupSpaceWithMember(t, tx, "attach-pub-2")
		topicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(1).
			WithName("private").
			WithVisibility(int32(model.TopicVisibilityPrivate)).
			Build()
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("Page 1").
			Build()
		attachmentID := testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			Build()
		if _, err := parRepo.CreateBatch(context.Background(), pageID, spaceID, []model.AttachmentID{attachmentID}); err != nil {
			t.Fatalf("CreateBatch() error = %v", err)
		}

		attachment, err := repo.FindPubliclyReferencedBlobByID(context.Background(), attachmentID)
		if err != nil {
			t.Fatalf("FindPubliclyReferencedBlobByID() error = %v", err)
		}
		if attachment != nil {
			t.Errorf("FindPubliclyReferencedBlobByID() = %v, want nil", attachment)
		}
	})

	t.Run("公開と非公開のページから混在参照されている添付は nil を返す", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		repo := NewAttachmentRepository(q)
		parRepo := NewPageAttachmentReferenceRepository(q)

		spaceID, spaceMemberID := testutil.SetupSpaceWithMember(t, tx, "attach-pub-3")
		publicTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(1).
			WithName("public").
			WithVisibility(int32(model.TopicVisibilityPublic)).
			Build()
		privateTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(2).
			WithName("private").
			WithVisibility(int32(model.TopicVisibilityPrivate)).
			Build()
		publicPageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(publicTopicID).
			WithNumber(1).
			WithTitle("Public").
			Build()
		privatePageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(privateTopicID).
			WithNumber(2).
			WithTitle("Private").
			Build()
		attachmentID := testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			Build()
		if _, err := parRepo.CreateBatch(context.Background(), publicPageID, spaceID, []model.AttachmentID{attachmentID}); err != nil {
			t.Fatalf("CreateBatch() error = %v", err)
		}
		if _, err := parRepo.CreateBatch(context.Background(), privatePageID, spaceID, []model.AttachmentID{attachmentID}); err != nil {
			t.Fatalf("CreateBatch() error = %v", err)
		}

		attachment, err := repo.FindPubliclyReferencedBlobByID(context.Background(), attachmentID)
		if err != nil {
			t.Fatalf("FindPubliclyReferencedBlobByID() error = %v", err)
		}
		if attachment != nil {
			t.Errorf("FindPubliclyReferencedBlobByID() = %v, want nil", attachment)
		}
	})

	t.Run("どのページからも参照されていない添付は nil を返す", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		repo := NewAttachmentRepository(q)

		spaceID, spaceMemberID := testutil.SetupSpaceWithMember(t, tx, "attach-pub-4")
		attachmentID := testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			Build()

		attachment, err := repo.FindPubliclyReferencedBlobByID(context.Background(), attachmentID)
		if err != nil {
			t.Fatalf("FindPubliclyReferencedBlobByID() error = %v", err)
		}
		if attachment != nil {
			t.Errorf("FindPubliclyReferencedBlobByID() = %v, want nil", attachment)
		}
	})

	t.Run("論理削除されたページからの参照は無視される (有効な公開参照がない場合 nil)", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		repo := NewAttachmentRepository(q)
		parRepo := NewPageAttachmentReferenceRepository(q)

		spaceID, spaceMemberID := testutil.SetupSpaceWithMember(t, tx, "attach-pub-5")
		topicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(1).
			WithName("public").
			WithVisibility(int32(model.TopicVisibilityPublic)).
			Build()
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("Discarded").
			Build()
		attachmentID := testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			Build()
		if _, err := parRepo.CreateBatch(context.Background(), pageID, spaceID, []model.AttachmentID{attachmentID}); err != nil {
			t.Fatalf("CreateBatch() error = %v", err)
		}
		// ページを論理削除する
		if _, err := tx.ExecContext(context.Background(),
			"UPDATE pages SET discarded_at = NOW() WHERE id = $1", string(pageID)); err != nil {
			t.Fatalf("論理削除に失敗: %v", err)
		}

		attachment, err := repo.FindPubliclyReferencedBlobByID(context.Background(), attachmentID)
		if err != nil {
			t.Fatalf("FindPubliclyReferencedBlobByID() error = %v", err)
		}
		if attachment != nil {
			t.Errorf("FindPubliclyReferencedBlobByID() = %v, want nil", attachment)
		}
	})

	t.Run("論理削除されたトピックからの参照は無視される (有効な公開参照がない場合 nil)", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		repo := NewAttachmentRepository(q)
		parRepo := NewPageAttachmentReferenceRepository(q)

		spaceID, spaceMemberID := testutil.SetupSpaceWithMember(t, tx, "attach-pub-6")
		topicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(1).
			WithName("public").
			WithVisibility(int32(model.TopicVisibilityPublic)).
			WithDiscarded().
			Build()
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("Page in discarded topic").
			Build()
		attachmentID := testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			Build()
		if _, err := parRepo.CreateBatch(context.Background(), pageID, spaceID, []model.AttachmentID{attachmentID}); err != nil {
			t.Fatalf("CreateBatch() error = %v", err)
		}

		attachment, err := repo.FindPubliclyReferencedBlobByID(context.Background(), attachmentID)
		if err != nil {
			t.Fatalf("FindPubliclyReferencedBlobByID() error = %v", err)
		}
		if attachment != nil {
			t.Errorf("FindPubliclyReferencedBlobByID() = %v, want nil", attachment)
		}
	})

	t.Run("公開トピックのゴミ箱に入ったページから参照されている添付は nil を返す", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		repo := NewAttachmentRepository(q)
		parRepo := NewPageAttachmentReferenceRepository(q)

		spaceID, spaceMemberID := testutil.SetupSpaceWithMember(t, tx, "attach-pub-7")
		topicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(1).
			WithName("public").
			WithVisibility(int32(model.TopicVisibilityPublic)).
			Build()
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("Trashed").
			WithTrashed().
			Build()
		attachmentID := testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			Build()
		if _, err := parRepo.CreateBatch(context.Background(), pageID, spaceID, []model.AttachmentID{attachmentID}); err != nil {
			t.Fatalf("CreateBatch() error = %v", err)
		}

		attachment, err := repo.FindPubliclyReferencedBlobByID(context.Background(), attachmentID)
		if err != nil {
			t.Fatalf("FindPubliclyReferencedBlobByID() error = %v", err)
		}
		if attachment != nil {
			t.Errorf("FindPubliclyReferencedBlobByID() = %v, want nil", attachment)
		}
	})

	t.Run("非公開トピックのゴミ箱に入ったページからの参照は visibility 判定に影響しない", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		repo := NewAttachmentRepository(q)
		parRepo := NewPageAttachmentReferenceRepository(q)

		spaceID, spaceMemberID := testutil.SetupSpaceWithMember(t, tx, "attach-pub-8")
		publicTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(1).
			WithName("public").
			WithVisibility(int32(model.TopicVisibilityPublic)).
			Build()
		privateTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(2).
			WithName("private").
			WithVisibility(int32(model.TopicVisibilityPrivate)).
			Build()
		publicPageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(publicTopicID).
			WithNumber(1).
			WithTitle("Public").
			Build()
		trashedPrivatePageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(privateTopicID).
			WithNumber(2).
			WithTitle("Trashed private").
			WithTrashed().
			Build()
		attachmentID := testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			Build()
		if _, err := parRepo.CreateBatch(context.Background(), publicPageID, spaceID, []model.AttachmentID{attachmentID}); err != nil {
			t.Fatalf("CreateBatch() error = %v", err)
		}
		if _, err := parRepo.CreateBatch(context.Background(), trashedPrivatePageID, spaceID, []model.AttachmentID{attachmentID}); err != nil {
			t.Fatalf("CreateBatch() error = %v", err)
		}

		attachment, err := repo.FindPubliclyReferencedBlobByID(context.Background(), attachmentID)
		if err != nil {
			t.Fatalf("FindPubliclyReferencedBlobByID() error = %v", err)
		}
		if attachment == nil {
			t.Fatal("FindPubliclyReferencedBlobByID() returned nil, want attachment")
		}
	})

	// Pins `p.space_id = a.space_id` in the EXISTS branch: the topic stays in the attachment's
	// space, so only the page condition can exclude this reference.
	// [Ja] EXISTS 側の `p.space_id = a.space_id` を単独で固定する。トピックは attachment と
	// 同じ space に置くため、この参照を除外できるのはページ側の条件だけになる。
	t.Run("参照元ページが別 space にある参照だけでは添付を公開しない", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		repo := NewAttachmentRepository(q)

		attachmentSpaceID, attachmentSpaceMemberID := testutil.SetupSpaceWithMember(t, tx, "attach-pub-9-attachment")
		pageSpaceID, _ := testutil.SetupSpaceWithMember(t, tx, "attach-pub-9-page")
		publicTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(attachmentSpaceID).
			WithNumber(1).
			WithName("public").
			WithVisibility(int32(model.TopicVisibilityPublic)).
			Build()
		publicPageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(pageSpaceID).
			WithTopicID(publicTopicID).
			WithNumber(1).
			WithTitle("Foreign public").
			Build()
		attachmentID := testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(attachmentSpaceID).
			WithSpaceMemberID(attachmentSpaceMemberID).
			Build()
		// CreateBatch rejects a reference whose page and attachment live in different spaces.
		// [Ja] CreateBatch は page と attachment の space が異なる参照を作れないため直接 INSERT する。
		if _, err := tx.ExecContext(
			context.Background(),
			`INSERT INTO page_attachment_references (attachment_id, page_id, created_at, updated_at)
			 VALUES ($1, $2, NOW(), NOW())`,
			string(attachmentID),
			string(publicPageID),
		); err != nil {
			t.Fatalf("cross-space の参照作成に失敗: %v", err)
		}

		attachment, err := repo.FindPubliclyReferencedBlobByID(context.Background(), attachmentID)
		if err != nil {
			t.Fatalf("FindPubliclyReferencedBlobByID() error = %v", err)
		}
		if attachment != nil {
			t.Errorf("FindPubliclyReferencedBlobByID() = %v, want nil", attachment)
		}
	})

	// Pins `t.space_id = a.space_id` in the EXISTS branch: the page stays in the attachment's
	// space, so only the topic condition can exclude this reference.
	// [Ja] EXISTS 側の `t.space_id = a.space_id` を単独で固定する。ページは attachment と
	// 同じ space に置くため、この参照を除外できるのはトピック側の条件だけになる。
	t.Run("参照元ページのトピックが別 space にある参照だけでは添付を公開しない", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		repo := NewAttachmentRepository(q)
		parRepo := NewPageAttachmentReferenceRepository(q)

		attachmentSpaceID, attachmentSpaceMemberID := testutil.SetupSpaceWithMember(t, tx, "attach-pub-11-attachment")
		topicSpaceID, _ := testutil.SetupSpaceWithMember(t, tx, "attach-pub-11-topic")
		foreignTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(topicSpaceID).
			WithNumber(1).
			WithName("public").
			WithVisibility(int32(model.TopicVisibilityPublic)).
			Build()
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(attachmentSpaceID).
			WithTopicID(foreignTopicID).
			WithNumber(1).
			WithTitle("Page in a foreign topic").
			Build()
		attachmentID := testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(attachmentSpaceID).
			WithSpaceMemberID(attachmentSpaceMemberID).
			Build()
		if _, err := parRepo.CreateBatch(context.Background(), pageID, attachmentSpaceID, []model.AttachmentID{attachmentID}); err != nil {
			t.Fatalf("CreateBatch() error = %v", err)
		}

		attachment, err := repo.FindPubliclyReferencedBlobByID(context.Background(), attachmentID)
		if err != nil {
			t.Fatalf("FindPubliclyReferencedBlobByID() error = %v", err)
		}
		if attachment != nil {
			t.Errorf("FindPubliclyReferencedBlobByID() = %v, want nil", attachment)
		}
	})

	// Pins `p.space_id = a.space_id` in the NOT EXISTS branch: the private topic stays in the
	// attachment's space, so only the page condition can keep it out of the visibility check.
	// [Ja] NOT EXISTS 側の `p.space_id = a.space_id` を単独で固定する。非公開トピックは
	// attachment と同じ space に置くため、visibility 判定から外せるのはページ側の条件だけになる。
	t.Run("別 space のページからの非公開参照は visibility 判定に影響しない", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		repo := NewAttachmentRepository(q)
		parRepo := NewPageAttachmentReferenceRepository(q)

		attachmentSpaceID, attachmentSpaceMemberID := testutil.SetupSpaceWithMember(t, tx, "attach-pub-10-attachment")
		pageSpaceID, _ := testutil.SetupSpaceWithMember(t, tx, "attach-pub-10-page")
		publicTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(attachmentSpaceID).
			WithNumber(1).
			WithName("public").
			WithVisibility(int32(model.TopicVisibilityPublic)).
			Build()
		privateTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(attachmentSpaceID).
			WithNumber(2).
			WithName("private").
			WithVisibility(int32(model.TopicVisibilityPrivate)).
			Build()
		publicPageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(attachmentSpaceID).
			WithTopicID(publicTopicID).
			WithNumber(1).
			WithTitle("Public").
			Build()
		privatePageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(pageSpaceID).
			WithTopicID(privateTopicID).
			WithNumber(1).
			WithTitle("Foreign private").
			Build()
		attachmentID := testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(attachmentSpaceID).
			WithSpaceMemberID(attachmentSpaceMemberID).
			Build()
		if _, err := parRepo.CreateBatch(context.Background(), publicPageID, attachmentSpaceID, []model.AttachmentID{attachmentID}); err != nil {
			t.Fatalf("CreateBatch() error = %v", err)
		}
		// CreateBatch rejects a reference whose page and attachment live in different spaces.
		// [Ja] CreateBatch は page と attachment の space が異なる参照を作れないため直接 INSERT する。
		if _, err := tx.ExecContext(
			context.Background(),
			`INSERT INTO page_attachment_references (attachment_id, page_id, created_at, updated_at)
			 VALUES ($1, $2, NOW(), NOW())`,
			string(attachmentID),
			string(privatePageID),
		); err != nil {
			t.Fatalf("cross-space の参照作成に失敗: %v", err)
		}

		attachment, err := repo.FindPubliclyReferencedBlobByID(context.Background(), attachmentID)
		if err != nil {
			t.Fatalf("FindPubliclyReferencedBlobByID() error = %v", err)
		}
		if attachment == nil {
			t.Fatal("FindPubliclyReferencedBlobByID() returned nil, want attachment")
		}
		if attachment.SpaceID != attachmentSpaceID {
			t.Errorf("attachment.SpaceID = %v, want %v", attachment.SpaceID, attachmentSpaceID)
		}
	})

	// Pins `t.space_id = a.space_id` in the NOT EXISTS branch: the private page stays in the
	// attachment's space, so only the topic condition can keep it out of the visibility check.
	// [Ja] NOT EXISTS 側の `t.space_id = a.space_id` を単独で固定する。非公開ページは
	// attachment と同じ space に置くため、visibility 判定から外せるのはトピック側の条件だけになる。
	t.Run("別 space のトピックに属する非公開参照は visibility 判定に影響しない", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		repo := NewAttachmentRepository(q)
		parRepo := NewPageAttachmentReferenceRepository(q)

		attachmentSpaceID, attachmentSpaceMemberID := testutil.SetupSpaceWithMember(t, tx, "attach-pub-12-attachment")
		topicSpaceID, _ := testutil.SetupSpaceWithMember(t, tx, "attach-pub-12-topic")
		publicTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(attachmentSpaceID).
			WithNumber(1).
			WithName("public").
			WithVisibility(int32(model.TopicVisibilityPublic)).
			Build()
		foreignPrivateTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(topicSpaceID).
			WithNumber(1).
			WithName("private").
			WithVisibility(int32(model.TopicVisibilityPrivate)).
			Build()
		publicPageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(attachmentSpaceID).
			WithTopicID(publicTopicID).
			WithNumber(1).
			WithTitle("Public").
			Build()
		privatePageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(attachmentSpaceID).
			WithTopicID(foreignPrivateTopicID).
			WithNumber(2).
			WithTitle("Page in a foreign private topic").
			Build()
		attachmentID := testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(attachmentSpaceID).
			WithSpaceMemberID(attachmentSpaceMemberID).
			Build()
		if _, err := parRepo.CreateBatch(context.Background(), publicPageID, attachmentSpaceID, []model.AttachmentID{attachmentID}); err != nil {
			t.Fatalf("CreateBatch() error = %v", err)
		}
		if _, err := parRepo.CreateBatch(context.Background(), privatePageID, attachmentSpaceID, []model.AttachmentID{attachmentID}); err != nil {
			t.Fatalf("CreateBatch() error = %v", err)
		}

		attachment, err := repo.FindPubliclyReferencedBlobByID(context.Background(), attachmentID)
		if err != nil {
			t.Fatalf("FindPubliclyReferencedBlobByID() error = %v", err)
		}
		if attachment == nil {
			t.Fatal("FindPubliclyReferencedBlobByID() returned nil, want attachment")
		}
		if attachment.SpaceID != attachmentSpaceID {
			t.Errorf("attachment.SpaceID = %v, want %v", attachment.SpaceID, attachmentSpaceID)
		}
	})

	t.Run("存在しないIDはnilを返す", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		repo := NewAttachmentRepository(q)

		attachment, err := repo.FindPubliclyReferencedBlobByID(context.Background(), model.AttachmentID("00000000-0000-0000-0000-000000000000"))
		if err != nil {
			t.Fatalf("FindPubliclyReferencedBlobByID() error = %v", err)
		}
		if attachment != nil {
			t.Errorf("FindPubliclyReferencedBlobByID() = %v, want nil", attachment)
		}
	})

	t.Run("UUID形式でないIDはnilを返す (DBアクセスを行わない)", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		repo := NewAttachmentRepository(q)

		attachment, err := repo.FindPubliclyReferencedBlobByID(context.Background(), model.AttachmentID("not-a-uuid"))
		if err != nil {
			t.Fatalf("FindPubliclyReferencedBlobByID() error = %v", err)
		}
		if attachment != nil {
			t.Errorf("FindPubliclyReferencedBlobByID() = %v, want nil", attachment)
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
