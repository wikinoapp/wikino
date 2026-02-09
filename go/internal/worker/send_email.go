package worker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/riverqueue/river"

	"github.com/wikinoapp/wikino/go/internal/email"
)

// SendEmailArgs はメール送信ジョブの引数です
type SendEmailArgs struct {
	To       string `json:"to"`
	Subject  string `json:"subject"`
	HTMLBody string `json:"html_body"`
	TextBody string `json:"text_body"`
}

// Kind はジョブの種類を返します
func (SendEmailArgs) Kind() string {
	return "send_email"
}

// InsertOpts はジョブのInsertオプションを返します
func (SendEmailArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       river.QueueDefault,
		MaxAttempts: 5,
	}
}

// SendEmailWorker はメール送信ワーカーです
type SendEmailWorker struct {
	river.WorkerDefaults[SendEmailArgs]
	sender email.Sender
}

// NewSendEmailWorker は新しいSendEmailWorkerを作成します
func NewSendEmailWorker(sender email.Sender) *SendEmailWorker {
	return &SendEmailWorker{
		sender: sender,
	}
}

// Work はメールを送信します
func (w *SendEmailWorker) Work(ctx context.Context, job *river.Job[SendEmailArgs]) error {
	slog.InfoContext(ctx, "メール送信ジョブを開始します",
		"to", job.Args.To,
		"subject", job.Args.Subject,
	)

	// メールアドレスの検証
	if job.Args.To == "" {
		slog.ErrorContext(ctx, "メールアドレスが空です")
		return fmt.Errorf("メールアドレスが空です")
	}

	// メール送信
	err := w.sender.SendRaw(ctx, email.SendRawInput{
		To:       job.Args.To,
		Subject:  job.Args.Subject,
		HTMLBody: job.Args.HTMLBody,
		TextBody: job.Args.TextBody,
	})
	if err != nil {
		slog.ErrorContext(ctx, "メール送信に失敗しました",
			"to", job.Args.To,
			"error", err,
		)
		return fmt.Errorf("メール送信に失敗: %w", err)
	}

	slog.InfoContext(ctx, "メールを送信しました",
		"to", job.Args.To,
	)

	return nil
}
