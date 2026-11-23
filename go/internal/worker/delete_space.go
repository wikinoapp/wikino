package worker

import (
	"context"
	"time"

	"github.com/riverqueue/river"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// deleteSpaceAttemptTimeoutは、スペースの物理削除の1回の試行の上限。実体の削除は添付ファイルの数だけ
// オブジェクトストレージへのリクエストを伴い、Riverの既定の上限 (1分) では添付ファイルの多いスペースを
// 消し切れないため、長く取る。打ち切られた試行は、再試行で最初から実体を消し直す。クライアントの
// RescueStuckJobsAfterは、この値より長くなければならない
const deleteSpaceAttemptTimeout = 30 * time.Minute

// DeleteSpaceWorkerは論理削除したスペースを物理削除するジョブを受け取り、UseCaseへ渡す。
type DeleteSpaceWorker struct {
	river.WorkerDefaults[dispatcher.DeleteSpaceArgs]
	uc *usecase.DeleteSpaceUsecase
}

// NewDeleteSpaceWorkerはDeleteSpaceWorkerを生成する。
func NewDeleteSpaceWorker(uc *usecase.DeleteSpaceUsecase) *DeleteSpaceWorker {
	return &DeleteSpaceWorker{uc: uc}
}

// Timeoutは1回の試行の上限を返す。
func (w *DeleteSpaceWorker) Timeout(_ *river.Job[dispatcher.DeleteSpaceArgs]) time.Duration {
	return deleteSpaceAttemptTimeout
}

// Workは論理削除したスペースを物理削除する。
func (w *DeleteSpaceWorker) Work(ctx context.Context, job *river.Job[dispatcher.DeleteSpaceArgs]) error {
	return w.uc.Execute(ctx, usecase.DeleteSpaceInput{
		SpaceID: model.SpaceID(job.Args.SpaceID),
	})
}
