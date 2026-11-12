package worker

import (
	"context"

	"github.com/riverqueue/river"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// DeleteTrashedPagesWorkerは、ゴミ箱に入れて保持期間を過ぎたページを物理削除する定期ジョブを受け取り、
// UseCaseへ渡す。
type DeleteTrashedPagesWorker struct {
	river.WorkerDefaults[dispatcher.DeleteTrashedPagesArgs]
	uc *usecase.DeleteTrashedPagesUsecase
}

// NewDeleteTrashedPagesWorkerはDeleteTrashedPagesWorkerを生成する。
func NewDeleteTrashedPagesWorker(uc *usecase.DeleteTrashedPagesUsecase) *DeleteTrashedPagesWorker {
	return &DeleteTrashedPagesWorker{uc: uc}
}

// Workは、ゴミ箱に入れて保持期間を過ぎたページを物理削除する。
func (w *DeleteTrashedPagesWorker) Work(ctx context.Context, _ *river.Job[dispatcher.DeleteTrashedPagesArgs]) error {
	return w.uc.Execute(ctx, usecase.DeleteTrashedPagesInput{})
}
