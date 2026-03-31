package dispatcher

import (
	"context"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// mockJobInserter はテスト用のモック
type mockJobInserter struct {
	called bool
	args   river.JobArgs
	opts   *river.InsertOpts
}

func (m *mockJobInserter) Insert(_ context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	m.called = true
	m.args = args
	m.opts = opts
	return &rivertype.JobInsertResult{}, nil
}

func TestEnqueueEmailConfirmation(t *testing.T) {
	t.Parallel()

	mock := &mockJobInserter{}
	d := NewDispatcher(mock)

	err := d.EnqueueEmailConfirmation(context.Background(), "test@example.com", "ABC123", "https://example.com", "ja")
	if err != nil {
		t.Fatalf("EnqueueEmailConfirmation() error = %v", err)
	}

	if !mock.called {
		t.Fatal("Insert が呼ばれていません")
	}

	args, ok := mock.args.(SendEmailConfirmationArgs)
	if !ok {
		t.Fatalf("args の型が SendEmailConfirmationArgs ではありません: %T", mock.args)
	}
	if args.Email != "test@example.com" {
		t.Errorf("Email = %s, want test@example.com", args.Email)
	}
	if args.Code != "ABC123" {
		t.Errorf("Code = %s, want ABC123", args.Code)
	}
	if args.AppURL != "https://example.com" {
		t.Errorf("AppURL = %s, want https://example.com", args.AppURL)
	}
	if args.Locale != "ja" {
		t.Errorf("Locale = %s, want ja", args.Locale)
	}
	if mock.opts == nil {
		t.Fatal("InsertOpts が nil です")
	}
	if mock.opts.MaxAttempts != 5 {
		t.Errorf("MaxAttempts = %d, want 5", mock.opts.MaxAttempts)
	}
}

func TestEnqueuePasswordReset(t *testing.T) {
	t.Parallel()

	mock := &mockJobInserter{}
	d := NewDispatcher(mock)

	err := d.EnqueuePasswordReset(context.Background(), "test@example.com", "https://example.com/reset?token=abc", "https://example.com", "en")
	if err != nil {
		t.Fatalf("EnqueuePasswordReset() error = %v", err)
	}

	if !mock.called {
		t.Fatal("Insert が呼ばれていません")
	}

	args, ok := mock.args.(SendPasswordResetArgs)
	if !ok {
		t.Fatalf("args の型が SendPasswordResetArgs ではありません: %T", mock.args)
	}
	if args.Email != "test@example.com" {
		t.Errorf("Email = %s, want test@example.com", args.Email)
	}
	if args.ResetURL != "https://example.com/reset?token=abc" {
		t.Errorf("ResetURL = %s, want https://example.com/reset?token=abc", args.ResetURL)
	}
	if args.AppURL != "https://example.com" {
		t.Errorf("AppURL = %s, want https://example.com", args.AppURL)
	}
	if args.Locale != "en" {
		t.Errorf("Locale = %s, want en", args.Locale)
	}
	if mock.opts == nil {
		t.Fatal("InsertOpts が nil です")
	}
	if mock.opts.MaxAttempts != 5 {
		t.Errorf("MaxAttempts = %d, want 5", mock.opts.MaxAttempts)
	}
}

func TestEnqueueCleanupRateLimits(t *testing.T) {
	t.Parallel()

	mock := &mockJobInserter{}
	d := NewDispatcher(mock)

	err := d.EnqueueCleanupRateLimits(context.Background(), 48)
	if err != nil {
		t.Fatalf("EnqueueCleanupRateLimits() error = %v", err)
	}

	if !mock.called {
		t.Fatal("Insert が呼ばれていません")
	}

	args, ok := mock.args.(CleanupRateLimitsArgs)
	if !ok {
		t.Fatalf("args の型が CleanupRateLimitsArgs ではありません: %T", mock.args)
	}
	if args.RetentionHours != 48 {
		t.Errorf("RetentionHours = %d, want 48", args.RetentionHours)
	}
	if mock.opts == nil {
		t.Fatal("InsertOpts が nil です")
	}
	if mock.opts.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d, want 3", mock.opts.MaxAttempts)
	}
}

func TestSendEmailConfirmationArgs_Kind(t *testing.T) {
	t.Parallel()
	if got := (SendEmailConfirmationArgs{}).Kind(); got != "send_email_confirmation" {
		t.Errorf("Kind() = %s, want send_email_confirmation", got)
	}
}

func TestSendPasswordResetArgs_Kind(t *testing.T) {
	t.Parallel()
	if got := (SendPasswordResetArgs{}).Kind(); got != "send_password_reset" {
		t.Errorf("Kind() = %s, want send_password_reset", got)
	}
}

func TestCleanupRateLimitsArgs_Kind(t *testing.T) {
	t.Parallel()
	if got := (CleanupRateLimitsArgs{}).Kind(); got != "cleanup_rate_limits" {
		t.Errorf("Kind() = %s, want cleanup_rate_limits", got)
	}
}
