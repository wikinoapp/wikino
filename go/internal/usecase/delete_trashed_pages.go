package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// deleteTrashedPagesBatchSizeは、1回のトランザクションで物理削除するページの上限。
// 下書きやリビジョンの多いページが並んでも、1つのトランザクションがロックを長く持たないようにする。
const deleteTrashedPagesBatchSize = 100

// DeleteTrashedPagesUsecaseは、ゴミ箱に入れて保持期間を過ぎたページを、関連レコードとともに
// 物理削除する。定期ジョブとしてWorkerから呼ぶ。
type DeleteTrashedPagesUsecase struct {
	db                          *sql.DB
	pageRepo                    *repository.PageRepository
	draftPageRepo               *repository.DraftPageRepository
	pageEditorRepo              *repository.PageEditorRepository
	pageRevisionRepo            *repository.PageRevisionRepository
	pageAttachmentReferenceRepo *repository.PageAttachmentReferenceRepository
}

// NewDeleteTrashedPagesUsecaseはDeleteTrashedPagesUsecaseを生成する。
func NewDeleteTrashedPagesUsecase(
	db *sql.DB,
	pageRepo *repository.PageRepository,
	draftPageRepo *repository.DraftPageRepository,
	pageEditorRepo *repository.PageEditorRepository,
	pageRevisionRepo *repository.PageRevisionRepository,
	pageAttachmentReferenceRepo *repository.PageAttachmentReferenceRepository,
) *DeleteTrashedPagesUsecase {
	return &DeleteTrashedPagesUsecase{
		db:                          db,
		pageRepo:                    pageRepo,
		draftPageRepo:               draftPageRepo,
		pageEditorRepo:              pageEditorRepo,
		pageRevisionRepo:            pageRevisionRepo,
		pageAttachmentReferenceRepo: pageAttachmentReferenceRepo,
	}
}

// DeleteTrashedPagesInputは物理削除の入力。どちらもゼロ値なら既定の値を使う。
type DeleteTrashedPagesInput struct {
	// Nowは、保持期間を過ぎたかを判定する基準の現在時刻
	Now time.Time
	// BatchSizeは1回のトランザクションで物理削除するページの上限
	BatchSize int32
}

// Executeは、ゴミ箱に入れて保持期間 (model.PageTrashRetentionDays) を過ぎたページを、スペースごとに
// 上限の件数ずつ物理削除する。消す範囲はRails版の `Pages::BulkDestroyTrashedService` と同じで、
// 保持期間はゴミ箱の一覧に出す「復元できるページ」と同じ定数から決める。
func (uc *DeleteTrashedPagesUsecase) Execute(ctx context.Context, input DeleteTrashedPagesInput) error {
	now := input.Now
	if now.IsZero() {
		now = time.Now()
	}
	batchSize := input.BatchSize
	if batchSize <= 0 {
		batchSize = deleteTrashedPagesBatchSize
	}
	trashedBefore := now.AddDate(0, 0, -model.PageTrashRetentionDays)

	spaceIDs, err := uc.pageRepo.ListSpaceIDsWithTrashedBefore(ctx, trashedBefore)
	if err != nil {
		return fmt.Errorf("物理削除するページがあるスペースの取得に失敗: %w", err)
	}

	var deletedCount int
	for _, spaceID := range spaceIDs {
		for {
			deleted, err := uc.deleteBatch(ctx, spaceID, trashedBefore, batchSize)
			if err != nil {
				return err
			}
			deletedCount += deleted
			if deleted < int(batchSize) {
				break
			}
		}
	}

	slog.InfoContext(ctx, "ゴミ箱の保持期間を過ぎたページを物理削除しました",
		"space_count", len(spaceIDs),
		"deleted_count", deletedCount,
	)
	return nil
}

// deleteBatchは、スペースの保持期間を過ぎたページを最大batchSize件、関連レコードとともに1つの
// トランザクションで物理削除し、消した件数を返す。
//
// ページの行を関連レコードより先にロックしてから消す。ロックを待つ間にゴミ箱から戻されたページは
// 対象から外れるため、戻したばかりのページを消すことはない。
func (uc *DeleteTrashedPagesUsecase) deleteBatch(ctx context.Context, spaceID model.SpaceID, trashedBefore time.Time, batchSize int32) (int, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("トランザクションの開始に失敗: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	pageRepo := uc.pageRepo.WithTx(tx)
	pageIDs, err := pageRepo.ListTrashedIDsBeforeForUpdate(ctx, spaceID, trashedBefore, batchSize)
	if err != nil {
		return 0, fmt.Errorf("物理削除するページの取得に失敗: %w", err)
	}

	if err := deletePagesWithRelatedRecords(ctx, pageDeletionRepos{
		pageRepo:                    pageRepo,
		draftPageRepo:               uc.draftPageRepo.WithTx(tx),
		pageEditorRepo:              uc.pageEditorRepo.WithTx(tx),
		pageRevisionRepo:            uc.pageRevisionRepo.WithTx(tx),
		pageAttachmentReferenceRepo: uc.pageAttachmentReferenceRepo.WithTx(tx),
	}, spaceID, pageIDs); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("トランザクションのコミットに失敗: %w", err)
	}
	return len(pageIDs), nil
}
