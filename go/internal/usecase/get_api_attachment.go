package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
)

// GetAPIAttachmentUsecaseは、公開APIでトークンが属するスペースの添付ファイルの情報と、
// 中身をダウンロードする署名付きURLを返すユースケース。
//
// 中身はAPIのサーバーで中継せず、クライアントがオブジェクトストレージから直接取得する
// (エクスポートのダウンロードと同じ形)
type GetAPIAttachmentUsecase struct {
	attachmentRepo *repository.AttachmentRepository
	objectStorage  storage.ObjectStorage
}

// NewGetAPIAttachmentUsecaseはGetAPIAttachmentUsecaseを生成する
func NewGetAPIAttachmentUsecase(attachmentRepo *repository.AttachmentRepository, objectStorage storage.ObjectStorage) *GetAPIAttachmentUsecase {
	return &GetAPIAttachmentUsecase{
		attachmentRepo: attachmentRepo,
		objectStorage:  objectStorage,
	}
}

// GetAPIAttachmentInputは添付ファイルの取得に必要な入力パラメータ
type GetAPIAttachmentInput struct {
	Principal       *model.APIPrincipal
	SpaceIdentifier model.SpaceIdentifier
	AttachmentID    model.AttachmentID
}

// GetAPIAttachmentOutputは添付ファイルの取得結果
type GetAPIAttachmentOutput struct {
	Attachment *model.Attachment
	// DownloadURLは中身をダウンロードする署名付きURL
	DownloadURL string
	// DownloadURLExpiresAtはDownloadURLの有効期限
	DownloadURLExpiresAt time.Time
}

// Executeはattachment:readを持つトークンが属するスペースの添付ファイルを解決し、その中身へのURLに署名する。
// 閲覧の判定はWeb画面の `/attachments/{ID}` (Rails版のcan_view_attachment?) と同じく、
// attachment:readを持つか、参照するページがすべて公開トピックにあるかで行う。
// 添付ファイルが存在しない場合に加え、見る権限が無い場合も存在を秘匿するため未存在として扱う
func (uc *GetAPIAttachmentUsecase) Execute(ctx context.Context, input GetAPIAttachmentInput) (*GetAPIAttachmentOutput, error) {
	if uc.objectStorage == nil {
		return nil, errors.New("オブジェクトストレージが設定されていないため添付ファイルを取得できません")
	}

	attachment, err := findViewableAPIAttachment(ctx, uc.attachmentRepo, input.Principal, input.SpaceIdentifier, input.AttachmentID)
	if err != nil {
		return nil, err
	}

	// 有効期限は署名より前の時刻から求め、秒未満を切り捨てる。署名付きURLが実際に使えなくなるより
	// 後の時刻をクライアントに伝えないようにするためである
	expiresAt := time.Now().UTC().Truncate(time.Second).Add(model.AttachmentDownloadExpiration)
	url, err := uc.objectStorage.PresignedGetURL(ctx, attachment.BlobKey, model.AttachmentDownloadExpiration)
	if err != nil {
		return nil, fmt.Errorf("添付ファイルのダウンロードURLの生成に失敗: %w", err)
	}

	return &GetAPIAttachmentOutput{
		Attachment:           attachment,
		DownloadURL:          url,
		DownloadURLExpiresAt: expiresAt,
	}, nil
}
