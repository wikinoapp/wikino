package worker

import (
	"context"
	"time"

	"github.com/riverqueue/river"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// GenerateExportFilesWorker receives the job that writes a space export and hands it to the UseCase.
//
// [Ja] GenerateExportFilesWorker はスペースのエクスポートを書き出すジョブを受け取り、UseCase へ渡す。
type GenerateExportFilesWorker struct {
	river.WorkerDefaults[dispatcher.GenerateExportFilesArgs]
	uc *usecase.GenerateExportFilesUsecase
}

// NewGenerateExportFilesWorker creates a GenerateExportFilesWorker.
//
// [Ja] NewGenerateExportFilesWorker は GenerateExportFilesWorker を生成する。
func NewGenerateExportFilesWorker(uc *usecase.GenerateExportFilesUsecase) *GenerateExportFilesWorker {
	return &GenerateExportFilesWorker{
		uc: uc,
	}
}

// Timeout bounds one attempt, so that an export whose storage connection never answers is cut off
// instead of holding a worker forever. The client's RescueStuckJobsAfter is set from the same
// constant, which River requires to be the larger of the two.
//
// [Ja] Timeout は 1 回の試行の上限であり、ストレージとの接続が応答しないエクスポートがワーカーを
// 永久に占有せず打ち切られるようにする。クライアントの RescueStuckJobsAfter も同じ定数から決める。
// River は 2 つのうち後者を大きくすることを要求する。
func (w *GenerateExportFilesWorker) Timeout(_ *river.Job[dispatcher.GenerateExportFilesArgs]) time.Duration {
	return model.ExportAttemptTimeout
}

// Work generates the export of a space.
//
// [Ja] Work はスペースのエクスポートを生成する。
func (w *GenerateExportFilesWorker) Work(ctx context.Context, job *river.Job[dispatcher.GenerateExportFilesArgs]) error {
	return w.uc.Execute(ctx, usecase.GenerateExportFilesInput{
		ExportID: model.ExportID(job.Args.ExportID),
		SpaceID:  model.SpaceID(job.Args.SpaceID),

		// A failure is only recorded on the attempt the queue will not follow with another, so the
		// worker tells the UseCase which one this is. River counts the attempt that is running, so
		// the last one is where the two numbers meet.
		//
		// [Ja] 失敗を記録するのは、キューがその後に再試行しない試行のときだけなので、ワーカーが
		// どの試行なのかを UseCase へ伝える。River は実行中の試行を数に含めるため、2 つの数が並ぶ
		// ところが最後の試行になる。
		FinalAttempt: job.Attempt >= job.MaxAttempts,
	})
}
