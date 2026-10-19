package worker

import (
	"context"

	"github.com/riverqueue/river"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// CleanupEmailConfirmationsWorkerは古いメール確認情報を削除するワーカーです
type CleanupEmailConfirmationsWorker struct {
	river.WorkerDefaults[dispatcher.CleanupEmailConfirmationsArgs]
	uc *usecase.CleanupEmailConfirmationsUsecase
}

// NewCleanupEmailConfirmationsWorkerは新しいCleanupEmailConfirmationsWorkerを作成します
func NewCleanupEmailConfirmationsWorker(uc *usecase.CleanupEmailConfirmationsUsecase) *CleanupEmailConfirmationsWorker {
	return &CleanupEmailConfirmationsWorker{
		uc: uc,
	}
}

// Workは古いメール確認情報を削除します
func (w *CleanupEmailConfirmationsWorker) Work(ctx context.Context, job *river.Job[dispatcher.CleanupEmailConfirmationsArgs]) error {
	return w.uc.Execute(ctx, usecase.CleanupEmailConfirmationsInput{
		RetentionHours: job.Args.RetentionHours,
	})
}
