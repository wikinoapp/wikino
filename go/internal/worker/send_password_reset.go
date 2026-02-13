package worker

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"

	"github.com/riverqueue/river"

	"github.com/wikinoapp/wikino/go/internal/email"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/templates/emails/password_reset"
)

// SendPasswordResetArgs はパスワードリセットメール送信ジョブの引数です
type SendPasswordResetArgs struct {
	Email    string `json:"email"`
	ResetURL string `json:"reset_url"`
	AppURL   string `json:"app_url"`
	Locale   string `json:"locale"`
}

// Kind はジョブの種類を返します
func (SendPasswordResetArgs) Kind() string {
	return "send_password_reset"
}

// InsertOpts はジョブのInsertオプションを返します
func (SendPasswordResetArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       river.QueueDefault,
		MaxAttempts: 5,
	}
}

// SendPasswordResetWorker はパスワードリセットメール送信ワーカーです
type SendPasswordResetWorker struct {
	river.WorkerDefaults[SendPasswordResetArgs]
	sender email.Sender
}

// NewSendPasswordResetWorker は新しいSendPasswordResetWorkerを作成します
func NewSendPasswordResetWorker(sender email.Sender) *SendPasswordResetWorker {
	return &SendPasswordResetWorker{
		sender: sender,
	}
}

// Work はパスワードリセットメールを送信します
func (w *SendPasswordResetWorker) Work(ctx context.Context, job *river.Job[SendPasswordResetArgs]) error {
	args := job.Args

	slog.InfoContext(ctx, "パスワードリセットメール送信ジョブを開始します",
		"email", args.Email,
		"locale", args.Locale,
	)

	if args.Email == "" {
		slog.ErrorContext(ctx, "メールアドレスが空です")
		return fmt.Errorf("メールアドレスが空です")
	}

	// テンプレートをレンダリング
	htmlBody, textBody, err := w.renderEmailTemplates(ctx, args)
	if err != nil {
		slog.ErrorContext(ctx, "メールテンプレートのレンダリングに失敗しました",
			"email", args.Email,
			"error", err,
		)
		return fmt.Errorf("メールテンプレートのレンダリングに失敗: %w", err)
	}

	// ロケールをコンテキストに設定してメール件名を取得
	ctx = i18n.SetLocale(ctx, args.Locale)
	subject := i18n.T(ctx, "password_reset_email_subject")

	// メール送信
	err = w.sender.SendRaw(ctx, email.SendRawInput{
		To:       args.Email,
		Subject:  subject,
		HTMLBody: htmlBody,
		TextBody: textBody,
	})
	if err != nil {
		slog.ErrorContext(ctx, "パスワードリセットメール送信に失敗しました",
			"email", args.Email,
			"error", err,
		)
		return fmt.Errorf("メール送信に失敗: %w", err)
	}

	slog.InfoContext(ctx, "パスワードリセットメールを送信しました",
		"email", args.Email,
	)

	return nil
}

// renderEmailTemplates はロケールに基づいてメールテンプレートをレンダリングする
func (w *SendPasswordResetWorker) renderEmailTemplates(ctx context.Context, args SendPasswordResetArgs) (htmlBody, textBody string, err error) {
	data := password_reset.Data{
		Email:    args.Email,
		ResetURL: args.ResetURL,
		AppURL:   args.AppURL,
	}

	var htmlBuf bytes.Buffer
	var textBuf bytes.Buffer

	switch args.Locale {
	case "ja":
		if err := password_reset.JaHTML(data).Render(ctx, &htmlBuf); err != nil {
			return "", "", fmt.Errorf("HTMLテンプレートのレンダリングに失敗しました: %w", err)
		}
		if err := password_reset.JaText(data).Render(ctx, &textBuf); err != nil {
			return "", "", fmt.Errorf("テキストテンプレートのレンダリングに失敗しました: %w", err)
		}
	default:
		if err := password_reset.EnHTML(data).Render(ctx, &htmlBuf); err != nil {
			return "", "", fmt.Errorf("HTMLテンプレートのレンダリングに失敗しました: %w", err)
		}
		if err := password_reset.EnText(data).Render(ctx, &textBuf); err != nil {
			return "", "", fmt.Errorf("テキストテンプレートのレンダリングに失敗しました: %w", err)
		}
	}

	return htmlBuf.String(), textBuf.String(), nil
}
