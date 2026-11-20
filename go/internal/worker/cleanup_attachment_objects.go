package worker

import (
	"context"

	"github.com/riverqueue/river"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// CleanupAttachmentObjectsWorkerは、削除した添付ファイルの実体を回収するジョブをUseCaseへ渡す。
type CleanupAttachmentObjectsWorker struct {
	river.WorkerDefaults[dispatcher.CleanupAttachmentObjectsArgs]
	uc *usecase.CleanupAttachmentObjectsUsecase
}

// NewCleanupAttachmentObjectsWorkerはCleanupAttachmentObjectsWorkerを生成する。
func NewCleanupAttachmentObjectsWorker(uc *usecase.CleanupAttachmentObjectsUsecase) *CleanupAttachmentObjectsWorker {
	return &CleanupAttachmentObjectsWorker{uc: uc}
}

// Workは、処理ジョブが終了するまで先送りし、終了したら実体を回収する。
func (w *CleanupAttachmentObjectsWorker) Work(ctx context.Context, job *river.Job[dispatcher.CleanupAttachmentObjectsArgs]) error {
	wait, err := w.uc.Execute(ctx, usecase.CleanupAttachmentObjectsInput{
		SpaceID: job.Args.SpaceID, AttachmentIDs: job.Args.AttachmentIDs, Keys: job.Args.Keys,
	})
	if err != nil {
		return err
	}
	if wait > 0 {
		return river.JobSnooze(wait)
	}
	return nil
}
