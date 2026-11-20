// Package dispatcherはジョブキューへの投入を抽象化する。
// Repositoryがデータベースアクセスを抽象化するのと同じ発想で、
// Dispatcherがジョブキューアクセスを抽象化する。
package dispatcher

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

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

// CleanupEmailConfirmationsArgsは古いメール確認情報を削除するジョブの引数
type CleanupEmailConfirmationsArgs struct {
	RetentionHours int `json:"retention_hours"`
}

// Kindはジョブの種類を返す
func (CleanupEmailConfirmationsArgs) Kind() string { return "cleanup_email_confirmations" }

// InsertOptsはジョブのInsertオプションを返す
func (CleanupEmailConfirmationsArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: river.QueueDefault, MaxAttempts: 3}
}

// DeleteTrashedPagesArgsは、ゴミ箱に入れて保持期間を過ぎたページを物理削除する定期ジョブの引数。
// 対象は実行した時点で決まり、何度試しても結果が変わらないため、再試行してよい。
type DeleteTrashedPagesArgs struct{}

// Kindはジョブの種類を返す。
func (DeleteTrashedPagesArgs) Kind() string { return "delete_trashed_pages" }

// InsertOptsはジョブのInsertオプションを返す。
func (DeleteTrashedPagesArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: river.QueueDefault, MaxAttempts: 3}
}

// --- Dispatcher ---

// JobInserterはジョブをキューに追加するインターフェース
type JobInserter interface {
	Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

// TxJobClientは、database/sqlのトランザクション内の投入と、処理中のジョブの確認を担当する。
// 実行用のpgxクライアントとは別に、同じDBのriverdatabasesqlクライアントを使う。
type TxJobClient interface {
	InsertTx(ctx context.Context, tx *sql.Tx, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
	JobList(ctx context.Context, params *river.JobListParams) (*river.JobListResult, error)
}

// Dispatcherはジョブキューへの投入を抽象化する
type Dispatcher struct {
	client   JobInserter
	txClient TxJobClient
}

// NewDispatcherは新しいDispatcherを生成する
func NewDispatcher(client JobInserter, txClient TxJobClient) *Dispatcher {
	return &Dispatcher{client: client, txClient: txClient}
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

// DeleteTopicArgsは論理削除したトピックを、関連レコードとともに物理削除するジョブの引数。
// 削除は同じトピックに何度試しても結果が変わらないため、再試行してよい。
type DeleteTopicArgs struct {
	SpaceID string `json:"space_id"`
	TopicID string `json:"topic_id"`
}

// Kindはジョブの種類を返す。
func (DeleteTopicArgs) Kind() string { return "delete_topic" }

// InsertOptsはジョブのInsertオプションを返す。
func (DeleteTopicArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: river.QueueDefault, MaxAttempts: 5}
}

// EnqueueDeleteTopicはトピックの物理削除のジョブをキューに追加する。
func (d *Dispatcher) EnqueueDeleteTopic(ctx context.Context, spaceID, topicID string) error {
	args := DeleteTopicArgs{SpaceID: spaceID, TopicID: topicID}
	opts := args.InsertOpts()
	_, err := d.client.Insert(ctx, args, &opts)
	return err
}

// DeleteSpaceArgsは論理削除したスペースを、トピック・ページ・添付ファイル・エクスポート・メンバーとともに
// 物理削除するジョブの引数。削除は同じスペースに何度試しても結果が変わらないため、再試行してよい。
type DeleteSpaceArgs struct {
	SpaceID string `json:"space_id"`
}

// Kindはジョブの種類を返す。
func (DeleteSpaceArgs) Kind() string { return "delete_space" }

// InsertOptsはジョブのInsertオプションを返す。
func (DeleteSpaceArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: river.QueueDefault, MaxAttempts: 5}
}

// EnqueueDeleteSpaceはスペースの物理削除のジョブをキューに追加する。
func (d *Dispatcher) EnqueueDeleteSpace(ctx context.Context, spaceID string) error {
	args := DeleteSpaceArgs{SpaceID: spaceID}
	opts := args.InsertOpts()
	_, err := d.client.Insert(ctx, args, &opts)
	return err
}

// DeleteAccountArgsは論理削除したユーザーを、関連レコードとともに物理削除するジョブの引数。
// 削除は同じユーザーに何度試しても結果が変わらないため、再試行してよい。
type DeleteAccountArgs struct {
	UserID string `json:"user_id"`
}

// Kindはジョブの種類を返す。
func (DeleteAccountArgs) Kind() string { return "delete_account" }

// InsertOptsはジョブのInsertオプションを返す。
func (DeleteAccountArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: river.QueueDefault, MaxAttempts: 5}
}

