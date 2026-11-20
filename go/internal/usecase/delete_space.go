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

// spaceAttachmentBatchSizeは、スペースの物理削除が添付ファイルを一度に読む件数。オブジェクトストレージの
// 一括削除が1回のリクエストで消せるキーの数 (1000) に揃え、1件の添付ファイルが元ファイルとサムネイルの
// 2つのキーを持っても、1回の読み出しにつき数回のリクエストで済むようにする
const spaceAttachmentBatchSize int32 = 1000

// errSpaceObjectStorageNotConfiguredは、消す実体があるのにオブジェクトストレージが設定されていないことを表す。
// 行だけを消すと、誰にも見えない実体がバケットに残るため、行も消さずに失敗させる
var errSpaceObjectStorageNotConfigured = errors.New("オブジェクトストレージが設定されていないためスペースの実体を削除できません")

// DeleteSpaceUsecaseは、論理削除したスペースを、トピック・ページ・添付ファイル・エクスポート・メンバーとともに
// 物理削除する。スペースの削除の画面から投入したジョブがWorkerから呼ぶ。
type DeleteSpaceUsecase struct {
	db                          *sql.DB
	spaceRepo                   *repository.SpaceRepository
	spaceMemberRepo             *repository.SpaceMemberRepository
	topicRepo                   *repository.TopicRepository
	topicMemberRepo             *repository.TopicMemberRepository
	pageRepo                    *repository.PageRepository
	draftPageRepo               *repository.DraftPageRepository
	pageEditorRepo              *repository.PageEditorRepository
	pageRevisionRepo            *repository.PageRevisionRepository
	pageAttachmentReferenceRepo *repository.PageAttachmentReferenceRepository
	attachmentRepo              *repository.AttachmentRepository
	exportRepo                  *repository.ExportRepository
	objectStorage               storage.ObjectStorage
	attachmentDeleter           *attachmentDeleter
	dispatcher                  *dispatcher.Dispatcher
}

// DeleteSpaceReposは、スペースの物理削除が使うリポジトリ。数が多いため、コンストラクタの引数を1つにまとめる
type DeleteSpaceRepos struct {
	SpaceRepo                   *repository.SpaceRepository
	SpaceMemberRepo             *repository.SpaceMemberRepository
	TopicRepo                   *repository.TopicRepository
	TopicMemberRepo             *repository.TopicMemberRepository
	PageRepo                    *repository.PageRepository
	DraftPageRepo               *repository.DraftPageRepository
	PageEditorRepo              *repository.PageEditorRepository
	PageRevisionRepo            *repository.PageRevisionRepository
	PageAttachmentReferenceRepo *repository.PageAttachmentReferenceRepository
	AttachmentRepo              *repository.AttachmentRepository
	ExportRepo                  *repository.ExportRepository
}

// NewDeleteSpaceUsecaseはDeleteSpaceUsecaseを生成する。objectStorageはnilでもよく、そのときは実体を持つ
// スペースの削除がエラーになる。dispatcherは、添付ファイルとZIPの後始末ジョブを行の削除と同じトランザクションで
// 登録するのに使う
func NewDeleteSpaceUsecase(db *sql.DB, repos DeleteSpaceRepos, objectStorage storage.ObjectStorage, d *dispatcher.Dispatcher) *DeleteSpaceUsecase {
	return &DeleteSpaceUsecase{
		db:                          db,
		spaceRepo:                   repos.SpaceRepo,
		spaceMemberRepo:             repos.SpaceMemberRepo,
		topicRepo:                   repos.TopicRepo,
		topicMemberRepo:             repos.TopicMemberRepo,
		pageRepo:                    repos.PageRepo,
		draftPageRepo:               repos.DraftPageRepo,
		pageEditorRepo:              repos.PageEditorRepo,
		pageRevisionRepo:            repos.PageRevisionRepo,
		pageAttachmentReferenceRepo: repos.PageAttachmentReferenceRepo,
		attachmentRepo:              repos.AttachmentRepo,
		exportRepo:                  repos.ExportRepo,
		objectStorage:               objectStorage,
		attachmentDeleter:           &attachmentDeleter{db: db, attachmentRepo: repos.AttachmentRepo, objectStorage: objectStorage, dispatcher: d},
		dispatcher:                  d,
	}
}

// DeleteSpaceInputは物理削除するスペースを指す。
type DeleteSpaceInput struct {
	SpaceID model.SpaceID
}

