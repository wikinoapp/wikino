package email

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
)

func TestPasswordResetSender_Send_Japanese(t *testing.T) {
	t.Parallel()

	noop := NewNoopSender()
	sender := NewPasswordResetSender(noop)

	ctx := i18n.SetLocale(context.Background(), "ja")
	err := sender.Send(ctx, "test@example.com", "https://example.dev/password/edit?token=abc123", "https://example.dev", "ja")
	if err != nil {
		t.Fatalf("Send()のエラー = %v", err)
	}

	if len(noop.SentEmails) != 1 {
		t.Fatalf("SentEmailsの件数 = %d、期待値 = 1", len(noop.SentEmails))
	}

	sent := noop.SentEmails[0]
	if sent.To != "test@example.com" {
		t.Errorf("To = %s、期待値 = test@example.com", sent.To)
	}
	if sent.Subject == "" {
		t.Error("Subjectが空です")
	}
	if sent.HTMLBody == nil {
		t.Error("HTMLBodyがnilです")
	}
	if sent.TextBody == nil {
		t.Error("TextBodyがnilです")
	}
}

func TestPasswordResetSender_Send_English(t *testing.T) {
	t.Parallel()

	noop := NewNoopSender()
	sender := NewPasswordResetSender(noop)

	ctx := i18n.SetLocale(context.Background(), "en")
	err := sender.Send(ctx, "test@example.com", "https://example.dev/password/edit?token=abc123", "https://example.dev", "en")
	if err != nil {
		t.Fatalf("Send()のエラー = %v", err)
	}

	if len(noop.SentEmails) != 1 {
		t.Fatalf("SentEmailsの件数 = %d、期待値 = 1", len(noop.SentEmails))
	}

	sent := noop.SentEmails[0]
	if sent.To != "test@example.com" {
		t.Errorf("To = %s、期待値 = test@example.com", sent.To)
	}
	if sent.Subject == "" {
		t.Error("Subjectが空です")
	}
	if sent.HTMLBody == nil {
		t.Error("HTMLBodyがnilです")
	}
	if sent.TextBody == nil {
		t.Error("TextBodyがnilです")
	}
}
