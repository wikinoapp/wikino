package usecase

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
)

// errAttachmentObjectStorageNotConfiguredは、消す実体があるのにオブジェクトストレージが設定されていない
// ことを表す。行だけを消すと、誰にも見えない実体がバケットに残るため、行も消さずに失敗させる
var errAttachmentObjectStorageNotConfigured = errors.New("オブジェクトストレージが設定されていないため添付ファイルの実体を削除できません")

// attachmentDeleterは、添付ファイルの実体と行を消す。添付ファイルの管理の画面からの削除と、スペースの
// 物理削除が使い、実体と行を消す範囲と順序を1か所で決める。
//
// 実体の最初の削除に失敗したら行を残す。行の削除と、実際に削除したキーを持つ後始末ジョブの登録は、
// 同じトランザクションで行う。書き戻された実体や、最初の削除のあとに増えたvariantも回収できる。
type attachmentDeleter struct {
	db             *sql.DB
	attachmentRepo *repository.AttachmentRepository
	objectStorage  storage.ObjectStorage
	dispatcher     *dispatcher.Dispatcher
}

// deleteは、スペースに属する指定した添付ファイルの実体と行を消す。ほかのスペースの添付ファイルと存在しない
// IDは何もしないため、同じIDで何度呼んでも結果は変わらない。
func (d *attachmentDeleter) delete(ctx context.Context, spaceID model.SpaceID, ids []model.AttachmentID) error {
	keys, err := d.attachmentRepo.ListObjectKeysByIDsAndSpace(ctx, ids, spaceID)
	if err != nil {
		return fmt.Errorf("添付ファイルの実体のキーの取得に失敗: %w", err)
	}
	if len(keys) > 0 && d.objectStorage == nil {
		return errAttachmentObjectStorageNotConfigured
	}

	if err := d.deleteObjects(ctx, keys); err != nil {
		return err
	}

	return d.deleteRows(ctx, spaceID, ids)
}

// deleteObjectsは、実体を1つずつ消す。オブジェクトストレージは無いキーの削除を成功として扱う
func (d *attachmentDeleter) deleteObjects(ctx context.Context, keys []string) error {
	for _, key := range keys {
		if err := d.objectStorage.Delete(ctx, key); err != nil {
			return fmt.Errorf("添付ファイルの実体の削除に失敗: %w", err)
		}
	}
	return nil
}

// deleteRowsは、添付ファイルの行を消し、後始末ジョブを同じトランザクションで登録する。
func (d *attachmentDeleter) deleteRows(ctx context.Context, spaceID model.SpaceID, ids []model.AttachmentID) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("トランザクションの開始に失敗: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := d.deleteRowsTx(ctx, tx, spaceID, ids); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("トランザクションのコミットに失敗: %w", err)
	}
	return nil
}

// deleteRowsTxは、呼び出し側のトランザクションで添付ファイルの行を消し、実際に削除したキーを持つ後始末ジョブを
// 同じトランザクションで登録する。スペースの物理削除は、スペースのほかの行と同じトランザクションでこれを呼ぶ
func (d *attachmentDeleter) deleteRowsTx(ctx context.Context, tx *sql.Tx, spaceID model.SpaceID, ids []model.AttachmentID) error {
	keys, err := d.attachmentRepo.WithTx(tx).DeleteByIDsAndSpace(ctx, ids, spaceID)
	if err != nil {
		return fmt.Errorf("添付ファイルの行の削除に失敗: %w", err)
	}
	if len(keys) == 0 {
		return nil
	}
	attachmentIDs := make([]string, len(ids))
	for i, id := range ids {
		attachmentIDs[i] = string(id)
	}
	if err := d.dispatcher.EnqueueCleanupAttachmentObjectsTx(ctx, tx, string(spaceID), attachmentIDs, keys); err != nil {
		return fmt.Errorf("添付ファイルの後始末ジョブの投入に失敗: %w", err)
	}
	return nil
}
