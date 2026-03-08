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
	attachmentRepo        *repository.AttachmentRepository
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
	attachmentRepo *repository.AttachmentRepository,
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
		attachmentRepo:        attachmentRepo,
		pageAttachmentRefRepo: pageAttachmentRefRepo,
	}
}

// PublishPageInput はページ公開の入力パラメータ
type PublishPageInput struct {
	SpaceID          model.SpaceID
	PageID           model.PageID
	SpaceMemberID    model.SpaceMemberID
	TopicID          model.TopicID
	DraftPageID      model.DraftPageID
	Title            *string
	Body             string
	SpaceIdentifier  model.SpaceIdentifier
	CurrentTopicName string
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
	attachmentRepo := uc.attachmentRepo.WithTx(tx)
	pageAttachmentRefRepo := uc.pageAttachmentRefRepo.WithTx(tx)

	// 1. Markdownレンダリング
	bodyHTML := markup.RenderMarkdown(input.Body)

	// 2. Wikiリンク解析・リンク先ページの自動作成
	linkedPageIDs, pageLocations, err := resolveAndCreateLinkedPages(
		ctx, input.Body, input.CurrentTopicName, input.SpaceID, input.SpaceMemberID, pageRepo, pageEditorRepo, topicRepo,
	)
	if err != nil {
		return nil, fmt.Errorf("wikiリンクの解析に失敗しました: %w", err)
	}

	// 3. bodyHTML内のWikiリンクを<a>タグに変換
	if len(pageLocations) > 0 {
		bodyHTML = markup.ReplaceWikilinks(bodyHTML, input.CurrentTopicName, input.SpaceIdentifier, pageLocations)
	}

	// 4. 添付ファイル参照の同期（FilterAttachments前に実行。FilterAttachmentsが/attachments/{id}をdata属性に変換するため）
	if err := syncAttachmentReferences(ctx, bodyHTML, input.PageID, input.SpaceID, attachmentRepo, pageAttachmentRefRepo); err != nil {
		return nil, fmt.Errorf("添付ファイル参照の同期に失敗しました: %w", err)
	}

	// 5. アイキャッチ画像の抽出
	featuredImageAttachmentID, err := extractFeaturedImageAttachmentID(ctx, input.Body, input.SpaceID, attachmentRepo)
	if err != nil {
		return nil, fmt.Errorf("アイキャッチ画像の抽出に失敗しました: %w", err)
	}

	// 6. 添付ファイルフィルター
	bodyHTML, err = markup.FilterAttachments(ctx, bodyHTML, input.SpaceID, attachmentRepo)
	if err != nil {
		return nil, fmt.Errorf("添付ファイルのフィルター処理に失敗しました: %w", err)
	}

	// 7. スタンドアロン画像のラッピング
	bodyHTML = markup.WrapStandaloneImageLinks(bodyHTML)

	// 8. Pageを更新（DraftPageの内容を反映 + publishedAtを更新）
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
		FeaturedImageAttachmentID: featuredImageAttachmentID,
	})
	if err != nil {
		return nil, fmt.Errorf("ページの更新に失敗しました: %w", err)
	}

	// 9. PageRevisionを作成（スナップショット）
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

	// 10. PageEditorを追加・更新
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

	// 11. TopicMemberのlast_page_modified_atを更新
	err = topicMemberRepo.UpdateLastPageModifiedAt(ctx, input.SpaceID, input.TopicID, input.SpaceMemberID, now)
	if err != nil {
		return nil, fmt.Errorf("トピックメンバーの更新に失敗しました: %w", err)
	}

	// 12. DraftPageRevisionとDraftPageを削除（下書きが存在する場合のみ）
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
	// bodyHTMLから添付ファイルIDを抽出
	newIDStrings := markup.ExtractAttachmentIDs(bodyHTML)

	// 既存の参照を取得
	existingRefs, err := pageAttachmentRefRepo.ListByPageID(ctx, pageID, spaceID)
	if err != nil {
		return fmt.Errorf("既存の添付ファイル参照の取得に失敗しました: %w", err)
	}

	existingIDSet := make(map[model.AttachmentID]bool, len(existingRefs))
	for _, ref := range existingRefs {
		existingIDSet[ref.AttachmentID] = true
	}

	newIDSet := make(map[model.AttachmentID]bool, len(newIDStrings))
	for _, idStr := range newIDStrings {
		newIDSet[model.AttachmentID(idStr)] = true
	}

	// 追加分: 新規IDに含まれるが既存に含まれないもの
	var toAdd []model.AttachmentID
	for id := range newIDSet {
		if !existingIDSet[id] {
			toAdd = append(toAdd, id)
		}
	}

	// 削除分: 既存IDに含まれるが新規に含まれないもの
	var toRemove []model.AttachmentID
	for id := range existingIDSet {
		if !newIDSet[id] {
			toRemove = append(toRemove, id)
		}
	}

	// 追加分は添付ファイルの存在確認後に参照を作成
	if len(toAdd) > 0 {
		var validIDs []model.AttachmentID
		for _, id := range toAdd {
			exists, err := attachmentRepo.ExistsByIDAndSpace(ctx, id, spaceID)
			if err != nil {
				return fmt.Errorf("添付ファイルの存在確認に失敗しました: %w", err)
			}
			if exists {
				validIDs = append(validIDs, id)
			}
		}
		if len(validIDs) > 0 {
			if _, err := pageAttachmentRefRepo.CreateBatch(ctx, pageID, spaceID, validIDs); err != nil {
				return fmt.Errorf("添付ファイル参照の作成に失敗しました: %w", err)
			}
		}
	}

	// 削除分の参照を削除
	if len(toRemove) > 0 {
		if err := pageAttachmentRefRepo.DeleteByPageAndAttachmentIDs(ctx, pageID, spaceID, toRemove); err != nil {
			return fmt.Errorf("添付ファイル参照の削除に失敗しました: %w", err)
		}
	}

	return nil
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
