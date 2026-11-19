package usecase

import (
	"bytes"
	"context"
	"crypto/md5" //nolint:gosec // ActiveStorageのチェックサムの形式 (MD5のBase64) に合わせるためで、改ざんの検知には使わない
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/wikinoapp/wikino/go/internal/imagemeta"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
	"github.com/wikinoapp/wikino/go/internal/svgsanitizer"
)

const (
	// attachmentUploadMinWaitとattachmentUploadMaxWaitは、アップロードを待つときに先送りする時間の
	// 下限と上限。事前署名からの経過時間だけ先送りし、確かめる間隔を倍々に延ばす
	attachmentUploadMinWait = 5 * time.Second
	attachmentUploadMaxWait = 5 * time.Minute

	// attachmentProcessingCleanupTimeoutは、処理待ちに戻す更新を待つ上限。ジョブの打ち切りからは
	// 切り離すが、DBのロック待ちでWorkerの実行枠を占有し続けないよう、独立した期限を持つ
	attachmentProcessingCleanupTimeout = 10 * time.Second
)

// attachmentTransformsは、種類ごとの、アップロードした内容の書き換え。ここに無い種類 (GIF・動画など) は
// 書き換えずに処理を終える。GIFは、Rails版と同じくアニメーションを保つために書き換えない
var attachmentTransforms = map[string]func([]byte) ([]byte, error){
	"image/jpeg":    imagemeta.Strip,
	"image/jpg":     imagemeta.Strip,
	"image/png":     imagemeta.Strip,
	"image/webp":    imagemeta.Strip,
	"image/svg+xml": svgsanitizer.Sanitize,
}

// errAttachmentUnprocessableは、添付ファイルの内容を処理できないことを表す。同じ内容で何度試しても
// 結果が変わらないため、再試行せずに失敗として記録する
var errAttachmentUnprocessable = errors.New("添付ファイルの内容を処理できません")

// ProcessAttachmentUsecaseは、事前署名した添付ファイルがアップロードされるのを待ち、画像のメタデータを
// 取り除き、SVGをサニタイズして、処理の状態を完了にするユースケース
type ProcessAttachmentUsecase struct {
	db             *sql.DB
	attachmentRepo *repository.AttachmentRepository
	objectStorage  storage.ObjectStorage
	now            func() time.Time
	cleanupTimeout time.Duration
}

// NewProcessAttachmentUsecaseはProcessAttachmentUsecaseを生成する
func NewProcessAttachmentUsecase(
	db *sql.DB,
	attachmentRepo *repository.AttachmentRepository,
	objectStorage storage.ObjectStorage,
) *ProcessAttachmentUsecase {
	return &ProcessAttachmentUsecase{
		db:             db,
		attachmentRepo: attachmentRepo,
		objectStorage:  objectStorage,
		now:            time.Now,
		cleanupTimeout: attachmentProcessingCleanupTimeout,
	}
}

// ProcessAttachmentInputはアップロード後の処理の入力パラメータ
type ProcessAttachmentInput struct {
	AttachmentID model.AttachmentID
	SpaceID      model.SpaceID
}

// ProcessAttachmentOutputはアップロード後の処理の結果
type ProcessAttachmentOutput struct {
	// RetryAfterは、オブジェクトがまだアップロードされていないため、処理を先送りする時間。
	// 0なら処理を終えた (完了・失敗・何もしなかった)
	RetryAfter time.Duration
}

