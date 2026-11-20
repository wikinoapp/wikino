package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
)

// GetSpaceSettingsAttachmentsUsecaseは、スペースの添付ファイルの管理の画面
// (GET /s/{space_identifier}/settings/attachments) に出すデータを集める。
type GetSpaceSettingsAttachmentsUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	attachmentRepo  *repository.AttachmentRepository
	objectStorage   storage.ObjectStorage
}

// NewGetSpaceSettingsAttachmentsUsecaseはGetSpaceSettingsAttachmentsUsecaseを生成する。objectStorageは
// nilでもよく、そのときはサムネイルの署名付きURLを作らずに一覧を返す
func NewGetSpaceSettingsAttachmentsUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	attachmentRepo *repository.AttachmentRepository,
	objectStorage storage.ObjectStorage,
) *GetSpaceSettingsAttachmentsUsecase {
	return &GetSpaceSettingsAttachmentsUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		attachmentRepo:  attachmentRepo,
		objectStorage:   objectStorage,
	}
}

// GetSpaceSettingsAttachmentsInputは添付ファイルの管理の画面の取得の入力
type GetSpaceSettingsAttachmentsInput struct {
	SpaceIdentifier model.SpaceIdentifier
	UserID          model.UserID
	// Queryはファイル名の検索語。空白だけのときは絞り込まない
	Query     string
	Page      int32
	PageLimit int32
}

// GetSpaceSettingsAttachmentsOutputは添付ファイルの管理の画面の取得の出力
type GetSpaceSettingsAttachmentsOutput struct {
	Space       *model.Space
	Attachments []*model.Attachment
	TotalCount  int64
	// ThumbnailURLsは、一覧の画像の添付ファイルのIDをキーにした、サムネイルの署名付きURL。
	// 画像でない添付ファイルと、オブジェクトストレージが設定されていないときは含めない
	ThumbnailURLs map[model.AttachmentID]string
	// CanDeleteAttachmentは、閲覧者が添付ファイルを削除できる (attachment:deleteを持つ) かどうか
	CanDeleteAttachment bool
}

// Executeは、スペースでspace:writeを持つメンバーに、スペースの添付ファイルを新しい順に返す。
//
// スペースのアクティブなメンバーでない閲覧者は「見つからない」として答え、space:writeを持たない
// メンバーには「許可されていない」を返す (一般設定と同じ)。
func (uc *GetSpaceSettingsAttachmentsUsecase) Execute(ctx context.Context, input GetSpaceSettingsAttachmentsInput) (*GetSpaceSettingsAttachmentsOutput, error) {
	space, err := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
	if err != nil {
		return nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, notFoundError(ctx)
	}

	spaceMember, err := uc.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}
	if spaceMember == nil {
		return nil, notFoundError(ctx)
	}

	authorizer := newAuthorizer(spaceMember, nil)
	if !authorizer.CanUpdateSpace() {
		return nil, forbiddenError(ctx)
	}

	result, err := uc.attachmentRepo.ListBySpacePaginated(ctx, space.ID, input.Query, input.Page, input.PageLimit)
	if err != nil {
		return nil, fmt.Errorf("添付ファイルの一覧の取得に失敗: %w", err)
	}

	thumbnailURLs, err := uc.presignThumbnails(ctx, result.Attachments)
	if err != nil {
		return nil, err
	}

	return &GetSpaceSettingsAttachmentsOutput{
		Space:               space,
		Attachments:         result.Attachments,
		TotalCount:          result.TotalCount,
		ThumbnailURLs:       thumbnailURLs,
		CanDeleteAttachment: authorizer.CanDeleteAttachment(),
	}, nil
}

// presignThumbnailsは、一覧の画像の添付ファイルに、サムネイルの署名付きURLを作る。応答ヘッダーは
// 本文の中の表示 (`POST /attachments/signed_urls`) と同じ規則で決め、SVGも `<img>` で表示できるようにする。
// オブジェクトストレージが設定されていなければ、サムネイルを出さずに空のmapを返す
func (uc *GetSpaceSettingsAttachmentsUsecase) presignThumbnails(ctx context.Context, attachments []*model.Attachment) (map[model.AttachmentID]string, error) {
	urls := map[model.AttachmentID]string{}
	if uc.objectStorage == nil {
		return urls, nil
	}

	for _, attachment := range attachments {
		if !attachment.IsImage() {
			continue
		}
		url, err := uc.objectStorage.PresignedGetURL(ctx, attachment.BlobKey, model.AttachmentSignedURLExpiration, attachmentSignedURLOptions(attachment))
		if err != nil {
			return nil, fmt.Errorf("添付ファイルのサムネイルの署名付きURLの生成に失敗: %w", err)
		}
		urls[attachment.ID] = url
	}
	return urls, nil
}