// EnqueueDeleteAccountはアカウントの物理削除のジョブをキューに追加する。
func (d *Dispatcher) EnqueueDeleteAccount(ctx context.Context, userID string) error {
	args := DeleteAccountArgs{UserID: userID}
	opts := args.InsertOpts()
	_, err := d.client.Insert(ctx, args, &opts)
	return err
}

// ProcessAttachmentArgsは、事前署名した添付ファイルのアップロードを待ち、メタデータの除去やSVGの
// サニタイズを行うジョブの引数。アップロードを待つ間は先送り (snooze) し、試行の回数を使わない。
// 試行の回数は、オブジェクトストレージの一時的な失敗の再試行にだけ使う。
type ProcessAttachmentArgs struct {
	AttachmentID string `json:"attachment_id"`
	SpaceID      string `json:"space_id"`
}

// Kindはジョブの種類を返す。
func (ProcessAttachmentArgs) Kind() string { return "process_attachment" }

// InsertOptsはジョブのInsertオプションを返す。
func (ProcessAttachmentArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: river.QueueDefault, MaxAttempts: 10}
}

// EnqueueProcessAttachmentは、添付ファイルのアップロード後の処理のジョブをキューに追加する。
func (d *Dispatcher) EnqueueProcessAttachment(ctx context.Context, attachmentID, spaceID string) error {
	args := ProcessAttachmentArgs{AttachmentID: attachmentID, SpaceID: spaceID}
	opts := args.InsertOpts()
	_, err := d.client.Insert(ctx, args, &opts)
	return err
}

// CleanupAttachmentObjectsArgsは、行を削除した添付ファイルの実体を回収するジョブの引数。
// キーを行の削除と同時に保存し、行が無い状態からも再試行できるようにする。
type CleanupAttachmentObjectsArgs struct {
	SpaceID       string   `json:"space_id"`
	AttachmentIDs []string `json:"attachment_ids"`
	Keys          []string `json:"keys"`
}

// Kindはジョブの種類を返す。
func (CleanupAttachmentObjectsArgs) Kind() string { return "cleanup_attachment_objects" }

// InsertOptsはジョブのInsertオプションを返す。
func (CleanupAttachmentObjectsArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: river.QueueDefault, MaxAttempts: 10}
}

// cleanupAttachmentObjectsDelayは、後始末ジョブを最初に実行するまでの時間。Rails版はサムネイル (variant) の
// 行をコミットしてから実体をアップロードするため、行の削除がvariantのキーを受け取った時点では、実体がまだ
// 書き込まれていないことがある。そのリクエストが実体を書き終えるのを待ってから消す
const cleanupAttachmentObjectsDelay = time.Minute

// EnqueueCleanupAttachmentObjectsTxは、行の削除と同じトランザクションで後始末を登録する。
// 最初の実行はcleanupAttachmentObjectsDelayだけ遅らせる。
func (d *Dispatcher) EnqueueCleanupAttachmentObjectsTx(ctx context.Context, tx *sql.Tx, spaceID string, attachmentIDs, keys []string) error {
	args := CleanupAttachmentObjectsArgs{SpaceID: spaceID, AttachmentIDs: attachmentIDs, Keys: keys}
	opts := args.InsertOpts()
	opts.ScheduledAt = time.Now().Add(cleanupAttachmentObjectsDelay)
	_, err := d.txClient.InsertTx(ctx, tx, args, &opts)
	return err
}

