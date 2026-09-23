package email

import (
	"context"

	"github.com/a-h/templ"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates/emails/export"
)

// ExportSenderはスペースのエクスポートが送るメールを送信する。
//
// 完了をメールで知らせるのは、エクスポートを開始したメンバーが画面を離れて待つためである。失敗も
// 同じ理由で知らせる。画面だけでは、たまたま戻って見た人にしか届かない。
type ExportSender struct {
	sender Sender
}

// NewExportSenderはExportSenderを生成する。
func NewExportSender(sender Sender) *ExportSender {
	return &ExportSender{sender: sender}
}

// SendSucceededはエクスポート完了メールをレンダリングして送信する。
func (s *ExportSender) SendSucceeded(ctx context.Context, to, downloadURL, appURL, locale string) error {
	data := export.Data{
		URL:             downloadURL,
		AppURL:          appURL,
		ExpirationHours: int(model.ExportDownloadExpiration.Hours()),
	}

	if locale == "ja" {
		return s.send(ctx, to, locale, "export_succeeded_email_subject", export.SucceededJaHTML(data), export.SucceededJaText(data))
	}
	return s.send(ctx, to, locale, "export_succeeded_email_subject", export.SucceededEnHTML(data), export.SucceededEnText(data))
}

// SendFailedはエクスポート失敗メールをレンダリングして送信する。
func (s *ExportSender) SendFailed(ctx context.Context, to, exportURL, appURL, locale string) error {
	data := export.Data{URL: exportURL, AppURL: appURL}

	if locale == "ja" {
		return s.send(ctx, to, locale, "export_failed_email_subject", export.FailedJaHTML(data), export.FailedJaText(data))
	}
	return s.send(ctx, to, locale, "export_failed_email_subject", export.FailedEnHTML(data), export.FailedEnText(data))
}

// sendはロケールを反映した件名で1通のメールを送信する。
func (s *ExportSender) send(ctx context.Context, to, locale, subjectKey string, htmlBody, textBody templ.Component) error {
	ctx = i18n.SetLocale(ctx, locale)

	return s.sender.Send(ctx, SendInput{
		To:       to,
		Subject:  i18n.T(ctx, subjectKey),
		HTMLBody: htmlBody,
		TextBody: textBody,
	})
}
