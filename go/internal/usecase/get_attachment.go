package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
)

// GetAttachmentUsecaseは、Web画面の `/attachments/{id}` で、閲覧者が見られる添付ファイルの中身を
// 取得する署名付きURLを返すユースケース。中身はアプリケーションで中継せず、ブラウザがオブジェクト
// ストレージから直接取得する
type GetAttachmentUsecase struct {
	attachmentRepo  *repository.AttachmentRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	objectStorage   storage.ObjectStorage
}

// NewGetAttachmentUsecaseはGetAttachmentUsecaseを生成する
func NewGetAttachmentUsecase(
	attachmentRepo *repository.AttachmentRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	objectStorage storage.ObjectStorage,
) *GetAttachmentUsecase {
	return &GetAttachmentUsecase{
		attachmentRepo:  attachmentRepo,
		spaceMemberRepo: spaceMemberRepo,
		objectStorage:   objectStorage,
	}
}

// GetAttachmentInputは添付ファイルの取得の入力パラメータ
type GetAttachmentInput struct {
	AttachmentID model.AttachmentID
	// UserIDは閲覧者のユーザーID。ログインしていなければnil
	UserID *model.UserID
}

// GetAttachmentOutputは添付ファイルの取得結果
type GetAttachmentOutput struct {
	// DownloadURLは中身を取得する署名付きURL。応答の種類と表示のしかたを署名に含める
	DownloadURL string
}

// Executeは添付ファイルを引き、閲覧者が見られるときだけ中身への署名付きURLを返す。
// 閲覧の判定は公開APIの添付ファイルの取得と同じく、添付ファイルのスペースでattachment:readを持つか、
// 参照するページがすべて公開トピックにあるかで行う。添付ファイルが存在しない場合に加え、見られない場合も
// 存在を秘匿するため未存在 (*model.AppError) として扱う
func (uc *GetAttachmentUsecase) Execute(ctx context.Context, input GetAttachmentInput) (*GetAttachmentOutput, error) {
	if uc.objectStorage == nil {
		return nil, errors.New("オブジェクトストレージが設定されていないため添付ファイルを表示できません")
	}

	notFound := &model.AppError{
		Code:    model.AppErrCodeResourceNotFound,
		UserMsg: i18n.T(ctx, "error_not_found_message"),
	}

	// URLにスペースを含まないため、IDだけで添付ファイルとそのスペースを引く。以降の判定は、
	// 引いた添付ファイルのスペースで行う
	attachment, err := uc.attachmentRepo.FindBlobByID(ctx, input.AttachmentID)
	if err != nil {
		return nil, fmt.Errorf("添付ファイルの取得に失敗: %w", err)
	}
	if attachment == nil {
		return nil, notFound
	}

	var spaceMember *model.SpaceMember
	if input.UserID != nil {
		spaceMember, err = uc.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, attachment.SpaceID, *input.UserID)
		if err != nil {
			return nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
		}
	}

	public, err := uc.attachmentRepo.FindPubliclyReferencedBlobByIDAndSpace(ctx, attachment.ID, attachment.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("添付ファイルの参照元の公開範囲の取得に失敗: %w", err)
	}
	allReferencingPagesPublic := public != nil

	// attachment:readはスペースにしか付けられず、判定でトピックを見ないため、トピックメンバーは要らない
	if !newAuthorizer(spaceMember, nil).CanShowAttachment(allReferencingPagesPublic) {
		return nil, notFound
	}

	url, err := uc.objectStorage.PresignedGetURL(ctx, attachment.BlobKey, model.AttachmentDownloadExpiration, attachmentDeliveryOptions(attachment))
	if err != nil {
		return nil, fmt.Errorf("添付ファイルの署名付きURLの生成に失敗: %w", err)
	}
	return &GetAttachmentOutput{DownloadURL: url}, nil
}
