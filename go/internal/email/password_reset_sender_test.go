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
		t.Fatalf("Send() error = %v", err)
	}

	if len(noop.SentEmails) != 1 {
		t.Fatalf("SentEmails length = %d, want 1", len(noop.SentEmails))
	}

	sent := noop.SentEmails[0]
	if sent.To != "test@example.com" {
		t.Errorf("To = %s, want test@example.com", sent.To)
	}
	if sent.Subject == "" {
		t.Error("Subject が空です")
	}
	if sent.HTMLBody == nil {
		t.Error("HTMLBody が nil です")
	}
	if sent.TextBody == nil {
		t.Error("TextBody が nil です")
	}
}

func TestPasswordResetSender_Send_English(t *testing.T) {
	t.Parallel()

	noop := NewNoopSender()
	sender := NewPasswordResetSender(noop)

	ctx := i18n.SetLocale(context.Background(), "en")
	err := sender.Send(ctx, "test@example.com", "https://example.dev/password/edit?token=abc123", "https://example.dev", "en")
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if len(noop.SentEmails) != 1 {
		t.Fatalf("SentEmails length = %d, want 1", len(noop.SentEmails))
	}

	sent := noop.SentEmails[0]
	if sent.To != "test@example.com" {
		t.Errorf("To = %s, want test@example.com", sent.To)
	}
	if sent.Subject == "" {
		t.Error("Subject が空です")
	}
	if sent.HTMLBody == nil {
		t.Error("HTMLBody が nil です")
	}
	if sent.TextBody == nil {
		t.Error("TextBody が nil です")
	}
}
