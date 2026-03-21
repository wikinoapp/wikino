package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// calculateAttachmentRefDiff はbodyHTMLから添付ファイルIDを抽出し、
// 既存の参照との差分を計算して追加・削除すべきIDを返す
func calculateAttachmentRefDiff(
	ctx context.Context,
	bodyHTML string,
	pageID model.PageID,
	spaceID model.SpaceID,
	attachmentRepo *repository.AttachmentRepository,
	pageAttachmentRefRepo *repository.PageAttachmentReferenceRepository,
) (toAdd []model.AttachmentID, toRemove []model.AttachmentID, err error) {
	newIDStrings := markup.ExtractAttachmentIDs(bodyHTML)

	existingRefs, err := pageAttachmentRefRepo.ListByPageID(ctx, pageID, spaceID)
	if err != nil {
		return nil, nil, fmt.Errorf("既存の添付ファイル参照の取得に失敗しました: %w", err)
	}

	existingIDSet := make(map[model.AttachmentID]bool, len(existingRefs))
	for _, ref := range existingRefs {
		existingIDSet[ref.AttachmentID] = true
	}

	newIDSet := make(map[model.AttachmentID]bool, len(newIDStrings))
	for _, idStr := range newIDStrings {
		newIDSet[model.AttachmentID(idStr)] = true
	}

	for id := range newIDSet {
		if !existingIDSet[id] {
			exists, err := attachmentRepo.ExistsByIDAndSpace(ctx, id, spaceID)
			if err != nil {
				return nil, nil, fmt.Errorf("添付ファイルの存在確認に失敗しました: %w", err)
			}
			if exists {
				toAdd = append(toAdd, id)
			}
		}
	}

	for id := range existingIDSet {
		if !newIDSet[id] {
			toRemove = append(toRemove, id)
		}
	}

	return toAdd, toRemove, nil
}

// applyAttachmentRefChanges は事前に計算された添付ファイル参照の追加・削除を実行する
func applyAttachmentRefChanges(
	ctx context.Context,
	pageID model.PageID,
	spaceID model.SpaceID,
	toAdd []model.AttachmentID,
	toRemove []model.AttachmentID,
	pageAttachmentRefRepo *repository.PageAttachmentReferenceRepository,
) error {
	if len(toAdd) > 0 {
		if _, err := pageAttachmentRefRepo.CreateBatch(ctx, pageID, spaceID, toAdd); err != nil {
			return fmt.Errorf("添付ファイル参照の作成に失敗しました: %w", err)
		}
	}

	if len(toRemove) > 0 {
		if err := pageAttachmentRefRepo.DeleteByPageAndAttachmentIDs(ctx, pageID, spaceID, toRemove); err != nil {
			return fmt.Errorf("添付ファイル参照の削除に失敗しました: %w", err)
		}
	}

	return nil
}

// syncAttachmentReferences はbodyHTMLから添付ファイルIDを抽出し、
// 既存の参照との差分を計算して追加・削除を行う
func syncAttachmentReferences(
	ctx context.Context,
	bodyHTML string,
	pageID model.PageID,
	spaceID model.SpaceID,
	attachmentRepo *repository.AttachmentRepository,
	pageAttachmentRefRepo *repository.PageAttachmentReferenceRepository,
) error {
	toAdd, toRemove, err := calculateAttachmentRefDiff(ctx, bodyHTML, pageID, spaceID, attachmentRepo, pageAttachmentRefRepo)
	if err != nil {
		return err
	}

	return applyAttachmentRefChanges(ctx, pageID, spaceID, toAdd, toRemove, pageAttachmentRefRepo)
}

// extractFeaturedImageAttachmentID はbodyの1行目から画像IDを抽出し、
// 添付ファイルの存在を確認した上でAttachmentIDを返す
func extractFeaturedImageAttachmentID(
	ctx context.Context,
	body string,
	spaceID model.SpaceID,
	attachmentRepo *repository.AttachmentRepository,
) (*model.AttachmentID, error) {
	imageIDStr := markup.ExtractFeaturedImageID(body)
	if imageIDStr == nil {
		return nil, nil
	}

	attachmentID := model.AttachmentID(*imageIDStr)
	exists, err := attachmentRepo.ExistsByIDAndSpace(ctx, attachmentID, spaceID)
	if err != nil {
		return nil, fmt.Errorf("アイキャッチ画像の存在確認に失敗しました: %w", err)
	}
	if !exists {
		return nil, nil
	}

	return &attachmentID, nil
}
