package worker

import (
	"context"

	"github.com/riverqueue/river"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// CleanupExportObjectsWorkerは、ZIPの後始末のジョブをUseCaseへ渡す。
type CleanupExportObjectsWorker struct {
	river.WorkerDefaults[dispatcher.CleanupExportObjectsArgs]
	uc *usecase.CleanupExportObjectsUsecase
}

// NewCleanupExportObjectsWorkerはCleanupExportObjectsWorkerを生成する。
func NewCleanupExportObjectsWorker(uc *usecase.CleanupExportObjectsUsecase) *CleanupExportObjectsWorker {
	return &CleanupExportObjectsWorker{uc: uc}
}

// Workは、生成中なら先送りし、生成が終わっていればZIPを回収する。
func (w *CleanupExportObjectsWorker) Work(ctx context.Context, job *river.Job[dispatcher.CleanupExportObjectsArgs]) error {
	blobs := make([]model.LegacyExportBlob, len(job.Args.LegacyBlobs))
	for i, blob := range job.Args.LegacyBlobs {
		blobs[i] = model.LegacyExportBlob{ID: blob.ID, Key: blob.Key}
	}
	wait, err := w.uc.Execute(ctx, usecase.CleanupExportObjectsInput{
		SpaceID: job.Args.SpaceID, ExportIDs: job.Args.ExportIDs, Keys: job.Args.Keys, LegacyBlobs: blobs,
	})
	if err != nil {
		return err
	}
	if wait > 0 {
		return river.JobSnooze(wait)
	}
	return nil
}
