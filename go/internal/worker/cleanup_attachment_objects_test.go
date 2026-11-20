package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/storage"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

func TestCleanupAttachmentObjectsWorker_Work(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	jobClient, jobSchema := testutil.SetupRiverClient(t, db)
	objectStorage := storage.NewFakeObjectStorage()
	w := NewCleanupAttachmentObjectsWorker(usecase.NewCleanupAttachmentObjectsUsecase(
		dispatcher.NewDispatcher(jobClient, jobClient), objectStorage,
	))

	const spaceID = "cleanup-attachment-objects-worker-space"
	work := func(attachmentID, key string) error {
		return w.Work(context.Background(), &river.Job[dispatcher.CleanupAttachmentObjectsArgs]{
			Args: dispatcher.CleanupAttachmentObjectsArgs{SpaceID: spaceID, AttachmentIDs: []string{attachmentID}, Keys: []string{key}},
		})
	}

	t.Run("処理ジョブが残っていれば先送りする", func(t *testing.T) {
		t.Parallel()

		const attachmentID, key = "cleanup-worker-processing", "cleanup-worker-processing-key"
		objectStorage.Put(key, []byte("written back"), "image/png")
		job, err := jobClient.Insert(t.Context(), dispatcher.ProcessAttachmentArgs{AttachmentID: attachmentID, SpaceID: spaceID}, nil)
		if err != nil {
			t.Fatalf("処理ジョブの登録に失敗: %v", err)
		}

		var snooze *rivertype.JobSnoozeError
		if err := work(attachmentID, key); !errors.As(err, &snooze) {
			t.Fatalf("Work()のエラー = %v、期待値 = 先送り", err)
		}
		if snooze.Duration <= 0 {
			t.Errorf("先送りする時間 = %v、期待値 = 0より長い時間", snooze.Duration)
		}
		if _, _, exists := objectStorage.Object(key); !exists {
			t.Error("処理ジョブが残っている間に実体を消した")
		}

		// 処理ジョブが終わったあとは、先送りせずに実体を消す
		if _, err := db.ExecContext(t.Context(), "UPDATE "+jobSchema+".river_job SET state = $1, finalized_at = NOW() WHERE id = $2",
			rivertype.JobStateCompleted, job.Job.ID); err != nil {
			t.Fatalf("処理ジョブの終了に失敗: %v", err)
		}
		if err := work(attachmentID, key); err != nil {
			t.Errorf("Work()のエラー = %v、期待値 = nil", err)
		}
		if _, _, exists := objectStorage.Object(key); exists {
			t.Error("処理ジョブが終わっても実体が残っている")
		}
	})

	t.Run("処理ジョブが無ければ実体を消してnilを返す", func(t *testing.T) {
		t.Parallel()

		const attachmentID, key = "cleanup-worker-idle", "cleanup-worker-idle-key"
		objectStorage.Put(key, []byte("variant"), "image/png")

		if err := work(attachmentID, key); err != nil {
			t.Errorf("Work()のエラー = %v、期待値 = nil", err)
		}
		if _, _, exists := objectStorage.Object(key); exists {
			t.Error("実体が残っている")
		}
	})
}