// Executeは、添付ファイルが処理待ちのときだけ処理する。オブジェクトがまだ無ければ先送りする時間を返し、
// 事前署名から上限の時間を過ぎても無ければ失敗にする。オブジェクトがあれば処理中にして、種類に応じて
// 内容を書き換え、blobと状態を更新する。処理できない内容は失敗にしてエラーを返さず、オブジェクト
// ストレージやデータベースの失敗は、状態を処理待ちに戻してエラーを返す (riverの再試行に任せる)
func (uc *ProcessAttachmentUsecase) Execute(ctx context.Context, input ProcessAttachmentInput) (*ProcessAttachmentOutput, error) {
	attachment, err := uc.attachmentRepo.FindForProcessingByIDAndSpace(ctx, input.AttachmentID, input.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("添付ファイルの取得に失敗: %w", err)
	}
	// 行が無いのは削除されたときで、処理待ちでないのはほかの試行が処理を終えたときのため、何もしない
	if attachment == nil || attachment.ProcessingStatus != model.AttachmentProcessingStatusPending {
		return &ProcessAttachmentOutput{}, nil
	}

	exists, err := uc.objectStorage.Exists(ctx, attachment.BlobKey)
	if err != nil {
		return nil, fmt.Errorf("添付ファイルのオブジェクトの存在の確認に失敗: %w", err)
	}
	if !exists {
		return uc.waitForUpload(ctx, attachment)
	}

	claimed, err := uc.attachmentRepo.UpdateProcessingStatus(ctx, attachment.ID, attachment.SpaceID,
		model.AttachmentProcessingStatusPending, model.AttachmentProcessingStatusProcessing, uc.now())
	if err != nil {
		return nil, fmt.Errorf("添付ファイルの処理の状態の更新に失敗: %w", err)
	}
	if !claimed {
		return &ProcessAttachmentOutput{}, nil
	}

	err = uc.process(ctx, attachment)
	if errors.Is(err, errAttachmentUnprocessable) {
		// 失敗として記録できなかったときは、処理待ちに戻し、次の試行で処理し直して記録する
		if failErr := uc.fail(ctx, attachment, model.AttachmentProcessingStatusProcessing, err); failErr != nil {
			return nil, errors.Join(failErr, uc.release(ctx, attachment))
		}
		return &ProcessAttachmentOutput{}, nil
	}
	if err != nil {
		return nil, errors.Join(err, uc.release(ctx, attachment))
	}
	return &ProcessAttachmentOutput{}, nil
}

// waitForUploadは、まだアップロードされていない添付ファイルについて、先送りする時間を返す。
// 事前署名から上限の時間を過ぎていれば、失敗にする
func (uc *ProcessAttachmentUsecase) waitForUpload(ctx context.Context, attachment *model.Attachment) (*ProcessAttachmentOutput, error) {
	elapsed := uc.now().Sub(attachment.AttachedAt)
	if elapsed >= model.AttachmentUploadDeadline {
		cause := fmt.Errorf("事前署名から%sを過ぎてもアップロードされていません", model.AttachmentUploadDeadline)
		return &ProcessAttachmentOutput{}, uc.fail(ctx, attachment, model.AttachmentProcessingStatusPending, cause)
	}
	return &ProcessAttachmentOutput{RetryAfter: attachmentUploadWait(elapsed)}, nil
}

// attachmentUploadWaitは、事前署名からelapsedが経ったときに、アップロードを待って先送りする時間を返す。
// 上限の時間を過ぎた直後にもう一度確かめられるよう、上限の時間を越えて先送りしない
func attachmentUploadWait(elapsed time.Duration) time.Duration {
	wait := min(max(elapsed, attachmentUploadMinWait), attachmentUploadMaxWait)
	return min(wait, model.AttachmentUploadDeadline-elapsed+time.Second)
}

// processは、処理中にした添付ファイルの内容を種類に応じて書き換え、blobと状態を更新する
func (uc *ProcessAttachmentUsecase) process(ctx context.Context, attachment *model.Attachment) error {
	transform, ok := attachmentTransforms[attachment.ContentType]
	if !ok {
		return uc.complete(ctx, attachment, nil)
	}

	content, err := uc.download(ctx, attachment)
	if err != nil {
		return err
	}
	processed, err := transform(content)
	if errors.Is(err, imagemeta.ErrMalformed) || errors.Is(err, imagemeta.ErrUnsupportedFormat) || errors.Is(err, svgsanitizer.ErrMalformed) {
		return fmt.Errorf("%w: %w", errAttachmentUnprocessable, err)
	}
	if err != nil {
		return fmt.Errorf("添付ファイルの内容の書き換えに失敗: %w", err)
	}

	if !bytes.Equal(processed, content) {
		if err := uc.objectStorage.Upload(ctx, storage.UploadInput{
			Key:         attachment.BlobKey,
			Body:        bytes.NewReader(processed),
			ContentType: attachment.ContentType,
		}); err != nil {
			return fmt.Errorf("書き換えた添付ファイルの書き戻しに失敗: %w", err)
		}
	}
	return uc.complete(ctx, attachment, processed)
}

