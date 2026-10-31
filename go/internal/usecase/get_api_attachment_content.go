package usecase

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
)

// GetAPIAttachmentContentUsecaseは、公開APIの呼び出し主体が見られる添付ファイルの中身を、種類と大きさの
// 条件を満たすときだけオブジェクトストレージから読むユースケース。MCPのツールが、クライアントが署名付きURLを
// 取りに行けなくてもモデルが画像を見られるよう、中身をツールの結果に入れるために使う
type GetAPIAttachmentContentUsecase struct {
	attachmentRepo *repository.AttachmentRepository
	objectStorage  storage.ObjectStorage
}

// NewGetAPIAttachmentContentUsecaseはGetAPIAttachmentContentUsecaseを生成する
func NewGetAPIAttachmentContentUsecase(attachmentRepo *repository.AttachmentRepository, objectStorage storage.ObjectStorage) *GetAPIAttachmentContentUsecase {
	return &GetAPIAttachmentContentUsecase{
		attachmentRepo: attachmentRepo,
		objectStorage:  objectStorage,
	}
}

// GetAPIAttachmentContentInputは添付ファイルの中身の取得の入力パラメータ
type GetAPIAttachmentContentInput struct {
	Principal       *model.APIPrincipal
	SpaceIdentifier model.SpaceIdentifier
	AttachmentID    model.AttachmentID

	// ContentTypesは中身を読む種類 (`image/png` など)。記録された種類と、中身から判別した種類の両方が
	// この中にあるときだけ中身を返す
	ContentTypes []string
	// MaxBytesは中身を読む大きさの上限 (バイト)
	MaxBytes int64
}

// GetAPIAttachmentContentOutputは添付ファイルの中身の取得結果
type GetAPIAttachmentContentOutput struct {
	Attachment *model.Attachment
	// Contentは添付ファイルの中身。種類か大きさの条件を満たさなければnil
	Content []byte
	// ContentTypeは中身から判別した種類。Contentがnilなら空
	ContentType string
}

// Executeは、添付ファイルの閲覧の判定 (GetAPIAttachmentUsecaseと同じ) を通し、条件を満たせば中身を読む。
// 中身を返さないとき (種類が対象でない・上限を超える) もエラーにせず、添付ファイルだけを返す。
// 記録された種類はアップロードのときの申告なので、それだけを信用せず、中身から判別した種類も確認する。
// 両方がContentTypesに含まれるときだけ、中身から判別した種類で返す
func (uc *GetAPIAttachmentContentUsecase) Execute(ctx context.Context, input GetAPIAttachmentContentInput) (*GetAPIAttachmentContentOutput, error) {
	if uc.objectStorage == nil {
		return nil, errors.New("オブジェクトストレージが設定されていないため添付ファイルを取得できません")
	}

	attachment, err := findViewableAPIAttachment(ctx, uc.attachmentRepo, input.Principal, input.SpaceIdentifier, input.AttachmentID)
	if err != nil {
		return nil, err
	}

	output := &GetAPIAttachmentContentOutput{Attachment: attachment}
	if !slices.Contains(input.ContentTypes, attachment.ContentType) || attachment.ByteSize > input.MaxBytes {
		return output, nil
	}

	content, err := uc.readContent(ctx, attachment.BlobKey, input.MaxBytes)
	if err != nil {
		return nil, err
	}
	if content == nil {
		return output, nil
	}
	detected := http.DetectContentType(content)
	if !slices.Contains(input.ContentTypes, detected) {
		return output, nil
	}
	output.Content = content
	output.ContentType = detected
	return output, nil
}

// readContentは、オブジェクトストレージからキーkeyの中身を上限maxBytesまで読む。
// 記録されたバイト数と実際の中身がずれて上限を超えた場合はnilを返す
func (uc *GetAPIAttachmentContentUsecase) readContent(ctx context.Context, key string, maxBytes int64) ([]byte, error) {
	body, err := uc.objectStorage.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("添付ファイルの中身の取得に失敗: %w", err)
	}
	defer func() { _ = body.Close() }()

	content, err := io.ReadAll(io.LimitReader(body, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("添付ファイルの中身の読み込みに失敗: %w", err)
	}
	if int64(len(content)) > maxBytes {
		return nil, nil
	}
	return content, nil
}
