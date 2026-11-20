package usecase

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
)

// DeleteAttachmentUsecaseは、スペースの添付ファイルを1件削除する。実体と、ページからの参照を含む行を消し、
// カバー画像にしていたページ・下書き・編集提案のページからはカバー画像が外れる
type DeleteAttachmentUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	attachmentRepo  *repository.AttachmentRepository
	deleter         *attachmentDeleter
}

// NewDeleteAttachmentUsecaseはDeleteAttachmentUsecaseを生成する。objectStorageはnilでもよく、そのときは
// 実体を持つ添付ファイルの削除がエラーになる
func NewDeleteAttachmentUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	attachmentRepo *repository.AttachmentRepository,
	objectStorage storage.ObjectStorage,
	d *dispatcher.Dispatcher,
) *DeleteAttachmentUsecase {
	return &DeleteAttachmentUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		attachmentRepo:  attachmentRepo,
		deleter:         &attachmentDeleter{db: db, attachmentRepo: attachmentRepo, objectStorage: objectStorage, dispatcher: d},
	}
}

// DeleteAttachmentInputは添付ファイルの削除の入力パラメータ
type DeleteAttachmentInput struct {
	SpaceIdentifier model.SpaceIdentifier
	UserID          model.UserID
	AttachmentID    model.AttachmentID
}

// Executeは、スペースでattachment:deleteを持つメンバーの依頼で、添付ファイルを削除する。
//
// スペースのアクティブなメンバーでない場合と、ほかのスペースの添付ファイル・存在しないID・不正な形のIDは
// 「見つからない」として答える。権限の無いメンバーには「許可されていない」を返す。
func (uc *DeleteAttachmentUsecase) Execute(ctx context.Context, input DeleteAttachmentInput) error {
	// 1. データ取得と認可チェック
	space, err := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
	if err != nil {
		return fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return notFoundError(ctx)
	}

	spaceMember, err := uc.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, input.UserID)
	if err != nil {
		return fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}
	if spaceMember == nil {
		return notFoundError(ctx)
	}

	if !newAuthorizer(spaceMember, nil).CanDeleteAttachment() {
		return forbiddenError(ctx)
	}

	exists, err := uc.attachmentRepo.ExistsByIDAndSpace(ctx, input.AttachmentID, space.ID)
	if err != nil {
		return fmt.Errorf("添付ファイルの存在の確認に失敗: %w", err)
	}
	if !exists {
		return notFoundError(ctx)
	}

	// 2. 実体と行の削除
	return uc.deleter.delete(ctx, space.ID, []model.AttachmentID{input.AttachmentID})
}
