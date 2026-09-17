package worker

import (
	"context"

	"github.com/riverqueue/river"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// SendEmailConfirmationWorkerはメール確認コード送信ワーカーです
type SendEmailConfirmationWorker struct {
	river.WorkerDefaults[dispatcher.SendEmailConfirmationArgs]
	uc *usecase.SendEmailConfirmationUsecase
}

// NewSendEmailConfirmationWorkerは新しいSendEmailConfirmationWorkerを作成します
func NewSendEmailConfirmationWorker(uc *usecase.SendEmailConfirmationUsecase) *SendEmailConfirmationWorker {
	return &SendEmailConfirmationWorker{
		uc: uc,
	}
}

// Workはメール確認コードを送信します
func (w *SendEmailConfirmationWorker) Work(ctx context.Context, job *river.Job[dispatcher.SendEmailConfirmationArgs]) error {
	return w.uc.Execute(ctx, usecase.SendEmailConfirmationInput{
		Email:  job.Args.Email,
		Code:   job.Args.Code,
		AppURL: job.Args.AppURL,
		Locale: job.Args.Locale,
	})
}
