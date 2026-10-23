package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// deleteDraftPageWithRevisionsは下書きを編集履歴 (リビジョン) ごと削除する。
// リポジトリは呼び出し元のトランザクションに属するもの (WithTx) を受け取る。
// Web画面の削除・公開・APIの削除のどの経路でも下書き → 編集履歴の順にロックし、同じ下書きを同時に
// 削除するトランザクションどうしが循環待ちにならないよう、先に下書きの行をロックする。
// ロックした時点で下書きが既に削除されていればfalseを返す。
// beforeDeleteがnilでなければ、ロックした下書きを渡して呼び、エラーを返せば削除せずにそのエラーを返す
func deleteDraftPageWithRevisions(
	ctx context.Context,
	draftPageRepo *repository.DraftPageRepository,
	draftPageRevisionRepo *repository.DraftPageRevisionRepository,
	draftPageID model.DraftPageID,
	spaceID model.SpaceID,
	beforeDelete func(locked *model.DraftPage) error,
) (bool, error) {
	locked, err := draftPageRepo.FindByIDForUpdate(ctx, draftPageID, spaceID)
	if err != nil {
		return false, fmt.Errorf("下書きのロックに失敗しました: %w", err)
	}
	if locked == nil {
		return false, nil
	}
	if beforeDelete != nil {
		if err := beforeDelete(locked); err != nil {
			return false, err
		}
	}

	// draft_page_revisionsのFK制約があるためリビジョンを先に削除する
	if err := draftPageRevisionRepo.DeleteByDraftPageID(ctx, draftPageID, spaceID); err != nil {
		return false, fmt.Errorf("下書きリビジョンの削除に失敗しました: %w", err)
	}

	if err := draftPageRepo.Delete(ctx, draftPageID, spaceID); err != nil {
		return false, fmt.Errorf("下書きの削除に失敗しました: %w", err)
	}
	return true, nil
}
