package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

func TestProcessAttachmentWorker_Work(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("process-attachment-worker@example.com").
		WithAtname("process_attachment_worker").
		Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier("process-attachment-worker").Build()
	memberID := testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(spaceID).WithUserID(userID).Build()

	objectStorage := storage.NewFakeObjectStorage()
	w := NewProcessAttachmentWorker(usecase.NewProcessAttachmentUsecase(db, repository.NewAttachmentRepository(query.New(db)), objectStorage))

	work := func(id model.AttachmentID) error {
		return w.Work(context.Background(), &river.Job[dispatcher.ProcessAttachmentArgs]{
			Args: dispatcher.ProcessAttachmentArgs{AttachmentID: string(id), SpaceID: string(spaceID)},
		})
	}

	t.Run("アップロードされていなければ先送りする", func(t *testing.T) {
		t.Parallel()

		id := testutil.NewAttachmentBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSpaceMemberID(memberID).
			WithBlobKey("process-attachment-worker-missing").
			Build()

		var snooze *rivertype.JobSnoozeError
		if err := work(id); !errors.As(err, &snooze) {
			t.Fatalf("Work()のエラー = %v、期待値 = 先送り", err)
		}
		// 並行するテストで処理が遅れると、事前署名からの経過時間だけ先送りが延びる
		if snooze.Duration < 5*time.Second || snooze.Duration > time.Minute {
			t.Errorf("先送りする時間 = %v、期待値 = 5秒から1分", snooze.Duration)
		}
	})

	t.Run("処理を終えたらnilを返す", func(t *testing.T) {
		t.Parallel()

		const key = "process-attachment-worker-gif"
		id := testutil.NewAttachmentBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSpaceMemberID(memberID).
			WithContentType("image/gif").
			WithBlobKey(key).
			Build()
		objectStorage.Put(key, []byte("GIF89a"), "image/gif")

		if err := work(id); err != nil {
			t.Errorf("Work()のエラー = %v、期待値 = nil", err)
		}
	})
}
