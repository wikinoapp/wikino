package worker

import (
	"context"

	"github.com/riverqueue/river"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// DeleteTopicWorkerは論理削除したトピックを物理削除するジョブを受け取り、UseCaseへ渡す。
type DeleteTopicWorker struct {
	river.WorkerDefaults[dispatcher.DeleteTopicArgs]
	uc *usecase.DeleteTopicUsecase
}

// NewDeleteTopicWorkerはDeleteTopicWorkerを生成する。
func NewDeleteTopicWorker(uc *usecase.DeleteTopicUsecase) *DeleteTopicWorker {
	return &DeleteTopicWorker{uc: uc}
}

// Workは論理削除したトピックを物理削除する。
func (w *DeleteTopicWorker) Work(ctx context.Context, job *river.Job[dispatcher.DeleteTopicArgs]) error {
	return w.uc.Execute(ctx, usecase.DeleteTopicInput{
		SpaceID: model.SpaceID(job.Args.SpaceID),
		TopicID: model.TopicID(job.Args.TopicID),
	})
}
