package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// PublishPageUsecase はページ公開ユースケース
type PublishPageUsecase struct {
	db                    *sql.DB
	pageRepo              *repository.PageRepository
	pageRevisionRepo      *repository.PageRevisionRepository
	pageEditorRepo        *repository.PageEditorRepository
	draftPageRepo         *repository.DraftPageRepository
	draftPageRevisionRepo *repository.DraftPageRevisionRepository
	topicRepo             *repository.TopicRepository
	topicMemberRepo       *repository.TopicMemberRepository
	pageAttachmentRefRepo *repository.PageAttachmentReferenceRepository
}

// NewPublishPageUsecase は PublishPageUsecase を生成する
func NewPublishPageUsecase(
	db *sql.DB,
	pageRepo *repository.PageRepository,
	pageRevisionRepo *repository.PageRevisionRepository,
	pageEditorRepo *repository.PageEditorRepository,
	draftPageRepo *repository.DraftPageRepository,
	draftPageRevisionRepo *repository.DraftPageRevisionRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	pageAttachmentRefRepo *repository.PageAttachmentReferenceRepository,
) *PublishPageUsecase {
	return &PublishPageUsecase{
		db:                    db,
		pageRepo:              pageRepo,
		pageRevisionRepo:      pageRevisionRepo,
		pageEditorRepo:        pageEditorRepo,
		draftPageRepo:         draftPageRepo,
		draftPageRevisionRepo: draftPageRevisionRepo,
		topicRepo:             topicRepo,
		topicMemberRepo:       topicMemberRepo,
		pageAttachmentRefRepo: pageAttachmentRefRepo,
	}
}

// PublishPageInput はページ公開の入力パラメータ
type PublishPageInput struct {
	SpaceID                      model.SpaceID
	PageID                       model.PageID
	SpaceMemberID                model.SpaceMemberID
	TopicID                      model.TopicID
	DraftPageID                  model.DraftPageID
	Title                        *string
	Body                         string
	BodyHTML                     string
	SpaceIdentifier              model.SpaceIdentifier
	CurrentTopicName             string
	UnpublishedConflictingPageID *model.PageID
	FeaturedImageAttachmentID    *model.AttachmentID
	AttachmentRefsToAdd          []model.AttachmentID
	AttachmentRefsToRemove       []model.AttachmentID
}

// PublishPageOutput はページ公開の出力パラメータ
type PublishPageOutput struct {
	Page        *model.Page
	PublishedAt time.Time
}

