package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
)

// GetAttachmentSignedURLsUsecaseは、Web画面のページ本文に添付ファイルを表示するため、指定したIDのうち
// 閲覧者が見られる添付ファイルの署名付きURLをまとめて返すユースケース (`POST /attachments/signed_urls`)
type GetAttachmentSignedURLsUsecase struct {
	attachmentRepo  *repository.AttachmentRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	objectStorage   storage.ObjectStorage
}

// NewGetAttachmentSignedURLsUsecaseはGetAttachmentSignedURLsUsecaseを生成する
func NewGetAttachmentSignedURLsUsecase(
	attachmentRepo *repository.AttachmentRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	objectStorage storage.ObjectStorage,
) *GetAttachmentSignedURLsUsecase {
	return &GetAttachmentSignedURLsUsecase{
		attachmentRepo:  attachmentRepo,
		spaceMemberRepo: spaceMemberRepo,
		objectStorage:   objectStorage,
	}
}

// GetAttachmentSignedURLsInputは署名付きURLの一括取得の入力パラメータ
type GetAttachmentSignedURLsInput struct {
	AttachmentIDs []model.AttachmentID
	// UserIDは閲覧者のユーザーID。ログインしていなければnil
	UserID *model.UserID
}

// GetAttachmentSignedURLsOutputは署名付きURLの一括取得の結果
type GetAttachmentSignedURLsOutput struct {
	// SignedURLsは、見られる添付ファイルのIDをキーにした署名付きURL。見られないIDと存在しないIDは含めない
	SignedURLs map[model.AttachmentID]string
}

// Executeは、指定したIDの添付ファイルのうち閲覧者が見られるものだけに署名する。閲覧の判定は
// `/attachments/{id}` と同じく、添付ファイルのスペースでattachment:readを持つか、参照するページが
// すべて公開トピックにあるかで行う。IDは複数のスペースにまたがりうるため、添付ファイルごとに
// そのスペースのメンバーシップで判定する。IDの数によらず、データベースへの問い合わせは3回までにする
func (uc *GetAttachmentSignedURLsUsecase) Execute(ctx context.Context, input GetAttachmentSignedURLsInput) (*GetAttachmentSignedURLsOutput, error) {
	if uc.objectStorage == nil {
		return nil, errors.New("オブジェクトストレージが設定されていないため添付ファイルの署名付きURLを生成できません")
	}

	output := &GetAttachmentSignedURLsOutput{SignedURLs: map[model.AttachmentID]string{}}

	attachments, err := uc.attachmentRepo.FindBlobsByIDs(ctx, input.AttachmentIDs)
	if err != nil {
		return nil, fmt.Errorf("添付ファイルの一括取得に失敗: %w", err)
	}
	if len(attachments) == 0 {
		return output, nil
	}

	spaceMembers, err := uc.findSpaceMembers(ctx, attachments, input.UserID)
	if err != nil {
		return nil, err
	}

	publicIDs, err := uc.attachmentRepo.ListPubliclyReferencedIDsByAttachments(ctx, attachments)
	if err != nil {
		return nil, fmt.Errorf("添付ファイルの参照元の公開範囲の取得に失敗: %w", err)
	}
	public := make(map[model.AttachmentID]bool, len(publicIDs))
	for _, id := range publicIDs {
		public[id] = true
	}

	for _, attachment := range attachments {
		// attachment:readはスペースにしか付けられず、判定でトピックを見ないため、トピックメンバーは要らない
		if !newAuthorizer(spaceMembers[attachment.SpaceID], nil).CanShowAttachment(public[attachment.ID]) {
			continue
		}
		url, err := uc.objectStorage.PresignedGetURL(ctx, attachment.BlobKey, model.AttachmentSignedURLExpiration, attachmentSignedURLOptions(attachment))
		if err != nil {
			return nil, fmt.Errorf("添付ファイルの署名付きURLの生成に失敗: %w", err)
		}
		output.SignedURLs[attachment.ID] = url
	}
	return output, nil
}

// findSpaceMembersは、添付ファイルのスペースのうち閲覧者がアクティブなメンバーであるものについて、
// スペースのIDをキーにしたスペースメンバーを返す。ログインしていなければ空のmapを返す
func (uc *GetAttachmentSignedURLsUsecase) findSpaceMembers(ctx context.Context, attachments []*model.Attachment, userID *model.UserID) (map[model.SpaceID]*model.SpaceMember, error) {
	spaceMembers := map[model.SpaceID]*model.SpaceMember{}
	if userID == nil {
		return spaceMembers, nil
	}

	seen := map[model.SpaceID]bool{}
	var spaceIDs []model.SpaceID
	for _, attachment := range attachments {
		if !seen[attachment.SpaceID] {
			seen[attachment.SpaceID] = true
			spaceIDs = append(spaceIDs, attachment.SpaceID)
		}
	}

	members, err := uc.spaceMemberRepo.ListActiveByUserAndSpaceIDs(ctx, *userID, spaceIDs)
	if err != nil {
		return nil, fmt.Errorf("スペースメンバーの一括取得に失敗: %w", err)
	}
	for _, member := range members {
		spaceMembers[member.SpaceID] = member
	}
	return spaceMembers, nil
}
