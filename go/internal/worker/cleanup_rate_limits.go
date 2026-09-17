package worker

import (
	"context"

	"github.com/riverqueue/river"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// CleanupRateLimitsWorkerは古いRate Limitレコードを削除するワーカーです
type CleanupRateLimitsWorker struct {
	river.WorkerDefaults[dispatcher.CleanupRateLimitsArgs]
	uc *usecase.CleanupRateLimitsUsecase
}

// NewCleanupRateLimitsWorkerは新しいCleanupRateLimitsWorkerを作成します
func NewCleanupRateLimitsWorker(uc *usecase.CleanupRateLimitsUsecase) *CleanupRateLimitsWorker {
	return &CleanupRateLimitsWorker{
		uc: uc,
	}
}

// Workは古いRate Limitレコードを削除します
func (w *CleanupRateLimitsWorker) Work(ctx context.Context, job *river.Job[dispatcher.CleanupRateLimitsArgs]) error {
	return w.uc.Execute(ctx, usecase.CleanupRateLimitsInput{
		RetentionHours: job.Args.RetentionHours,
	})
}
