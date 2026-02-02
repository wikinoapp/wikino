package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/riverqueue/river"

	"github.com/wikinoapp/wikino/go/internal/email"
)

// mockSender はテスト用のモック Sender
type mockSender struct {
	called bool
	input  email.SendInput
	err    error
}

func (m *mockSender) Send(_ context.Context, input email.SendInput) error {
	m.called = true
	m.input = input
	return m.err
}

func TestSendEmailConfirmationWorker_Work_Japanese(t *testing.T) {
	t.Parallel()

	sender := &mockSender{}
	worker := NewSendEmailConfirmationWorker(sender)

	job := &river.Job[SendEmailConfirmationArgs]{
		Args: SendEmailConfirmationArgs{
			Email:   "test@example.com",
			Code:    "ABC123",
			AppURL:  "https://wikino.app",
			Subject: "確認用コード",
			Locale:  "ja",
		},
	}

	err := worker.Work(context.Background(), job)
	if err != nil {
		t.Fatalf("Work() error = %v", err)
	}

	if !sender.called {
		t.Error("Send() が呼ばれていません")
	}
	if sender.input.To != "test@example.com" {
		t.Errorf("To = %s, want test@example.com", sender.input.To)
	}
	if sender.input.Subject != "確認用コード" {
		t.Errorf("Subject = %s, want 確認用コード", sender.input.Subject)
	}
	if sender.input.HTMLBody == nil {
		t.Error("HTMLBody is nil")
	}
	if sender.input.TextBody == nil {
		t.Error("TextBody is nil")
	}
}

func TestSendEmailConfirmationWorker_Work_English(t *testing.T) {
	t.Parallel()

	sender := &mockSender{}
	worker := NewSendEmailConfirmationWorker(sender)

	job := &river.Job[SendEmailConfirmationArgs]{
		Args: SendEmailConfirmationArgs{
			Email:   "test@example.com",
			Code:    "ABC123",
			AppURL:  "https://wikino.app",
			Subject: "Confirmation Code",
			Locale:  "en",
		},
	}

	err := worker.Work(context.Background(), job)
	if err != nil {
		t.Fatalf("Work() error = %v", err)
	}

	if !sender.called {
		t.Error("Send() が呼ばれていません")
	}
	if sender.input.To != "test@example.com" {
		t.Errorf("To = %s, want test@example.com", sender.input.To)
	}
	if sender.input.HTMLBody == nil {
		t.Error("HTMLBody is nil")
	}
	if sender.input.TextBody == nil {
		t.Error("TextBody is nil")
	}
}

func TestSendEmailConfirmationWorker_Work_EmptyEmail(t *testing.T) {
	t.Parallel()

	sender := &mockSender{}
	worker := NewSendEmailConfirmationWorker(sender)

	job := &river.Job[SendEmailConfirmationArgs]{
		Args: SendEmailConfirmationArgs{
			Email:   "",
			Code:    "ABC123",
			AppURL:  "https://wikino.app",
			Subject: "確認用コード",
			Locale:  "ja",
		},
	}

	err := worker.Work(context.Background(), job)
	if err == nil {
		t.Fatal("Work() error = nil, want error")
	}
	// メールアドレスが空の場合、Send() は呼ばれない
	if sender.called {
		t.Error("Send() が呼ばれるべきではありません")
	}
}

func TestSendEmailConfirmationWorker_Work_SendError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("メール送信エラー")
	sender := &mockSender{err: expectedErr}
	worker := NewSendEmailConfirmationWorker(sender)

	job := &river.Job[SendEmailConfirmationArgs]{
		Args: SendEmailConfirmationArgs{
			Email:   "test@example.com",
			Code:    "ABC123",
			AppURL:  "https://wikino.app",
			Subject: "確認用コード",
			Locale:  "ja",
		},
	}

	err := worker.Work(context.Background(), job)
	if err == nil {
		t.Fatal("Work() error = nil, want error")
	}
	if !sender.called {
		t.Error("Send() が呼ばれていません")
	}
}

func TestSendEmailConfirmationArgs_Kind(t *testing.T) {
	t.Parallel()

	args := SendEmailConfirmationArgs{}
	if args.Kind() != "send_email_confirmation" {
		t.Errorf("Kind() = %s, want send_email_confirmation", args.Kind())
	}
}
