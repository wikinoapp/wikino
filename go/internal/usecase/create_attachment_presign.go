package usecase

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// CreateAttachmentPresignUsecaseは、エディタにドロップしたファイルをブラウザがオブジェクトストレージへ
// 直接アップロードできるよう、添付ファイルを処理待ちの状態で作り、アップロードの署名付きURLを返すユースケース
// (`POST /s/{space_identifier}/attachments/presign`)。アップロードの完了は、投入したジョブが
// オブジェクトストレージを確かめて検知する
type CreateAttachmentPresignUsecase struct {
	db               *sql.DB
	spaceRepo        *repository.SpaceRepository
	spaceMemberRepo  *repository.SpaceMemberRepository
	attachmentRepo   *repository.AttachmentRepository
	presignValidator *validator.AttachmentPresignCreateValidator
	dispatcher       *dispatcher.Dispatcher
	objectStorage    storage.ObjectStorage
}

// NewCreateAttachmentPresignUsecaseはCreateAttachmentPresignUsecaseを生成する
func NewCreateAttachmentPresignUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	attachmentRepo *repository.AttachmentRepository,
	presignValidator *validator.AttachmentPresignCreateValidator,
	d *dispatcher.Dispatcher,
	objectStorage storage.ObjectStorage,
) *CreateAttachmentPresignUsecase {
	return &CreateAttachmentPresignUsecase{
		db:               db,
		spaceRepo:        spaceRepo,
		spaceMemberRepo:  spaceMemberRepo,
		attachmentRepo:   attachmentRepo,
		presignValidator: presignValidator,
		dispatcher:       d,
		objectStorage:    objectStorage,
	}
}

// CreateAttachmentPresignInputは事前署名の入力パラメータ
type CreateAttachmentPresignInput struct {
	SpaceIdentifier model.SpaceIdentifier
	UserID          model.UserID
	Filename        string
	ContentType     string
	// ByteSizeはアップロードする大きさ。リクエストに無ければnil
	ByteSize *int64
	// Checksumは内容のMD5をBase64にした値
	Checksum string
}

// CreateAttachmentPresignOutputは事前署名の結果
type CreateAttachmentPresignOutput struct {
	AttachmentID model.AttachmentID
	// DirectUploadURLは、ブラウザがPUTで内容を送る署名付きURL
	DirectUploadURL string
	// DirectUploadHeadersは、ブラウザがアップロードのときに送るヘッダー
	DirectUploadHeaders map[string]string
}

// Executeは、スペースでattachment:writeを持つメンバーのリクエストを検証し、添付ファイルを処理待ちの状態で作って、
// アップロード後の処理のジョブを投入し、アップロードの署名付きURLを返す。ジョブを投入できなかったときは、
// 署名付きURLを返さずにエラーにする。作った行は処理待ちのまま残るが、アップロードされないため配信もされない
func (uc *CreateAttachmentPresignUsecase) Execute(ctx context.Context, input CreateAttachmentPresignInput) (*CreateAttachmentPresignOutput, error) {
	if uc.objectStorage == nil {
		return nil, errors.New("オブジェクトストレージが設定されていないため添付ファイルのアップロードを署名できません")
	}

	space, spaceMember, err := uc.fetchSpaceMember(ctx, input)
	if err != nil {
		return nil, err
	}

	// 添付ファイルはスペースに属し、attachment:writeはスペースにしか付けられないため、トピックメンバーは要らない
	if !newAuthorizer(spaceMember, nil).CanCreateAttachment() {
		return nil, &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	filename, err := uc.presignValidator.Validate(ctx, validator.AttachmentPresignCreateValidatorInput{
		Filename:    input.Filename,
		ContentType: input.ContentType,
		ByteSize:    input.ByteSize,
		Checksum:    input.Checksum,
	})
	if err != nil {
		return nil, err
	}

	// キーはサーバーが決め、利用者が送ったファイル名をキーに使わない
	blobKey, err := auth.GenerateBlobKey()
	if err != nil {
		return nil, fmt.Errorf("添付ファイルのキーの生成に失敗: %w", err)
	}
	// 署名は行を作る前に済ませ、署名できないときに処理待ちの行を残さないようにする
	uploadURL, err := uc.objectStorage.PresignedPutURL(ctx, blobKey, model.AttachmentUploadURLExpiration, storage.PresignedPutOptions{
		ContentType:   input.ContentType,
		ContentLength: *input.ByteSize,
		ContentMD5:    input.Checksum,
	})
	if err != nil {
		return nil, fmt.Errorf("添付ファイルのアップロードの署名付きURLの生成に失敗: %w", err)
	}

	attachmentID, err := uc.createAttachment(ctx, repository.CreateAttachmentInput{
		SpaceID:       space.ID,
		UserID:        input.UserID,
		SpaceMemberID: spaceMember.ID,
		BlobKey:       blobKey,
		Filename:      filename,
		ContentType:   input.ContentType,
		ByteSize:      *input.ByteSize,
		Checksum:      input.Checksum,
		AttachedAt:    time.Now(),
	})
	if err != nil {
		return nil, err
	}

	if err := uc.dispatcher.EnqueueProcessAttachment(ctx, string(attachmentID), string(space.ID)); err != nil {
		return nil, fmt.Errorf("添付ファイルのアップロード後の処理のジョブの投入に失敗 (attachment_id: %s): %w", attachmentID, err)
	}

	return &CreateAttachmentPresignOutput{
		AttachmentID:    attachmentID,
		DirectUploadURL: uploadURL,
		// ActiveStorageのheaders_for_direct_uploadと同じ組にする。Content-Dispositionは署名に
		// 含めず、オブジェクトに保存させるだけである
		DirectUploadHeaders: map[string]string{
			"Content-Type":        input.ContentType,
			"Content-MD5":         input.Checksum,
			"Content-Disposition": formatContentDisposition("inline", sanitizeAttachmentFilename(filename)),
		},
	}, nil
}

// fetchSpaceMemberは、スペースと、閲覧者のそのスペースでのメンバーを引く。スペースが無ければ未存在に、
// メンバーでなければ権限不足にする (スペースの存在は隠さない)
func (uc *CreateAttachmentPresignUsecase) fetchSpaceMember(ctx context.Context, input CreateAttachmentPresignInput) (*model.Space, *model.SpaceMember, error) {
	space, err := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
	if err != nil {
		return nil, nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	spaceMember, err := uc.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, input.UserID)
	if err != nil {
		return nil, nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}
	return space, spaceMember, nil
}

// createAttachmentは、添付ファイルの3つの行を1つのトランザクションで作る
func (uc *CreateAttachmentPresignUsecase) createAttachment(ctx context.Context, input repository.CreateAttachmentInput) (model.AttachmentID, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("トランザクションの開始に失敗: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	attachmentID, err := uc.attachmentRepo.WithTx(tx).Create(ctx, input)
	if err != nil {
		return "", fmt.Errorf("添付ファイルの作成に失敗: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("トランザクションのコミットに失敗: %w", err)
	}
	return attachmentID, nil
}
