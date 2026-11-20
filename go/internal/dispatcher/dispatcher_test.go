package dispatcher

import (
	"context"
	"database/sql"
	"slices"
	"testing"
	"time"

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
	d := NewDispatcher(mock, nil)

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
	d := NewDispatcher(mock, nil)

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
	d := NewDispatcher(mock, nil)

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

func TestEnqueueDeleteTopic(t *testing.T) {
	t.Parallel()

	mock := &mockJobInserter{}
	d := NewDispatcher(mock, nil)

	if err := d.EnqueueDeleteTopic(context.Background(), "space-id", "topic-id"); err != nil {
		t.Fatalf("EnqueueDeleteTopic()のエラー = %v", err)
	}

	args, ok := mock.args.(DeleteTopicArgs)
	if !ok {
		t.Fatalf("argsの型がDeleteTopicArgsではありません: %T", mock.args)
	}
	if args.SpaceID != "space-id" {
		t.Errorf("SpaceID = %s、期待値 = space-id", args.SpaceID)
	}
	if args.TopicID != "topic-id" {
		t.Errorf("TopicID = %s、期待値 = topic-id", args.TopicID)
	}
	if args.Kind() != "delete_topic" {
		t.Errorf("Kind() = %s、期待値 = delete_topic", args.Kind())
	}
	if mock.opts == nil {
		t.Fatal("InsertOptsがnilです")
	}
	if mock.opts.MaxAttempts != 5 {
		t.Errorf("MaxAttempts = %d、期待値 = 5", mock.opts.MaxAttempts)
	}
}

func TestEnqueueDeleteSpace(t *testing.T) {
	t.Parallel()

	mock := &mockJobInserter{}
	d := NewDispatcher(mock, nil)

	if err := d.EnqueueDeleteSpace(context.Background(), "space-id"); err != nil {
		t.Fatalf("EnqueueDeleteSpace()のエラー = %v", err)
	}

	args, ok := mock.args.(DeleteSpaceArgs)
	if !ok {
		t.Fatalf("argsの型がDeleteSpaceArgsではありません: %T", mock.args)
	}
	if args.SpaceID != "space-id" {
		t.Errorf("SpaceID = %s、期待値 = space-id", args.SpaceID)
	}
	if args.Kind() != "delete_space" {
		t.Errorf("Kind() = %s、期待値 = delete_space", args.Kind())
	}
	if mock.opts == nil {
		t.Fatal("InsertOptsがnilです")
	}
	if mock.opts.MaxAttempts != 5 {
		t.Errorf("MaxAttempts = %d、期待値 = 5", mock.opts.MaxAttempts)
	}
}

func TestDeleteTrashedPagesArgs(t *testing.T) {
	t.Parallel()

	args := DeleteTrashedPagesArgs{}
	if got := args.Kind(); got != "delete_trashed_pages" {
		t.Errorf("Kind() = %s、期待値 = delete_trashed_pages", got)
	}
	if got := args.InsertOpts().MaxAttempts; got != 3 {
		t.Errorf("MaxAttempts = %d、期待値 = 3", got)
	}
}

func TestEnqueueProcessAttachment(t *testing.T) {
	t.Parallel()

	mock := &mockJobInserter{}
	d := NewDispatcher(mock, nil)

	if err := d.EnqueueProcessAttachment(context.Background(), "attachment-id", "space-id"); err != nil {
		t.Fatalf("EnqueueProcessAttachment()のエラー = %v", err)
	}

	args, ok := mock.args.(ProcessAttachmentArgs)
	if !ok {
		t.Fatalf("argsの型がProcessAttachmentArgsではありません: %T", mock.args)
	}
	if args.AttachmentID != "attachment-id" {
		t.Errorf("AttachmentID = %s、期待値 = attachment-id", args.AttachmentID)
	}
	if args.SpaceID != "space-id" {
		t.Errorf("SpaceID = %s、期待値 = space-id", args.SpaceID)
	}
	if args.Kind() != "process_attachment" {
		t.Errorf("Kind() = %s、期待値 = process_attachment", args.Kind())
	}
	if mock.opts == nil {
		t.Fatal("InsertOptsがnilです")
	}
	if mock.opts.MaxAttempts != 10 {
		t.Errorf("MaxAttempts = %d、期待値 = 10", mock.opts.MaxAttempts)
	}
}

// mockTxJobClientは、トランザクション内の投入の引数を記録するテスト用のモック
type mockTxJobClient struct {
	TxJobClient
	args river.JobArgs
	opts *river.InsertOpts
}

func (m *mockTxJobClient) InsertTx(_ context.Context, _ *sql.Tx, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	m.args = args
	m.opts = opts
	return &rivertype.JobInsertResult{}, nil
}

func TestEnqueueCleanupAttachmentObjectsTx(t *testing.T) {
	t.Parallel()

	mock := &mockTxJobClient{}
	d := NewDispatcher(nil, mock)

	before := time.Now()
	if err := d.EnqueueCleanupAttachmentObjectsTx(context.Background(), nil, "space-id", []string{"attachment-id"}, []string{"key-1", "key-2"}); err != nil {
		t.Fatalf("EnqueueCleanupAttachmentObjectsTx()のエラー = %v", err)
	}
	after := time.Now()

	args, ok := mock.args.(CleanupAttachmentObjectsArgs)
	if !ok {
		t.Fatalf("argsの型がCleanupAttachmentObjectsArgsではありません: %T", mock.args)
	}
	if args.SpaceID != "space-id" {
		t.Errorf("SpaceID = %s、期待値 = space-id", args.SpaceID)
	}
	if !slices.Equal(args.AttachmentIDs, []string{"attachment-id"}) {
		t.Errorf("AttachmentIDs = %v、期待値 = [attachment-id]", args.AttachmentIDs)
	}
	if !slices.Equal(args.Keys, []string{"key-1", "key-2"}) {
		t.Errorf("Keys = %v、期待値 = [key-1 key-2]", args.Keys)
	}
	if args.Kind() != "cleanup_attachment_objects" {
		t.Errorf("Kind() = %s、期待値 = cleanup_attachment_objects", args.Kind())
	}
	if mock.opts == nil {
		t.Fatal("InsertOptsがnilです")
	}
	if mock.opts.Queue != river.QueueDefault {
		t.Errorf("Queue = %s、期待値 = %s", mock.opts.Queue, river.QueueDefault)
	}
	if mock.opts.MaxAttempts != 10 {
		t.Errorf("MaxAttempts = %d、期待値 = 10", mock.opts.MaxAttempts)
	}
	// Rails版がvariantの実体を書き終えるまで、最初の実行を遅らせる
	earliest, latest := before.Add(cleanupAttachmentObjectsDelay), after.Add(cleanupAttachmentObjectsDelay)
	if mock.opts.ScheduledAt.Before(earliest) || mock.opts.ScheduledAt.After(latest) {
		t.Errorf("ScheduledAt = %v、期待値 = %vから%vの間", mock.opts.ScheduledAt, earliest, latest)
	}
}

func TestEnqueueCleanupExportObjectsTx(t *testing.T) {
	t.Parallel()

	mock := &mockTxJobClient{}
	d := NewDispatcher(nil, mock)

	if err := d.EnqueueCleanupExportObjectsTx(context.Background(), nil, "space-id", []string{"export-id"}, []string{"exports/space-id/export-id.zip"}); err != nil {
		t.Fatalf("EnqueueCleanupExportObjectsTx()のエラー = %v", err)
	}

	args, ok := mock.args.(CleanupExportObjectsArgs)
	if !ok {
		t.Fatalf("argsの型がCleanupExportObjectsArgsではありません: %T", mock.args)
	}
	if args.SpaceID != "space-id" {
		t.Errorf("SpaceID = %s、期待値 = space-id", args.SpaceID)
	}
	if !slices.Equal(args.ExportIDs, []string{"export-id"}) {
		t.Errorf("ExportIDs = %v、期待値 = [export-id]", args.ExportIDs)
	}
	if !slices.Equal(args.Keys, []string{"exports/space-id/export-id.zip"}) {
		t.Errorf("Keys = %v、期待値 = [exports/space-id/export-id.zip]", args.Keys)
	}
	if len(args.LegacyBlobs) != 0 {
		t.Errorf("LegacyBlobs = %v、期待値 = 空", args.LegacyBlobs)
	}
	assertCleanupExportObjectsOpts(t, args, mock.opts)
}

func TestEnqueueCleanupLegacyExportBlobsTx(t *testing.T) {
	t.Parallel()

	t.Run("blobのIDとキーを同じ順に組にして登録する", func(t *testing.T) {
		t.Parallel()

		mock := &mockTxJobClient{}
		d := NewDispatcher(nil, mock)

		if err := d.EnqueueCleanupLegacyExportBlobsTx(context.Background(), nil, "space-id", []string{"blob-1", "blob-2"}, []string{"key-1", "key-2"}); err != nil {
			t.Fatalf("EnqueueCleanupLegacyExportBlobsTx()のエラー = %v", err)
		}

		args, ok := mock.args.(CleanupExportObjectsArgs)
		if !ok {
			t.Fatalf("argsの型がCleanupExportObjectsArgsではありません: %T", mock.args)
		}
		if args.SpaceID != "space-id" {
			t.Errorf("SpaceID = %s、期待値 = space-id", args.SpaceID)
		}
		want := []LegacyExportBlobArgs{{ID: "blob-1", Key: "key-1"}, {ID: "blob-2", Key: "key-2"}}
		if !slices.Equal(args.LegacyBlobs, want) {
			t.Errorf("LegacyBlobs = %v、期待値 = %v", args.LegacyBlobs, want)
		}
		// Rails期のZIPは生成ジョブを持たないため、生成の終了を待つエクスポートもキーも持たない
		if len(args.ExportIDs) != 0 || len(args.Keys) != 0 {
			t.Errorf("ExportIDs = %v、Keys = %v、期待値 = どちらも空", args.ExportIDs, args.Keys)
		}
		assertCleanupExportObjectsOpts(t, args, mock.opts)
	})

	t.Run("IDとキーの数が違えば登録しない", func(t *testing.T) {
		t.Parallel()

		mock := &mockTxJobClient{}
		d := NewDispatcher(nil, mock)

		if err := d.EnqueueCleanupLegacyExportBlobsTx(context.Background(), nil, "space-id", []string{"blob-1", "blob-2"}, []string{"key-1"}); err == nil {
			t.Fatal("エラーを期待したが、nilだった")
		}
		if mock.args != nil {
			t.Errorf("登録した引数 = %v、期待値 = 登録しない", mock.args)
		}
	})
}

// assertCleanupExportObjectsOptsは、ZIPの後始末のジョブの種類と投入のオプションを確かめる
func assertCleanupExportObjectsOpts(t *testing.T, args CleanupExportObjectsArgs, opts *river.InsertOpts) {
	t.Helper()
	if args.Kind() != "cleanup_export_objects" {
		t.Errorf("Kind() = %s、期待値 = cleanup_export_objects", args.Kind())
	}
	if opts == nil {
		t.Fatal("InsertOptsがnilです")
	}
	if opts.Queue != river.QueueDefault {
		t.Errorf("Queue = %s、期待値 = %s", opts.Queue, river.QueueDefault)
	}
	if opts.MaxAttempts != 10 {
		t.Errorf("MaxAttempts = %d、期待値 = 10", opts.MaxAttempts)
	}
}
