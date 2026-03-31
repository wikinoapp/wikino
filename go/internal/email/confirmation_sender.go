package email

import (
	"context"

	"github.com/a-h/templ"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/templates/emails/email_confirmation"
)

// ConfirmationSender はメール確認コードのメール送信を行う
type ConfirmationSender struct {
	sender Sender
}

// NewConfirmationSender は新しい ConfirmationSender を作成する
func NewConfirmationSender(sender Sender) *ConfirmationSender {
	return &ConfirmationSender{sender: sender}
}

// Send はメール確認コードのメールをレンダリングして送信する
func (s *ConfirmationSender) Send(ctx context.Context, to, code, appURL, locale string) error {
	ctx = i18n.SetLocale(ctx, locale)
	subject := i18n.T(ctx, "email_confirmation_subject")

	data := email_confirmation.Data{
		Email:  to,
		Code:   code,
		AppURL: appURL,
	}

	var htmlBody, textBody templ.Component
	switch locale {
	case "ja":
		htmlBody = email_confirmation.JaHTML(data)
		textBody = email_confirmation.JaText(data)
	default:
		htmlBody = email_confirmation.EnHTML(data)
		textBody = email_confirmation.EnText(data)
	}

	return s.sender.Send(ctx, SendInput{
		To:       to,
		Subject:  subject,
		HTMLBody: htmlBody,
		TextBody: textBody,
	})
}
