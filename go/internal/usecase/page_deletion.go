package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// pageDeletionReposは、ページとその関連レコードを物理削除するために使うリポジトリ。
// 呼び出し側が開いたトランザクションに参加させたもの (WithTxで得たもの) を渡す。
type pageDeletionRepos struct {
	pageRepo                    *repository.PageRepository
	draftPageRepo               *repository.DraftPageRepository
	pageEditorRepo              *repository.PageEditorRepository
	pageRevisionRepo            *repository.PageRevisionRepository
	pageAttachmentReferenceRepo *repository.PageAttachmentReferenceRepository
}

// deletePagesWithRelatedRecordsは、ページの下書き・編集者・リビジョン・添付ファイル参照を消してから
// ページを物理削除する (Rails版の `PageRecord.destroy_all_with_related_records!` と同じ範囲)。
// 下書きのリビジョンと編集提案のページは外部キーのON DELETE CASCADEで消える。添付ファイル自体は
// スペースに属するため削除しない。
//
// ページの関連レコードだけが消えた状態を残さないよう、呼び出し側は1つのトランザクションの中で呼ぶ。
func deletePagesWithRelatedRecords(ctx context.Context, repos pageDeletionRepos, spaceID model.SpaceID, pageIDs []model.PageID) error {
	if len(pageIDs) == 0 {
		return nil
	}

	if err := repos.draftPageRepo.DeleteByPageIDs(ctx, pageIDs, spaceID); err != nil {
		return fmt.Errorf("ページの下書きの削除に失敗: %w", err)
	}
	if err := repos.pageEditorRepo.DeleteByPageIDs(ctx, pageIDs, spaceID); err != nil {
		return fmt.Errorf("ページの編集者の削除に失敗: %w", err)
	}
	if err := repos.pageRevisionRepo.DeleteByPageIDs(ctx, pageIDs, spaceID); err != nil {
		return fmt.Errorf("ページのリビジョンの削除に失敗: %w", err)
	}
	if err := repos.pageAttachmentReferenceRepo.DeleteByPageIDs(ctx, pageIDs, spaceID); err != nil {
		return fmt.Errorf("ページの添付ファイル参照の削除に失敗: %w", err)
	}
	if _, err := repos.pageRepo.DeleteByIDs(ctx, pageIDs, spaceID); err != nil {
		return fmt.Errorf("ページの削除に失敗: %w", err)
	}

	return nil
}
