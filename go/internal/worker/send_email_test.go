package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/riverqueue/river"

	"github.com/wikinoapp/wikino/go/internal/email"
)

// mockRawSender はテスト用のモック Sender
type mockRawSender struct {
	called   bool
	rawInput email.SendRawInput
	err      error
}

func (m *mockRawSender) Send(_ context.Context, _ email.SendInput) error {
	return nil
}

func (m *mockRawSender) SendRaw(_ context.Context, input email.SendRawInput) error {
	m.called = true
	m.rawInput = input
	return m.err
}

func TestSendEmailWorker_Work(t *testing.T) {
	t.Parallel()

	sender := &mockRawSender{}
	w := NewSendEmailWorker(sender)

	job := &river.Job[SendEmailArgs]{
		Args: SendEmailArgs{
			To:       "test@example.com",
			Subject:  "テスト件名",
			HTMLBody: "<p>HTMLボディ</p>",
			TextBody: "テキストボディ",
		},
	}

	err := w.Work(context.Background(), job)
	if err != nil {
		t.Fatalf("Work() error = %v", err)
	}

	if !sender.called {
		t.Error("SendRaw() が呼ばれていません")
	}
	if sender.rawInput.To != "test@example.com" {
		t.Errorf("To = %s, want test@example.com", sender.rawInput.To)
	}
	if sender.rawInput.Subject != "テスト件名" {
		t.Errorf("Subject = %s, want テスト件名", sender.rawInput.Subject)
	}
	if sender.rawInput.HTMLBody != "<p>HTMLボディ</p>" {
		t.Errorf("HTMLBody = %s, want <p>HTMLボディ</p>", sender.rawInput.HTMLBody)
	}
	if sender.rawInput.TextBody != "テキストボディ" {
		t.Errorf("TextBody = %s, want テキストボディ", sender.rawInput.TextBody)
	}
}

func TestSendEmailWorker_Work_EmptyEmail(t *testing.T) {
	t.Parallel()

	sender := &mockRawSender{}
	w := NewSendEmailWorker(sender)

	job := &river.Job[SendEmailArgs]{
		Args: SendEmailArgs{
			To:       "",
			Subject:  "テスト件名",
			HTMLBody: "<p>HTMLボディ</p>",
			TextBody: "テキストボディ",
		},
	}

	err := w.Work(context.Background(), job)
	if err == nil {
		t.Fatal("Work() error = nil, want error")
	}
	// メールアドレスが空の場合、SendRaw() は呼ばれない
	if sender.called {
		t.Error("SendRaw() が呼ばれるべきではありません")
	}
}

func TestSendEmailWorker_Work_SendError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("メール送信エラー")
	sender := &mockRawSender{err: expectedErr}
	w := NewSendEmailWorker(sender)

	job := &river.Job[SendEmailArgs]{
		Args: SendEmailArgs{
			To:       "test@example.com",
			Subject:  "テスト件名",
			HTMLBody: "<p>HTMLボディ</p>",
			TextBody: "テキストボディ",
		},
	}

	err := w.Work(context.Background(), job)
	if err == nil {
		t.Fatal("Work() error = nil, want error")
	}
	if !sender.called {
		t.Error("SendRaw() が呼ばれていません")
	}
}

func TestSendEmailArgs_Kind(t *testing.T) {
	t.Parallel()

	args := SendEmailArgs{}
	if args.Kind() != "send_email" {
		t.Errorf("Kind() = %s, want send_email", args.Kind())
	}
}

func TestSendEmailArgs_InsertOpts(t *testing.T) {
	t.Parallel()

	args := SendEmailArgs{}
	opts := args.InsertOpts()

	if opts.Queue != river.QueueDefault {
		t.Errorf("Queue = %s, want %s", opts.Queue, river.QueueDefault)
	}
	if opts.MaxAttempts != 5 {
		t.Errorf("MaxAttempts = %d, want 5", opts.MaxAttempts)
	}
}