// downloadは、添付ファイルの内容を、種類ごとの大きさの上限まで読む。上限を超える内容は処理できない
func (uc *ProcessAttachmentUsecase) download(ctx context.Context, attachment *model.Attachment) ([]byte, error) {
	body, err := uc.objectStorage.Get(ctx, attachment.BlobKey)
	if err != nil {
		return nil, fmt.Errorf("添付ファイルのオブジェクトの取得に失敗: %w", err)
	}
	defer func() { _ = body.Close() }()

	limit := model.AttachmentMaxByteSize(attachment.ContentType)
	content, err := io.ReadAll(io.LimitReader(body, limit+1))
	if err != nil {
		return nil, fmt.Errorf("添付ファイルのオブジェクトの読み取りに失敗: %w", err)
	}
	if int64(len(content)) > limit {
		return nil, fmt.Errorf("%w: 大きさが上限の%dバイトを超えています", errAttachmentUnprocessable, limit)
	}
	return content, nil
}

// completeは、処理中の添付ファイルを完了にする。contentを書き換えた内容として渡したときは、blobの
// バイト数とチェックサムも同じトランザクションで更新する
func (uc *ProcessAttachmentUsecase) complete(ctx context.Context, attachment *model.Attachment, content []byte) error {
	var byteSize int64
	var checksum string
	if content != nil {
		byteSize = int64(len(content))
		checksum = attachmentChecksum(content)
	}

	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("トランザクションの開始に失敗: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	attachmentRepo := uc.attachmentRepo.WithTx(tx)
	if content != nil {
		if err := attachmentRepo.UpdateBlobContent(ctx, attachment.ID, attachment.SpaceID, byteSize, checksum); err != nil {
			return fmt.Errorf("添付ファイルのblobの更新に失敗: %w", err)
		}
	}
	completed, err := attachmentRepo.UpdateProcessingStatus(ctx, attachment.ID, attachment.SpaceID,
		model.AttachmentProcessingStatusProcessing, model.AttachmentProcessingStatusCompleted, uc.now())
	if err != nil {
		return fmt.Errorf("添付ファイルの処理の状態の更新に失敗: %w", err)
	}
	// ほかの処理が先に状態を変えていたら、blobも更新しない
	if !completed {
		return nil
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("トランザクションのコミットに失敗: %w", err)
	}
	return nil
}

// failは、添付ファイルの状態をfromから失敗にし、理由をログに残す
func (uc *ProcessAttachmentUsecase) fail(ctx context.Context, attachment *model.Attachment, from model.AttachmentProcessingStatus, cause error) error {
	if _, err := uc.attachmentRepo.UpdateProcessingStatus(ctx, attachment.ID, attachment.SpaceID,
		from, model.AttachmentProcessingStatusFailed, uc.now()); err != nil {
		return fmt.Errorf("添付ファイルの処理の状態の更新に失敗: %w", err)
	}
	slog.WarnContext(ctx, "添付ファイルのアップロード後の処理を失敗にしました",
		"attachment_id", attachment.ID.String(),
		"space_id", attachment.SpaceID.String(),
		"content_type", attachment.ContentType,
		"reason", cause.Error(),
	)
	return nil
}

// releaseは、処理中にした添付ファイルを処理待ちに戻し、次の試行で処理し直せるようにする。
// 試行のcontextが打ち切られて失敗したときも戻せるよう、打ち切りから切り離し、独立した期限で実行する
func (uc *ProcessAttachmentUsecase) release(ctx context.Context, attachment *model.Attachment) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), uc.cleanupTimeout)
	defer cancel()

	if _, err := uc.attachmentRepo.UpdateProcessingStatus(cleanupCtx, attachment.ID, attachment.SpaceID,
		model.AttachmentProcessingStatusProcessing, model.AttachmentProcessingStatusPending, uc.now()); err != nil {
		return fmt.Errorf("添付ファイルの処理の状態を処理待ちに戻せませんでした: %w", err)
	}
	return nil
}

// attachmentChecksumは、ActiveStorageと同じ形式 (MD5のBase64) のチェックサムを返す
func attachmentChecksum(content []byte) string {
	sum := md5.Sum(content) //nolint:gosec // ActiveStorageのチェックサムの形式に合わせる
	return base64.StdEncoding.EncodeToString(sum[:])
}
