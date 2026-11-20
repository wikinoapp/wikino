package usecase

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// DeleteTopicUsecaseは、論理削除したトピックを関連レコードとともに物理削除する。
// トピックの削除の画面から投入したジョブがWorkerから呼ぶ。
type DeleteTopicUsecase struct {
	db                          *sql.DB
	topicRepo                   *repository.TopicRepository
	topicMemberRepo             *repository.TopicMemberRepository
	pageRepo                    *repository.PageRepository
	draftPageRepo               *repository.DraftPageRepository
	pageEditorRepo              *repository.PageEditorRepository
	pageRevisionRepo            *repository.PageRevisionRepository
	pageAttachmentReferenceRepo *repository.PageAttachmentReferenceRepository
}

// NewDeleteTopicUsecaseはDeleteTopicUsecaseを生成する。
func NewDeleteTopicUsecase(
	db *sql.DB,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	pageRepo *repository.PageRepository,
	draftPageRepo *repository.DraftPageRepository,
	pageEditorRepo *repository.PageEditorRepository,
	pageRevisionRepo *repository.PageRevisionRepository,
	pageAttachmentReferenceRepo *repository.PageAttachmentReferenceRepository,
) *DeleteTopicUsecase {
	return &DeleteTopicUsecase{
		db:                          db,
		topicRepo:                   topicRepo,
		topicMemberRepo:             topicMemberRepo,
		pageRepo:                    pageRepo,
		draftPageRepo:               draftPageRepo,
		pageEditorRepo:              pageEditorRepo,
		pageRevisionRepo:            pageRevisionRepo,
		pageAttachmentReferenceRepo: pageAttachmentReferenceRepo,
	}
}

// DeleteTopicInputは物理削除するトピックを指す。
type DeleteTopicInput struct {
	SpaceID model.SpaceID
	TopicID model.TopicID
}

// Executeは、論理削除したトピックを1つのトランザクションで物理削除する (Rails版の
// `Topics::DestroyService` と同じ範囲)。
//
// 論理削除していないトピックと、既に物理削除したトピックには何もしない。物理削除は論理削除した
// トピックだけを対象にすると決めており、再試行で同じジョブが2度動いても結果が変わらないようにする。
func (uc *DeleteTopicUsecase) Execute(ctx context.Context, input DeleteTopicInput) error {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("トランザクションの開始に失敗: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	topicRepo := uc.topicRepo.WithTx(tx)
	pageRepo := uc.pageRepo.WithTx(tx)
	draftPageRepo := uc.draftPageRepo.WithTx(tx)

	// 論理削除したトピックをロックしてから、そのページを読む。ロックの前に読むと、その間に
	// 論理削除を取り消されたトピックのページまで消してしまう。
	locked, err := topicRepo.LockDiscarded(ctx, input.TopicID, input.SpaceID)
	if err != nil {
		return fmt.Errorf("論理削除したトピックのロックに失敗: %w", err)
	}
	if !locked {
		return nil
	}

	// ページの行を関連レコードより先にロックし、移動のコミット後もこのトピックに属するものだけを消す。
	pageIDs, err := pageRepo.ListIDsByTopicForUpdate(ctx, input.TopicID, input.SpaceID)
	if err != nil {
		return fmt.Errorf("トピックのページの取得に失敗: %w", err)
	}

	if err := deletePagesWithRelatedRecords(ctx, pageDeletionRepos{
		pageRepo:                    pageRepo,
		draftPageRepo:               draftPageRepo,
		pageEditorRepo:              uc.pageEditorRepo.WithTx(tx),
		pageRevisionRepo:            uc.pageRevisionRepo.WithTx(tx),
		pageAttachmentReferenceRepo: uc.pageAttachmentReferenceRepo.WithTx(tx),
	}, input.SpaceID, pageIDs); err != nil {
		return err
	}

	// 別トピックのページに属する下書きは保持する。下書きの所属だけが古い場合は、本文と履歴を
	// 保ったままページの現在の所属にそろえ、削除するトピックへの外部キー参照を外す。
	if err := draftPageRepo.SyncTopicsFromPages(ctx, input.TopicID, input.SpaceID); err != nil {
		return fmt.Errorf("下書きのトピック参照の修復に失敗: %w", err)
	}
	if err := uc.topicMemberRepo.WithTx(tx).DeleteByTopic(ctx, input.TopicID, input.SpaceID); err != nil {
		return fmt.Errorf("トピックのメンバーの削除に失敗: %w", err)
	}
	// 編集提案は外部キーのON DELETE CASCADEでトピックとともに消える。
	if _, err := topicRepo.DeleteDiscarded(ctx, input.TopicID, input.SpaceID); err != nil {
		return fmt.Errorf("トピックの削除に失敗: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("トランザクションのコミットに失敗: %w", err)
	}

	return nil
}
