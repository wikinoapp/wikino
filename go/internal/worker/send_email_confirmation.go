package worker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/a-h/templ"
	"github.com/riverqueue/river"

	"github.com/wikinoapp/wikino/go/internal/email"
	"github.com/wikinoapp/wikino/go/internal/templates/emails/email_confirmation"
)

// SendEmailConfirmationArgs はメール確認送信ジョブの引数です
type SendEmailConfirmationArgs struct {
	Email   string `json:"email"`
	Code    string `json:"code"`
	AppURL  string `json:"app_url"`
	Subject string `json:"subject"`
	Locale  string `json:"locale"`
}

// Kind はジョブの種類を返します
func (SendEmailConfirmationArgs) Kind() string {
	return "send_email_confirmation"
}

// SendEmailConfirmationWorker はメール確認送信ワーカーです
type SendEmailConfirmationWorker struct {
	river.WorkerDefaults[SendEmailConfirmationArgs]
	sender email.Sender
}

// NewSendEmailConfirmationWorker は新しいSendEmailConfirmationWorkerを作成します
func NewSendEmailConfirmationWorker(sender email.Sender) *SendEmailConfirmationWorker {
	return &SendEmailConfirmationWorker{
		sender: sender,
	}
}

// Work はメール確認メールを送信します
func (w *SendEmailConfirmationWorker) Work(ctx context.Context, job *river.Job[SendEmailConfirmationArgs]) error {
	slog.InfoContext(ctx, "メール確認送信ジョブを開始します",
		"email", job.Args.Email,
		"locale", job.Args.Locale,
	)

	// メールアドレスの検証
	if job.Args.Email == "" {
		slog.ErrorContext(ctx, "メールアドレスが空です")
		return fmt.Errorf("メールアドレスが空です")
	}

	// テンプレートデータを作成
	data := email_confirmation.Data{
		Email:  job.Args.Email,
		Code:   job.Args.Code,
		AppURL: job.Args.AppURL,
	}

	// ロケールに基づいてテンプレートを選択
	htmlBody, textBody := selectEmailTemplates(job.Args.Locale, data)

	// メール送信
	err := w.sender.Send(ctx, email.SendInput{
		To:       job.Args.Email,
		Subject:  job.Args.Subject,
		HTMLBody: htmlBody,
		TextBody: textBody,
	})
	if err != nil {
		slog.ErrorContext(ctx, "メール送信に失敗しました",
			"email", job.Args.Email,
			"error", err,
		)
		return fmt.Errorf("メール送信に失敗: %w", err)
	}

	slog.InfoContext(ctx, "メール確認メールを送信しました",
		"email", job.Args.Email,
	)

	return nil
}

// selectEmailTemplates はロケールに基づいてHTML/テキストテンプレートを選択する
func selectEmailTemplates(locale string, data email_confirmation.Data) (htmlBody, textBody templ.Component) {
	switch locale {
	case "ja":
		return email_confirmation.JaHTML(data), email_confirmation.JaText(data)
	default:
		return email_confirmation.EnHTML(data), email_confirmation.EnText(data)
	}
}
