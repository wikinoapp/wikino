package worker

import (
	"context"

	"github.com/riverqueue/river"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// SendPasswordResetWorker はパスワードリセットメール送信ワーカーです
type SendPasswordResetWorker struct {
	river.WorkerDefaults[dispatcher.SendPasswordResetArgs]
	uc *usecase.SendPasswordResetUsecase
}

// NewSendPasswordResetWorker は新しいSendPasswordResetWorkerを作成します
func NewSendPasswordResetWorker(uc *usecase.SendPasswordResetUsecase) *SendPasswordResetWorker {
	return &SendPasswordResetWorker{
		uc: uc,
	}
}

// Work はパスワードリセットメールを送信します
func (w *SendPasswordResetWorker) Work(ctx context.Context, job *river.Job[dispatcher.SendPasswordResetArgs]) error {
	return w.uc.Execute(ctx, usecase.SendPasswordResetInput{
		Email:    job.Args.Email,
		ResetURL: job.Args.ResetURL,
		AppURL:   job.Args.AppURL,
		Locale:   job.Args.Locale,
	})
}