// Executeは、論理削除したスペースの実体を消してから、行を1つのトランザクションで消す。
//
// 論理削除していないスペースと、既に物理削除したスペースには何もしない。実体の削除に失敗したら行を残し、
// 再試行で最初から消し直す。行の削除が途中で失敗しても、1つも消さずにロールバックする。
// 行を消したあとの書き戻しと共有状態の変化は、同じトランザクションで登録する後始末ジョブが回収する。
func (uc *DeleteSpaceUsecase) Execute(ctx context.Context, input DeleteSpaceInput) error {
	discarded, err := uc.spaceRepo.ExistsDiscarded(ctx, input.SpaceID)
	if err != nil {
		return fmt.Errorf("論理削除したスペースの確認に失敗: %w", err)
	}
	if !discarded {
		return nil
	}

	if err := uc.deleteObjects(ctx, input.SpaceID); err != nil {
		return err
	}
	return uc.deleteRows(ctx, input.SpaceID)
}

// deleteObjectsは、スペースのエクスポートのアーカイブと、添付ファイルとそのサムネイル (variant) の実体を
// トランザクションの外で消す。どれも一定の件数ずつ読み、読んだ分をまとめて消す。ほかのスペースの
// エクスポートも参照しているRails期のZIPは、参照が無くなるまで残すため、ここでは消さずに後始末へ回す
func (uc *DeleteSpaceUsecase) deleteObjects(ctx context.Context, spaceID model.SpaceID) error {
	if err := eachSpaceExportBatch(ctx, uc.exportRepo, spaceID, func(exports []*model.Export) error {
		_, keys := exportCleanupTargets(exports)
		return uc.deleteObjectKeys(ctx, keys)
	}); err != nil {
		return fmt.Errorf("エクスポートの実体の削除に失敗: %w", err)
	}
	if err := eachSpaceLegacyExportBlobBatch(ctx, uc.exportRepo, spaceID, func(blobs []model.LegacyExportBlob) error {
		var keys []string
		for _, blob := range blobs {
			if !blob.Shared {
				keys = append(keys, blob.Key)
			}
		}
		return uc.deleteObjectKeys(ctx, keys)
	}); err != nil {
		return fmt.Errorf("エクスポートの実体の削除に失敗: %w", err)
	}

	return eachSpaceAttachmentBatch(ctx, uc.attachmentRepo, spaceID, func(ids []model.AttachmentID) error {
		keys, err := uc.attachmentRepo.ListObjectKeysByIDsAndSpace(ctx, ids, spaceID)
		if err != nil {
			return fmt.Errorf("添付ファイルの実体のキーの取得に失敗: %w", err)
		}
		if err := uc.deleteObjectKeys(ctx, keys); err != nil {
			return fmt.Errorf("添付ファイルの実体の削除に失敗: %w", err)
		}
		return nil
	})
}

// deleteObjectKeysは、keysの実体をまとめて消す。消すキーが無ければ、オブジェクトストレージが無くても成功する
func (uc *DeleteSpaceUsecase) deleteObjectKeys(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	if uc.objectStorage == nil {
		return errSpaceObjectStorageNotConfigured
	}
	return uc.objectStorage.DeleteMany(ctx, keys)
}

// eachSpaceAttachmentBatchは、スペースの添付ファイルのIDをspaceAttachmentBatchSizeずつ読み、fnに渡す
func eachSpaceAttachmentBatch(ctx context.Context, attachmentRepo *repository.AttachmentRepository, spaceID model.SpaceID, fn func(ids []model.AttachmentID) error) error {
	var after model.AttachmentID
	for {
		ids, err := attachmentRepo.ListIDsBySpaceAfter(ctx, spaceID, after, spaceAttachmentBatchSize)
		if err != nil {
			return fmt.Errorf("スペースの添付ファイルの取得に失敗: %w", err)
		}
		if len(ids) == 0 {
			return nil
		}
		if err := fn(ids); err != nil {
			return err
		}
		if len(ids) < int(spaceAttachmentBatchSize) {
			return nil
		}
		after = ids[len(ids)-1]
	}
}

