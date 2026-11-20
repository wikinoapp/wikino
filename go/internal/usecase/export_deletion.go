package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// spaceExportBatchSizeは、ZIPのキーの読み出しと後始末ジョブ1件に含めるレコードの上限。
const spaceExportBatchSize int32 = 1000

func eachSpaceExportBatch(ctx context.Context, repo *repository.ExportRepository, spaceID model.SpaceID, fn func([]*model.Export) error) error {
	var after model.ExportID
	for {
		exports, err := repo.ListForDeletionBySpaceAfter(ctx, spaceID, after, spaceExportBatchSize)
		if err != nil {
			return fmt.Errorf("削除するエクスポートの取得に失敗: %w", err)
		}
		if len(exports) == 0 {
			return nil
		}
		if err := fn(exports); err != nil {
			return err
		}
		if len(exports) < int(spaceExportBatchSize) {
			return nil
		}
		after = exports[len(exports)-1].ID
	}
}

func eachSpaceLegacyExportBlobBatch(ctx context.Context, repo *repository.ExportRepository, spaceID model.SpaceID, fn func([]model.LegacyExportBlob) error) error {
	var after string
	for {
		blobs, err := repo.ListLegacyBlobsBySpaceAfter(ctx, spaceID, after, spaceExportBatchSize)
		if err != nil {
			return fmt.Errorf("削除するZIPのblobの取得に失敗: %w", err)
		}
		if len(blobs) == 0 {
			return nil
		}
		if err := fn(blobs); err != nil {
			return err
		}
		if len(blobs) < int(spaceExportBatchSize) {
			return nil
		}
		after = blobs[len(blobs)-1].ID
	}
}

// exportCleanupTargetsは、エクスポートのIDと、記録されたキーと、生成が成功を記録する前にも書き込める
// 保存先を返す。同じキーは一度だけ返し、ジョブ引数の大きさを読み出したバッチの件数で制限する。
func exportCleanupTargets(exports []*model.Export) (exportIDs, keys []string) {
	seen := map[string]bool{}
	for _, export := range exports {
		exportIDs = append(exportIDs, string(export.ID))
		candidates := []string{exportObjectKey(export)}
		if export.ObjectKey != nil {
			candidates = append(candidates, *export.ObjectKey)
		}
		for _, key := range candidates {
			if !seen[key] {
				seen[key] = true
				keys = append(keys, key)
			}
		}
	}
	return exportIDs, keys
}
