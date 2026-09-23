package email

import (
	"context"
	"testing"
)

// ResendSenderがSenderインターフェースを実装していることを確認
var _ Sender = (*ResendSender)(nil)

// NoopSenderがSenderインターフェースを実装していることを確認
var _ Sender = (*NoopSender)(nil)

func TestNewResendSender(t *testing.T) {
	t.Parallel()

	sender := NewResendSender("test-api-key", "test@example.com", "Wikino")

	if sender == nil {
		t.Fatal("NewResendSender()がnilを返した")
	}
	if sender.client == nil {
		t.Error("clientがnil")
	}
	if sender.fromEmail != "test@example.com" {
		t.Errorf("fromEmail = %s、期待値 = test@example.com", sender.fromEmail)
	}
	if sender.fromName != "Wikino" {
		t.Errorf("fromName = %s、期待値 = Wikino", sender.fromName)
	}
}

func TestResendSender_from(t *testing.T) {
	t.Parallel()

	t.Run("fromNameが設定されている場合", func(t *testing.T) {
		t.Parallel()

		sender := NewResendSender("test-api-key", "noreply@wikino.app", "Wikino")
		got := sender.from()
		want := "Wikino <noreply@wikino.app>"
		if got != want {
			t.Errorf("from() = %s、期待値 = %s", got, want)
		}
	})

	t.Run("fromNameが空の場合", func(t *testing.T) {
		t.Parallel()

		sender := NewResendSender("test-api-key", "noreply@wikino.app", "")
		got := sender.from()
		want := "noreply@wikino.app"
		if got != want {
			t.Errorf("from() = %s、期待値 = %s", got, want)
		}
	})
}

func TestSendInput_Fields(t *testing.T) {
	t.Parallel()

	input := SendInput{
		To:       "recipient@example.com",
		Subject:  "Test Subject",
		HTMLBody: nil, // templ.ComponentはnilでもOK
	}

	if input.To != "recipient@example.com" {
		t.Errorf("To = %s、期待値 = recipient@example.com", input.To)
	}
	if input.Subject != "Test Subject" {
		t.Errorf("Subject = %s、期待値 = Test Subject", input.Subject)
	}
}

func TestNewNoopSender(t *testing.T) {
	t.Parallel()

	sender := NewNoopSender()

	if sender == nil {
		t.Fatal("NewNoopSender()がnilを返した")
	}
	if sender.SentEmails == nil {
		t.Error("SentEmailsがnil")
	}
	if len(sender.SentEmails) != 0 {
		t.Errorf("SentEmailsの件数 = %d、期待値 = 0", len(sender.SentEmails))
	}
}

func TestNoopSender_Send(t *testing.T) {
	t.Parallel()

	sender := NewNoopSender()
	ctx := context.Background()

	// 1通目を送信
	input1 := SendInput{
		To:      "user1@example.com",
		Subject: "Subject 1",
	}
	err := sender.Send(ctx, input1)
	if err != nil {
		t.Fatalf("Send()のエラー = %v", err)
	}

	if len(sender.SentEmails) != 1 {
		t.Fatalf("SentEmailsの件数 = %d、期待値 = 1", len(sender.SentEmails))
	}
	if sender.SentEmails[0].To != "user1@example.com" {
		t.Errorf("SentEmails[0].To = %s、期待値 = user1@example.com", sender.SentEmails[0].To)
	}

	// 2通目を送信
	input2 := SendInput{
		To:      "user2@example.com",
		Subject: "Subject 2",
	}
	err = sender.Send(ctx, input2)
	if err != nil {
		t.Fatalf("Send()のエラー = %v", err)
	}

	if len(sender.SentEmails) != 2 {
		t.Fatalf("SentEmailsの件数 = %d、期待値 = 2", len(sender.SentEmails))
	}
	if sender.SentEmails[1].To != "user2@example.com" {
		t.Errorf("SentEmails[1].To = %s、期待値 = user2@example.com", sender.SentEmails[1].To)
	}
}

func TestNoopSender_Reset(t *testing.T) {
	t.Parallel()

	sender := NewNoopSender()
	ctx := context.Background()

	// メールを送信
	err := sender.Send(ctx, SendInput{To: "test@example.com", Subject: "Test"})
	if err != nil {
		t.Fatalf("Send()のエラー = %v", err)
	}
	if len(sender.SentEmails) != 1 {
		t.Fatalf("SentEmailsの件数 = %d、期待値 = 1", len(sender.SentEmails))
	}

	// リセット
	sender.Reset()

	if len(sender.SentEmails) != 0 {
		t.Errorf("Reset()後のSentEmailsの件数 = %d、期待値 = 0", len(sender.SentEmails))
	}
}
