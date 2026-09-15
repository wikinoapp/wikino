package worker

import (
	"context"
	"time"

	"github.com/riverqueue/river"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// GenerateExportFilesWorkerはスペースのエクスポートを書き出すジョブを受け取り、UseCaseへ渡す。
type GenerateExportFilesWorker struct {
	river.WorkerDefaults[dispatcher.GenerateExportFilesArgs]
	uc *usecase.GenerateExportFilesUsecase
}

// NewGenerateExportFilesWorkerはGenerateExportFilesWorkerを生成する。
func NewGenerateExportFilesWorker(uc *usecase.GenerateExportFilesUsecase) *GenerateExportFilesWorker {
	return &GenerateExportFilesWorker{
		uc: uc,
	}
}

// Timeoutは1回の試行の上限であり、ストレージとの接続が応答しないエクスポートがワーカーを
// 永久に占有せず打ち切られるようにする。クライアントのRescueStuckJobsAfterも同じ定数から決める。
// Riverは2つのうち後者を大きくすることを要求する。
func (w *GenerateExportFilesWorker) Timeout(_ *river.Job[dispatcher.GenerateExportFilesArgs]) time.Duration {
	return model.ExportAttemptTimeout
}

// Workはスペースのエクスポートを生成する。
func (w *GenerateExportFilesWorker) Work(ctx context.Context, job *river.Job[dispatcher.GenerateExportFilesArgs]) error {
	return w.uc.Execute(ctx, usecase.GenerateExportFilesInput{
		ExportID: model.ExportID(job.Args.ExportID),
		SpaceID:  model.SpaceID(job.Args.SpaceID),

		// 失敗を記録するのは、キューがその後に再試行しない試行のときだけなので、ワーカーが
		// どの試行なのかをUseCaseへ伝える。Riverは実行中の試行を数に含めるため、2つの数が並ぶ
		// ところが最後の試行になる。
		FinalAttempt: job.Attempt >= job.MaxAttempts,
	})
}
