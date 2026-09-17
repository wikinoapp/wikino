package email

import (
	"context"

	"github.com/a-h/templ"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/templates/emails/password_reset"
)

// PasswordResetSenderはパスワードリセットメールの送信を行う
type PasswordResetSender struct {
	sender Sender
}

// NewPasswordResetSenderは新しいPasswordResetSenderを作成する
func NewPasswordResetSender(sender Sender) *PasswordResetSender {
	return &PasswordResetSender{sender: sender}
}

// Sendはパスワードリセットメールをレンダリングして送信する
func (s *PasswordResetSender) Send(ctx context.Context, to, resetURL, appURL, locale string) error {
	ctx = i18n.SetLocale(ctx, locale)
	subject := i18n.T(ctx, "password_reset_email_subject")

	data := password_reset.Data{
		Email:    to,
		ResetURL: resetURL,
		AppURL:   appURL,
	}

	var htmlBody, textBody templ.Component
	switch locale {
	case "ja":
		htmlBody = password_reset.JaHTML(data)
		textBody = password_reset.JaText(data)
	default:
		htmlBody = password_reset.EnHTML(data)
		textBody = password_reset.EnText(data)
	}

	return s.sender.Send(ctx, SendInput{
		To:       to,
		Subject:  subject,
		HTMLBody: htmlBody,
		TextBody: textBody,
	})
}