// deleteRowsは、論理削除したスペースをロックしてから、スペースに属する行を1つのトランザクションで消す。
// 外部キーで参照される側を後に消す順にする
func (uc *DeleteSpaceUsecase) deleteRows(ctx context.Context, spaceID model.SpaceID) error {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("トランザクションの開始に失敗: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	spaceRepo := uc.spaceRepo.WithTx(tx)
	pageRepo := uc.pageRepo.WithTx(tx)
	draftPageRepo := uc.draftPageRepo.WithTx(tx)

	// 論理削除したスペースをロックしてから、そのページを読む。ロックの前に読むと、その間に追加された
	// ページを消し漏らす。ロックはスペースを参照する行の追加も、コミットまで待たせる
	locked, err := spaceRepo.LockDiscarded(ctx, spaceID)
	if err != nil {
		return fmt.Errorf("論理削除したスペースのロックに失敗: %w", err)
	}
	if !locked {
		return nil
	}
	if err := uc.checkObjectStorageAfterLock(ctx, tx, spaceID); err != nil {
		return err
	}

	pageIDs, err := pageRepo.ListIDsBySpaceForUpdate(ctx, spaceID)
	if err != nil {
		return fmt.Errorf("スペースのページの取得に失敗: %w", err)
	}
	if err := deletePagesWithRelatedRecords(ctx, pageDeletionRepos{
		pageRepo:                    pageRepo,
		draftPageRepo:               draftPageRepo,
		pageEditorRepo:              uc.pageEditorRepo.WithTx(tx),
		pageRevisionRepo:            uc.pageRevisionRepo.WithTx(tx),
		pageAttachmentReferenceRepo: uc.pageAttachmentReferenceRepo.WithTx(tx),
	}, spaceID, pageIDs); err != nil {
		return err
	}

	if err := draftPageRepo.DeleteBySpace(ctx, spaceID); err != nil {
		return fmt.Errorf("スペースの下書きの削除に失敗: %w", err)
	}
	if err := uc.topicMemberRepo.WithTx(tx).DeleteBySpace(ctx, spaceID); err != nil {
		return fmt.Errorf("スペースのトピックのメンバーの削除に失敗: %w", err)
	}
	// 編集提案は外部キーのON DELETE CASCADEでトピックとともに消える。
	if err := uc.topicRepo.WithTx(tx).DeleteBySpace(ctx, spaceID); err != nil {
		return fmt.Errorf("スペースのトピックの削除に失敗: %w", err)
	}

	// 添付ファイルは、管理の画面からの削除と同じ範囲の行を消し、実際に削除したキーを持つ後始末ジョブを
	// 同じトランザクションで登録する。後始末ジョブは、スペースの行が無くなったあとも保存したキーで動く
	if err := eachSpaceAttachmentBatch(ctx, uc.attachmentRepo.WithTx(tx), spaceID, func(ids []model.AttachmentID) error {
		return uc.attachmentDeleter.deleteRowsTx(ctx, tx, spaceID, ids)
	}); err != nil {
		return err
	}

	exportRepo := uc.exportRepo.WithTx(tx)
	if err := eachSpaceExportBatch(ctx, exportRepo, spaceID, func(exports []*model.Export) error {
		exportIDs, keys := exportCleanupTargets(exports)
		return uc.dispatcher.EnqueueCleanupExportObjectsTx(ctx, tx, string(spaceID), exportIDs, keys)
	}); err != nil {
		return fmt.Errorf("エクスポートの後始末ジョブの登録に失敗: %w", err)
	}
	if err := eachSpaceLegacyExportBlobBatch(ctx, exportRepo, spaceID, func(blobs []model.LegacyExportBlob) error {
		blobIDs, blobKeys := make([]string, len(blobs)), make([]string, len(blobs))
		for i, blob := range blobs {
			blobIDs[i], blobKeys[i] = blob.ID, blob.Key
		}
		return uc.dispatcher.EnqueueCleanupLegacyExportBlobsTx(ctx, tx, string(spaceID), blobIDs, blobKeys)
	}); err != nil {
		return fmt.Errorf("共有ZIPの後始末ジョブの登録に失敗: %w", err)
	}
	if err := exportRepo.DeleteLegacyFilesBySpace(ctx, spaceID); err != nil {
		return fmt.Errorf("エクスポートのRails版のZIPの行の削除に失敗: %w", err)
	}
	if err := uc.spaceMemberRepo.WithTx(tx).DeleteBySpace(ctx, spaceID); err != nil {
		return fmt.Errorf("スペースメンバーの削除に失敗: %w", err)
	}
	// エクスポート・編集提案・個人用アクセストークン・OAuthの行は、外部キーのON DELETE CASCADEで
	// スペースとともに消える。
	if _, err := spaceRepo.DeleteDiscarded(ctx, spaceID); err != nil {
		return fmt.Errorf("スペースの削除に失敗: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("トランザクションのコミットに失敗: %w", err)
	}
	return nil
}

// checkObjectStorageAfterLockは、最初のキーの取得後に追加された行を見落として行だけを消すことを防ぐ。
// スペースのロックは新しい添付ファイルとエクスポートの追加を待たせる。ここではDBの確認だけを行う。
func (uc *DeleteSpaceUsecase) checkObjectStorageAfterLock(ctx context.Context, tx *sql.Tx, spaceID model.SpaceID) error {
	if uc.objectStorage != nil {
		return nil
	}
	ids, err := uc.attachmentRepo.WithTx(tx).ListIDsBySpaceAfter(ctx, spaceID, "", 1)
	if err != nil {
		return fmt.Errorf("削除する添付ファイルの確認に失敗: %w", err)
	}
	exports, err := uc.exportRepo.WithTx(tx).ListForDeletionBySpaceAfter(ctx, spaceID, "", 1)
	if err != nil {
		return fmt.Errorf("削除するエクスポートの確認に失敗: %w", err)
	}
	if len(ids) > 0 || len(exports) > 0 {
		return errSpaceObjectStorageNotConfigured
	}
	return nil
}