// HasActiveAttachmentProcessingは、書き戻す可能性がある処理ジョブが残っているかを返す。
// 行の削除前から実行中の処理だけでなく、再試行待ちの処理も終了してから実体を回収する。
func (d *Dispatcher) HasActiveAttachmentProcessing(ctx context.Context, spaceID string, attachmentIDs []string) (bool, error) {
	ids, err := json.Marshal(attachmentIDs)
	if err != nil {
		return false, err
	}
	jobs, err := d.txClient.JobList(ctx, river.NewJobListParams().
		Kinds(ProcessAttachmentArgs{}.Kind()).
		States(rivertype.JobStateAvailable, rivertype.JobStatePending, rivertype.JobStateRetryable, rivertype.JobStateRunning, rivertype.JobStateScheduled).
		Where("args ->> 'space_id' = @space_id AND args ->> 'attachment_id' IN (SELECT jsonb_array_elements_text(@attachment_ids::jsonb))", river.NamedArgs{
			"space_id": spaceID, "attachment_ids": string(ids),
		}).First(1))
	if err != nil {
		return false, err
	}
	return len(jobs.Jobs) > 0, nil
}

// CleanupExportObjectsArgsは、削除したエクスポートのZIPのキーを行に依存せず保持する。
// LegacyBlobsは共有中も含む候補で、後始末が実行時の参照を確認する。
type CleanupExportObjectsArgs struct {
	SpaceID     string                 `json:"space_id"`
	ExportIDs   []string               `json:"export_ids"`
	Keys        []string               `json:"keys"`
	LegacyBlobs []LegacyExportBlobArgs `json:"legacy_blobs"`
}

// LegacyExportBlobArgsは、共有参照の確認と再試行のためジョブに保持するblobのIDとキー。
type LegacyExportBlobArgs struct {
	ID  string `json:"id"`
	Key string `json:"key"`
}

// Kindはジョブの種類を返す。
func (CleanupExportObjectsArgs) Kind() string { return "cleanup_export_objects" }

// InsertOptsはジョブのInsertオプションを返す。
func (CleanupExportObjectsArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: river.QueueDefault, MaxAttempts: 10}
}

// EnqueueCleanupExportObjectsTxは、行の削除と同じトランザクションでGo版のZIPの後始末を登録する。
// exportIDsは、後始末が生成ジョブの終了を待つエクスポート。
func (d *Dispatcher) EnqueueCleanupExportObjectsTx(ctx context.Context, tx *sql.Tx, spaceID string, exportIDs, keys []string) error {
	args := CleanupExportObjectsArgs{SpaceID: spaceID, ExportIDs: exportIDs, Keys: keys}
	opts := args.InsertOpts()
	_, err := d.txClient.InsertTx(ctx, tx, args, &opts)
	return err
}

// EnqueueCleanupLegacyExportBlobsTxは、行の削除と同じトランザクションでRails期のZIPのblobの後始末を登録する。
// blobIDsとblobKeysは同じ順に並べる。
func (d *Dispatcher) EnqueueCleanupLegacyExportBlobsTx(ctx context.Context, tx *sql.Tx, spaceID string, blobIDs, blobKeys []string) error {
	if len(blobIDs) != len(blobKeys) {
		return fmt.Errorf("blobのIDとキーの数が一致しません (ID: %d、キー: %d)", len(blobIDs), len(blobKeys))
	}
	blobs := make([]LegacyExportBlobArgs, len(blobIDs))
	for i := range blobIDs {
		blobs[i] = LegacyExportBlobArgs{ID: blobIDs[i], Key: blobKeys[i]}
	}
	args := CleanupExportObjectsArgs{SpaceID: spaceID, LegacyBlobs: blobs}
	opts := args.InsertOpts()
	_, err := d.txClient.InsertTx(ctx, tx, args, &opts)
	return err
}

// HasActiveExportGenerationは、ZIPを書き戻す可能性がある生成ジョブが残っているかを返す。
func (d *Dispatcher) HasActiveExportGeneration(ctx context.Context, spaceID string, exportIDs []string) (bool, error) {
	if len(exportIDs) == 0 {
		return false, nil
	}
	ids, err := json.Marshal(exportIDs)
	if err != nil {
		return false, err
	}
	jobs, err := d.txClient.JobList(ctx, river.NewJobListParams().
		Kinds(GenerateExportFilesArgs{}.Kind()).
		States(rivertype.JobStateAvailable, rivertype.JobStatePending, rivertype.JobStateRetryable, rivertype.JobStateRunning, rivertype.JobStateScheduled).
		Where("args ->> 'space_id' = @space_id AND args ->> 'export_id' IN (SELECT jsonb_array_elements_text(@export_ids::jsonb))", river.NamedArgs{
			"space_id": spaceID, "export_ids": string(ids),
		}).First(1))
	if err != nil {
		return false, err
	}
	return len(jobs.Jobs) > 0, nil
}
