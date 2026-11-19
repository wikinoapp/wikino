package worker

import (
	"context"

	"github.com/riverqueue/river"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// ProcessAttachmentWorkerは、添付ファイルのアップロード後の処理のジョブを受け取り、UseCaseへ渡す。
type ProcessAttachmentWorker struct {
	river.WorkerDefaults[dispatcher.ProcessAttachmentArgs]
	uc *usecase.ProcessAttachmentUsecase
}

// NewProcessAttachmentWorkerはProcessAttachmentWorkerを生成する。
func NewProcessAttachmentWorker(uc *usecase.ProcessAttachmentUsecase) *ProcessAttachmentWorker {
	return &ProcessAttachmentWorker{uc: uc}
}

// Workは添付ファイルを処理する。UseCaseがアップロードを待つ時間を返したときは、ジョブを先送りする。
// riverは先送りを試行の回数に数えないため、試行の回数はオブジェクトストレージの失敗の再試行にだけ使われる。
func (w *ProcessAttachmentWorker) Work(ctx context.Context, job *river.Job[dispatcher.ProcessAttachmentArgs]) error {
	out, err := w.uc.Execute(ctx, usecase.ProcessAttachmentInput{
		AttachmentID: model.AttachmentID(job.Args.AttachmentID),
		SpaceID:      model.SpaceID(job.Args.SpaceID),
	})
	if err != nil {
		return err
	}
	if out.RetryAfter > 0 {
		return river.JobSnooze(out.RetryAfter)
	}
	return nil
}
