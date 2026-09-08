// Package dispatcher はジョブキューへの投入を抽象化する。
// Repository がデータベースアクセスを抽象化するのと同じ発想で、
// Dispatcher がジョブキューアクセスを抽象化する。
package dispatcher

import (
	"context"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// --- ジョブ引数型 ---

// SendEmailConfirmationArgs はメール確認コード送信ジョブの引数
type SendEmailConfirmationArgs struct {
	Email  string `json:"email"`
	Code   string `json:"code"`
	AppURL string `json:"app_url"`
	Locale string `json:"locale"`
}

// Kind はジョブの種類を返す
func (SendEmailConfirmationArgs) Kind() string { return "send_email_confirmation" }

// InsertOpts はジョブの Insert オプションを返す
func (SendEmailConfirmationArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: river.QueueDefault, MaxAttempts: 5}
}

// SendPasswordResetArgs はパスワードリセットメール送信ジョブの引数
type SendPasswordResetArgs struct {
	Email    string `json:"email"`
	ResetURL string `json:"reset_url"`
	AppURL   string `json:"app_url"`
	Locale   string `json:"locale"`
}

// Kind はジョブの種類を返す
func (SendPasswordResetArgs) Kind() string { return "send_password_reset" }

// InsertOpts はジョブの Insert オプションを返す
func (SendPasswordResetArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: river.QueueDefault, MaxAttempts: 5}
}

// CleanupRateLimitsArgs は古い Rate Limit レコード削除ジョブの引数
type CleanupRateLimitsArgs struct {
	RetentionHours int `json:"retention_hours"`
}

// Kind はジョブの種類を返す
func (CleanupRateLimitsArgs) Kind() string { return "cleanup_rate_limits" }

// InsertOpts はジョブの Insert オプションを返す
func (CleanupRateLimitsArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: river.QueueDefault, MaxAttempts: 3}
}

// --- Dispatcher ---

// JobInserter はジョブをキューに追加するインターフェース
type JobInserter interface {
	Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

// Dispatcher はジョブキューへの投入を抽象化する
type Dispatcher struct {
	client JobInserter
}

// NewDispatcher は新しい Dispatcher を生成する
func NewDispatcher(client JobInserter) *Dispatcher {
	return &Dispatcher{client: client}
}

// EnqueueEmailConfirmation はメール確認コード送信ジョブをキューに追加する
func (d *Dispatcher) EnqueueEmailConfirmation(ctx context.Context, email, code, appURL, locale string) error {
	args := SendEmailConfirmationArgs{Email: email, Code: code, AppURL: appURL, Locale: locale}
	opts := args.InsertOpts()
	_, err := d.client.Insert(ctx, args, &opts)
	return err
}

// EnqueuePasswordReset はパスワードリセットメール送信ジョブをキューに追加する
func (d *Dispatcher) EnqueuePasswordReset(ctx context.Context, email, resetURL, appURL, locale string) error {
	args := SendPasswordResetArgs{Email: email, ResetURL: resetURL, AppURL: appURL, Locale: locale}
	opts := args.InsertOpts()
	_, err := d.client.Insert(ctx, args, &opts)
	return err
}

// EnqueueCleanupRateLimits は古い Rate Limit レコード削除ジョブをキューに追加する
func (d *Dispatcher) EnqueueCleanupRateLimits(ctx context.Context, retentionHours int) error {
	args := CleanupRateLimitsArgs{RetentionHours: retentionHours}
	opts := args.InsertOpts()
	_, err := d.client.Insert(ctx, args, &opts)
	return err
}

// GenerateExportFilesArgs are the arguments of the job that writes a space export.
//
// MaxAttempts is lower than the other jobs': one attempt reads every page of a space and fetches
// every attachment, so a retry is expensive. Three attempts still ride out a transient failure of
// the object storage, and the fourth would mostly repeat work that is not going to succeed.
//
// [Ja] GenerateExportFilesArgs はスペースのエクスポートを書き出すジョブの引数。
//
// MaxAttempts は他のジョブより小さくする。1 回の試行がスペースの全ページを読み、すべての添付
// ファイルを取得するため、リトライのコストが高いためである。3 回あればオブジェクトストレージの
// 一時的な失敗は乗り切れる。4 回目は、成功する見込みの無い仕事をほぼ繰り返すことになる。
type GenerateExportFilesArgs struct {
	ExportID string `json:"export_id"`
	SpaceID  string `json:"space_id"`
}

// Kind returns the kind of the job.
//
// [Ja] Kind はジョブの種類を返す。
func (GenerateExportFilesArgs) Kind() string { return "generate_export_files" }

// InsertOpts returns the Insert options of the job.
//
// [Ja] InsertOpts はジョブの Insert オプションを返す。
func (GenerateExportFilesArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: river.QueueDefault, MaxAttempts: 3}
}

// EnqueueGenerateExportFiles adds the job that generates the files of an export to the queue.
//
// [Ja] EnqueueGenerateExportFiles はエクスポートファイル生成ジョブをキューに追加する。
func (d *Dispatcher) EnqueueGenerateExportFiles(ctx context.Context, exportID, spaceID string) error {
	args := GenerateExportFilesArgs{ExportID: exportID, SpaceID: spaceID}
	opts := args.InsertOpts()
	_, err := d.client.Insert(ctx, args, &opts)
	return err
}