// Execute はページを公開する
func (uc *PublishPageUsecase) Execute(ctx context.Context, input PublishPageInput) (*PublishPageOutput, error) {
	now := time.Now()

	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	pageRepo := uc.pageRepo.WithTx(tx)
	pageRevisionRepo := uc.pageRevisionRepo.WithTx(tx)
	pageEditorRepo := uc.pageEditorRepo.WithTx(tx)
	draftPageRepo := uc.draftPageRepo.WithTx(tx)
	draftPageRevisionRepo := uc.draftPageRevisionRepo.WithTx(tx)
	topicRepo := uc.topicRepo.WithTx(tx)
	topicMemberRepo := uc.topicMemberRepo.WithTx(tx)
	pageAttachmentRefRepo := uc.pageAttachmentRefRepo.WithTx(tx)

	// 1. Wikiリンク解析・リンク先ページの自動作成
	keys, topicMapForLinks, err := scanAndLookupWikilinks(ctx, input.Body, input.CurrentTopicName, input.SpaceID, topicRepo)
	if err != nil {
		return nil, fmt.Errorf("wikiリンクのスキャンに失敗しました: %w", err)
	}

	linkedPageIDs, pageLocations, err := resolveAndCreateLinkedPages(
		ctx, keys, topicMapForLinks, input.SpaceID, input.SpaceMemberID, pageRepo, pageEditorRepo,
	)
	if err != nil {
		return nil, fmt.Errorf("wikiリンクの解析に失敗しました: %w", err)
	}

	// 2. bodyHTML内のWikiリンクを<a>タグに変換
	bodyHTML := input.BodyHTML
	if len(pageLocations) > 0 {
		bodyHTML = markup.ReplaceWikilinks(bodyHTML, input.CurrentTopicName, input.SpaceIdentifier, pageLocations)
	}

	// 3. 添付ファイル参照の書き込み
	if err := applyAttachmentRefChanges(ctx, input.PageID, input.SpaceID, input.AttachmentRefsToAdd, input.AttachmentRefsToRemove, pageAttachmentRefRepo); err != nil {
		return nil, fmt.Errorf("添付ファイル参照の同期に失敗しました: %w", err)
	}

	// 4. 競合する未公開ページが存在する場合、論理削除する
	if input.UnpublishedConflictingPageID != nil {
		err := pageRepo.DiscardByID(ctx, *input.UnpublishedConflictingPageID, input.SpaceID, now)
		if err != nil {
			return nil, fmt.Errorf("競合する未公開ページの論理削除に失敗しました: %w", err)
		}
	}

	// 5. Pageを更新（DraftPageの内容を反映 + publishedAtを更新）
	var title string
	if input.Title != nil {
		title = *input.Title
	}
	updatedPage, err := pageRepo.Update(ctx, repository.UpdatePageInput{
		ID:                        input.PageID,
		SpaceID:                   input.SpaceID,
		TopicID:                   input.TopicID,
		Title:                     input.Title,
		Body:                      input.Body,
		BodyHTML:                  bodyHTML,
		LinkedPageIDs:             linkedPageIDs,
		ModifiedAt:                now,
		PublishedAt:               &now,
		FeaturedImageAttachmentID: input.FeaturedImageAttachmentID,
	})
	if err != nil {
		return nil, fmt.Errorf("ページの更新に失敗しました: %w", err)
	}

	// 6. PageRevisionを作成（スナップショット）
	_, err = pageRevisionRepo.Create(ctx, repository.CreatePageRevisionInput{
		SpaceID:       input.SpaceID,
		SpaceMemberID: input.SpaceMemberID,
		PageID:        input.PageID,
		Title:         title,
		Body:          input.Body,
		BodyHTML:      bodyHTML,
	})
	if err != nil {
		return nil, fmt.Errorf("ページリビジョンの作成に失敗しました: %w", err)
	}

	// 7. PageEditorを追加・更新
	pageEditor, err := pageEditorRepo.FindOrCreate(ctx, repository.FindOrCreateInput{
		SpaceID:            input.SpaceID,
		PageID:             input.PageID,
		SpaceMemberID:      input.SpaceMemberID,
		LastPageModifiedAt: now,
	})
	if err != nil {
		return nil, fmt.Errorf("ページ編集者の追加に失敗しました: %w", err)
	}

	_, err = pageEditorRepo.UpdateLastPageModifiedAt(ctx, repository.UpdateLastPageModifiedAtInput{
		ID:                 pageEditor.ID,
		SpaceID:            input.SpaceID,
		LastPageModifiedAt: now,
	})
	if err != nil {
		return nil, fmt.Errorf("ページ編集者の更新に失敗しました: %w", err)
	}

	// 8. TopicMemberのlast_page_modified_atを更新
	err = topicMemberRepo.UpdateLastPageModifiedAt(ctx, input.SpaceID, input.TopicID, input.SpaceMemberID, now)
	if err != nil {
		return nil, fmt.Errorf("トピックメンバーの更新に失敗しました: %w", err)
	}

	// 9. DraftPageRevisionとDraftPageを削除（下書きが存在する場合のみ）
	if input.DraftPageID != "" {
		err = draftPageRevisionRepo.DeleteByDraftPageID(ctx, input.DraftPageID, input.SpaceID)
		if err != nil {
			return nil, fmt.Errorf("下書きページリビジョンの削除に失敗しました: %w", err)
		}

		err = draftPageRepo.Delete(ctx, input.DraftPageID, input.SpaceID)
		if err != nil {
			return nil, fmt.Errorf("下書きページの削除に失敗しました: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	return &PublishPageOutput{
		Page:        updatedPage,
		PublishedAt: now,
	}, nil
}
