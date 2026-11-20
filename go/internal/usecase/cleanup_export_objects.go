package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
)

// CleanupExportObjectsUsecaseは、行を消したエクスポートのZIPを回収する。
// キーをジョブに保持するため、blobの確保後にストレージが失敗しても再試行できる。
type CleanupExportObjectsUsecase struct {
	exportRepo    *repository.ExportRepository
	dispatcher    *dispatcher.Dispatcher
	objectStorage storage.ObjectStorage
}

// NewCleanupExportObjectsUsecaseはCleanupExportObjectsUsecaseを生成する。
func NewCleanupExportObjectsUsecase(repo *repository.ExportRepository, d *dispatcher.Dispatcher, objectStorage storage.ObjectStorage) *CleanupExportObjectsUsecase {
	return &CleanupExportObjectsUsecase{exportRepo: repo, dispatcher: d, objectStorage: objectStorage}
}

// CleanupExportObjectsInputは、削除した行から受け取ったZIPのキーと共有blobの候補。
type CleanupExportObjectsInput struct {
	SpaceID     string
	ExportIDs   []string
	Keys        []string
	LegacyBlobs []model.LegacyExportBlob
}

// Executeは、生成ジョブの終了を待ってからZIPを回収する。正の待機時間は、試行回数を消費せず先送りする指示。
func (uc *CleanupExportObjectsUsecase) Execute(ctx context.Context, input CleanupExportObjectsInput) (time.Duration, error) {
	active, err := uc.dispatcher.HasActiveExportGeneration(ctx, input.SpaceID, input.ExportIDs)
	if err != nil {
		return 0, fmt.Errorf("エクスポートの生成ジョブの確認に失敗: %w", err)
	}
	if active {
		return 10 * time.Second, nil
	}
	legacyKeys, err := uc.exportRepo.ClaimUnreferencedLegacyBlobs(ctx, input.LegacyBlobs)
	if err != nil {
		return 0, fmt.Errorf("共有参照のないZIPのblobの確保に失敗: %w", err)
	}
	keys := append(legacyKeys, input.Keys...)
	if err := uc.objectStorage.DeleteMany(ctx, keys); err != nil {
		return 0, fmt.Errorf("エクスポートの実体の後始末に失敗: %w", err)
	}
	return 0, nil
}
