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
