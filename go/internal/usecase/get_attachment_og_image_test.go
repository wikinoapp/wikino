package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetAttachmentOgImageUsecase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("正常系: 公開トピックのページから参照されている添付の blob 情報を返す", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		attachmentRepo := repository.NewAttachmentRepository(q)
		parRepo := repository.NewPageAttachmentReferenceRepository(q)
		uc := NewGetAttachmentOgImageUsecase(attachmentRepo)

		spaceID, spaceMemberID := testutil.SetupSpaceWithMember(t, tx, "ogimg-uc-1")
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
			WithTitle("Page").
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

		output, err := uc.Execute(context.Background(), GetAttachmentOgImageInput{AttachmentID: attachmentID})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil || output.Attachment == nil {
			t.Fatal("Execute() returned nil attachment")
		}
		if output.Attachment.ID != attachmentID {
			t.Errorf("Attachment.ID = %v, want %v", output.Attachment.ID, attachmentID)
		}
		if output.Attachment.BlobKey == "" {
			t.Error("Attachment.BlobKey should not be empty")
		}
	})

	t.Run("異常系: 非公開トピックのページからの参照のみは AppErrCodeResourceNotFound を返す", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		attachmentRepo := repository.NewAttachmentRepository(q)
		parRepo := repository.NewPageAttachmentReferenceRepository(q)
		uc := NewGetAttachmentOgImageUsecase(attachmentRepo)

		spaceID, spaceMemberID := testutil.SetupSpaceWithMember(t, tx, "ogimg-uc-2")
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
			WithTitle("Page").
			Build()
		attachmentID := testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			Build()
		if _, err := parRepo.CreateBatch(context.Background(), pageID, spaceID, []model.AttachmentID{attachmentID}); err != nil {
			t.Fatalf("CreateBatch() error = %v", err)
		}

		output, err := uc.Execute(context.Background(), GetAttachmentOgImageInput{AttachmentID: attachmentID})
		if output != nil {
			t.Errorf("Execute() output = %v, want nil", output)
		}
		assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
	})

	t.Run("異常系: どのページからも参照されていない添付は AppErrCodeResourceNotFound を返す", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		attachmentRepo := repository.NewAttachmentRepository(q)
		uc := NewGetAttachmentOgImageUsecase(attachmentRepo)

		spaceID, spaceMemberID := testutil.SetupSpaceWithMember(t, tx, "ogimg-uc-3")
		attachmentID := testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			Build()

		_, err := uc.Execute(context.Background(), GetAttachmentOgImageInput{AttachmentID: attachmentID})
		assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
	})

	t.Run("異常系: 存在しない attachment_id は AppErrCodeResourceNotFound を返す", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		attachmentRepo := repository.NewAttachmentRepository(q)
		uc := NewGetAttachmentOgImageUsecase(attachmentRepo)

		_, err := uc.Execute(context.Background(), GetAttachmentOgImageInput{
			AttachmentID: model.AttachmentID("00000000-0000-0000-0000-000000000000"),
		})
		assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
	})

	t.Run("異常系: UUID 形式でない ID も AppErrCodeResourceNotFound を返す (DB アクセスせず)", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		attachmentRepo := repository.NewAttachmentRepository(q)
		uc := NewGetAttachmentOgImageUsecase(attachmentRepo)

		_, err := uc.Execute(context.Background(), GetAttachmentOgImageInput{
			AttachmentID: model.AttachmentID("not-a-uuid"),
		})
		assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
	})
}
