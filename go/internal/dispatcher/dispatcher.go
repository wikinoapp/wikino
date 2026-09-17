// Package dispatcherはジョブキューへの投入を抽象化する。
// Repositoryがデータベースアクセスを抽象化するのと同じ発想で、
// Dispatcherがジョブキューアクセスを抽象化する。
package dispatcher

import (
	"context"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// --- ジョブ引数型 ---

// SendEmailConfirmationArgsはメール確認コード送信ジョブの引数
type SendEmailConfirmationArgs struct {
	Email  string `json:"email"`
	Code   string `json:"code"`
	AppURL string `json:"app_url"`
	Locale string `json:"locale"`
}

// Kindはジョブの種類を返す
func (SendEmailConfirmationArgs) Kind() string { return "send_email_confirmation" }

// InsertOptsはジョブのInsertオプションを返す
func (SendEmailConfirmationArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: river.QueueDefault, MaxAttempts: 5}
}

// SendPasswordResetArgsはパスワードリセットメール送信ジョブの引数
type SendPasswordResetArgs struct {
	Email    string `json:"email"`
	ResetURL string `json:"reset_url"`
	AppURL   string `json:"app_url"`
	Locale   string `json:"locale"`
}

// Kindはジョブの種類を返す
func (SendPasswordResetArgs) Kind() string { return "send_password_reset" }

// InsertOptsはジョブのInsertオプションを返す
func (SendPasswordResetArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: river.QueueDefault, MaxAttempts: 5}
}

// CleanupRateLimitsArgsは古いRate Limitレコード削除ジョブの引数
type CleanupRateLimitsArgs struct {
	RetentionHours int `json:"retention_hours"`
}

// Kindはジョブの種類を返す
func (CleanupRateLimitsArgs) Kind() string { return "cleanup_rate_limits" }

// InsertOptsはジョブのInsertオプションを返す
func (CleanupRateLimitsArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: river.QueueDefault, MaxAttempts: 3}
}

// --- Dispatcher ---

// JobInserterはジョブをキューに追加するインターフェース
type JobInserter interface {
	Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

// Dispatcherはジョブキューへの投入を抽象化する
type Dispatcher struct {
	client JobInserter
}

// NewDispatcherは新しいDispatcherを生成する
func NewDispatcher(client JobInserter) *Dispatcher {
	return &Dispatcher{client: client}
}

// EnqueueEmailConfirmationはメール確認コード送信ジョブをキューに追加する
func (d *Dispatcher) EnqueueEmailConfirmation(ctx context.Context, email, code, appURL, locale string) error {
	args := SendEmailConfirmationArgs{Email: email, Code: code, AppURL: appURL, Locale: locale}
	opts := args.InsertOpts()
	_, err := d.client.Insert(ctx, args, &opts)
	return err
}

// EnqueuePasswordResetはパスワードリセットメール送信ジョブをキューに追加する
func (d *Dispatcher) EnqueuePasswordReset(ctx context.Context, email, resetURL, appURL, locale string) error {
	args := SendPasswordResetArgs{Email: email, ResetURL: resetURL, AppURL: appURL, Locale: locale}
	opts := args.InsertOpts()
	_, err := d.client.Insert(ctx, args, &opts)
	return err
}

// EnqueueCleanupRateLimitsは古いRate Limitレコード削除ジョブをキューに追加する
func (d *Dispatcher) EnqueueCleanupRateLimits(ctx context.Context, retentionHours int) error {
	args := CleanupRateLimitsArgs{RetentionHours: retentionHours}
	opts := args.InsertOpts()
	_, err := d.client.Insert(ctx, args, &opts)
	return err
}

// GenerateExportFilesArgsはスペースのエクスポートを書き出すジョブの引数。
//
// MaxAttemptsは他のジョブより小さくする。1回の試行がスペースの全ページを読み、すべての添付
// ファイルを取得するため、リトライのコストが高いためである。3回あればオブジェクトストレージの
// 一時的な失敗は乗り切れる。4回目は、成功する見込みの無い仕事をほぼ繰り返すことになる。
type GenerateExportFilesArgs struct {
	ExportID string `json:"export_id"`
	SpaceID  string `json:"space_id"`
}

// Kindはジョブの種類を返す。
func (GenerateExportFilesArgs) Kind() string { return "generate_export_files" }

// InsertOptsはジョブのInsertオプションを返す。
func (GenerateExportFilesArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: river.QueueDefault, MaxAttempts: 3}
}

// EnqueueGenerateExportFilesはエクスポートファイル生成ジョブをキューに追加する。
func (d *Dispatcher) EnqueueGenerateExportFiles(ctx context.Context, exportID, spaceID string) error {
	args := GenerateExportFilesArgs{ExportID: exportID, SpaceID: spaceID}
	opts := args.InsertOpts()
	_, err := d.client.Insert(ctx, args, &opts)
	return err
}
