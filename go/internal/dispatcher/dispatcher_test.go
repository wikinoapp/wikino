package dispatcher

import (
	"context"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// mockJobInserterはテスト用のモック
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
		t.Fatalf("EnqueueEmailConfirmation()のエラー = %v", err)
	}

	if !mock.called {
		t.Fatal("Insertが呼ばれていません")
	}

	args, ok := mock.args.(SendEmailConfirmationArgs)
	if !ok {
		t.Fatalf("argsの型がSendEmailConfirmationArgsではありません: %T", mock.args)
	}
	if args.Email != "test@example.com" {
		t.Errorf("Email = %s、期待値 = test@example.com", args.Email)
	}
	if args.Code != "ABC123" {
		t.Errorf("Code = %s、期待値 = ABC123", args.Code)
	}
	if args.AppURL != "https://example.com" {
		t.Errorf("AppURL = %s、期待値 = https://example.com", args.AppURL)
	}
	if args.Locale != "ja" {
		t.Errorf("Locale = %s、期待値 = ja", args.Locale)
	}
	if mock.opts == nil {
		t.Fatal("InsertOptsがnilです")
	}
	if mock.opts.MaxAttempts != 5 {
		t.Errorf("MaxAttempts = %d、期待値 = 5", mock.opts.MaxAttempts)
	}
}

func TestEnqueuePasswordReset(t *testing.T) {
	t.Parallel()

	mock := &mockJobInserter{}
	d := NewDispatcher(mock)

	err := d.EnqueuePasswordReset(context.Background(), "test@example.com", "https://example.com/reset?token=abc", "https://example.com", "en")
	if err != nil {
		t.Fatalf("EnqueuePasswordReset()のエラー = %v", err)
	}

	if !mock.called {
		t.Fatal("Insertが呼ばれていません")
	}

	args, ok := mock.args.(SendPasswordResetArgs)
	if !ok {
		t.Fatalf("argsの型がSendPasswordResetArgsではありません: %T", mock.args)
	}
	if args.Email != "test@example.com" {
		t.Errorf("Email = %s、期待値 = test@example.com", args.Email)
	}
	if args.ResetURL != "https://example.com/reset?token=abc" {
		t.Errorf("ResetURL = %s、期待値 = https://example.com/reset?token=abc", args.ResetURL)
	}
	if args.AppURL != "https://example.com" {
		t.Errorf("AppURL = %s、期待値 = https://example.com", args.AppURL)
	}
	if args.Locale != "en" {
		t.Errorf("Locale = %s、期待値 = en", args.Locale)
	}
	if mock.opts == nil {
		t.Fatal("InsertOptsがnilです")
	}
	if mock.opts.MaxAttempts != 5 {
		t.Errorf("MaxAttempts = %d、期待値 = 5", mock.opts.MaxAttempts)
	}
}

func TestEnqueueCleanupRateLimits(t *testing.T) {
	t.Parallel()

	mock := &mockJobInserter{}
	d := NewDispatcher(mock)

	err := d.EnqueueCleanupRateLimits(context.Background(), 48)
	if err != nil {
		t.Fatalf("EnqueueCleanupRateLimits()のエラー = %v", err)
	}

	if !mock.called {
		t.Fatal("Insertが呼ばれていません")
	}

	args, ok := mock.args.(CleanupRateLimitsArgs)
	if !ok {
		t.Fatalf("argsの型がCleanupRateLimitsArgsではありません: %T", mock.args)
	}
	if args.RetentionHours != 48 {
		t.Errorf("RetentionHours = %d、期待値 = 48", args.RetentionHours)
	}
	if mock.opts == nil {
		t.Fatal("InsertOptsがnilです")
	}
	if mock.opts.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d、期待値 = 3", mock.opts.MaxAttempts)
	}
}

func TestSendEmailConfirmationArgs_Kind(t *testing.T) {
	t.Parallel()
	if got := (SendEmailConfirmationArgs{}).Kind(); got != "send_email_confirmation" {
		t.Errorf("Kind() = %s、期待値 = send_email_confirmation", got)
	}
}

func TestSendPasswordResetArgs_Kind(t *testing.T) {
	t.Parallel()
	if got := (SendPasswordResetArgs{}).Kind(); got != "send_password_reset" {
		t.Errorf("Kind() = %s、期待値 = send_password_reset", got)
	}
}

func TestCleanupRateLimitsArgs_Kind(t *testing.T) {
	t.Parallel()
	if got := (CleanupRateLimitsArgs{}).Kind(); got != "cleanup_rate_limits" {
		t.Errorf("Kind() = %s、期待値 = cleanup_rate_limits", got)
	}
}
