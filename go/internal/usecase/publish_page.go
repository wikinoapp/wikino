package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// PublishPageUsecase はページ公開ユースケース
type PublishPageUsecase struct {
	db                    *sql.DB
	spaceRepo             *repository.SpaceRepository
	spaceMemberRepo       *repository.SpaceMemberRepository
	pageRepo              *repository.PageRepository
	pageRevisionRepo      *repository.PageRevisionRepository
	pageEditorRepo        *repository.PageEditorRepository
	draftPageRepo         *repository.DraftPageRepository
	draftPageRevisionRepo *repository.DraftPageRevisionRepository
	topicRepo             *repository.TopicRepository
	topicMemberRepo       *repository.TopicMemberRepository
	attachmentRepo        *repository.AttachmentRepository
	pageAttachmentRefRepo *repository.PageAttachmentReferenceRepository
	updateValidator       *validator.PageUpdateValidator
}

// NewPublishPageUsecase は PublishPageUsecase を生成する
func NewPublishPageUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
	pageRevisionRepo *repository.PageRevisionRepository,
	pageEditorRepo *repository.PageEditorRepository,
	draftPageRepo *repository.DraftPageRepository,
	draftPageRevisionRepo *repository.DraftPageRevisionRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	attachmentRepo *repository.AttachmentRepository,
	pageAttachmentRefRepo *repository.PageAttachmentReferenceRepository,
	updateValidator *validator.PageUpdateValidator,
) *PublishPageUsecase {
	return &PublishPageUsecase{
		db:                    db,
		spaceRepo:             spaceRepo,
		spaceMemberRepo:       spaceMemberRepo,
		pageRepo:              pageRepo,
		pageRevisionRepo:      pageRevisionRepo,
		pageEditorRepo:        pageEditorRepo,
		draftPageRepo:         draftPageRepo,
		draftPageRevisionRepo: draftPageRevisionRepo,
		topicRepo:             topicRepo,
		topicMemberRepo:       topicMemberRepo,
		attachmentRepo:        attachmentRepo,
		pageAttachmentRefRepo: pageAttachmentRefRepo,
		updateValidator:       updateValidator,
	}
}

// PublishPageInput はページ公開の入力パラメータ
type PublishPageInput struct {
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	UserID          model.UserID
	// Title はバリデーション前の必須フィールド（空文字列の場合はバリデーションエラー）
	Title string
	Body  string
}

// PublishPageOutput はページ公開の出力パラメータ
type PublishPageOutput struct {
	Page        *model.Page
	PublishedAt time.Time
}

// Execute はページを公開する
func (uc *PublishPageUsecase) Execute(ctx context.Context, input PublishPageInput) (*PublishPageOutput, error) {
	// 1. データ取得
	data, err := fetchPageAccessData(ctx, uc.pageAccessRepos(), input.SpaceIdentifier, input.PageNumber, input.UserID)
	if err != nil {
		return nil, err
	}

	// 2. 認可チェック
	if err := authorizePageUpdate(ctx, data); err != nil {
		return nil, err
	}

	// 3. バリデーション
	unpublishedConflictingPageID, err := uc.updateValidator.Validate(ctx, validator.PageUpdateValidatorInput{
		Title:           input.Title,
		PageID:          data.page.ID,
		TopicID:         data.page.TopicID,
		SpaceID:         data.space.ID,
		SpaceIdentifier: input.SpaceIdentifier,
	})
	if err != nil {
		return nil, err
	}

	// 4. 追加データ取得（下書きページ）
	var draftPage *model.DraftPage
	if data.spaceMember != nil {
		draftPage, err = uc.draftPageRepo.FindByPageAndMember(ctx, data.page.ID, data.spaceMember.ID, data.space.ID)
		if err != nil {
			return nil, fmt.Errorf("下書きページの取得に失敗: %w", err)
		}
	}

	// 5. ビジネスロジック（トランザクション前）
	return uc.publishPage(ctx, data, draftPage, input, unpublishedConflictingPageID)
}

func (uc *PublishPageUsecase) pageAccessRepos() pageAccessRepos {
	return pageAccessRepos{
		spaceRepo:       uc.spaceRepo,
		spaceMemberRepo: uc.spaceMemberRepo,
		pageRepo:        uc.pageRepo,
		topicRepo:       uc.topicRepo,
		topicMemberRepo: uc.topicMemberRepo,
	}
}

func (uc *PublishPageUsecase) publishPage(ctx context.Context, data *pageAccessData, draftPage *model.DraftPage, input PublishPageInput, unpublishedConflictingPageID *model.PageID) (*PublishPageOutput, error) {
	now := time.Now()

	// トランザクション前: ページ公開に必要な事前計算データを取得
	pd, err := uc.calculatePublishData(ctx, input.Body, data.page.ID, data.space.ID)
	if err != nil {
		return nil, err
	}

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
	topicMemberRepo := uc.topicMemberRepo.WithTx(tx)
	pageAttachmentRefRepo := uc.pageAttachmentRefRepo.WithTx(tx)

	// Render the body HTML through markup.RenderHTML, the same unified path used by the
	// preview and page detail screens. The resolver auto-creates missing linked pages
	// within this transaction and records their IDs.
	//
	// [Ja] プレビュー・ページ詳細画面と同じ統合経路 markup.RenderHTML で本文 HTML を
	// レンダリングする。resolver がこのトランザクション内で存在しないリンク先ページを自動作成し、
	// その ID を記録する。
	resolver := &linkCreatingPageLocationResolver{
		spaceMemberID:  data.spaceMember.ID,
		topicRepo:      uc.topicRepo,
		pageRepo:       pageRepo,
		pageEditorRepo: pageEditorRepo,
	}
	bodyHTML, err := markup.RenderHTML(ctx, input.Body, data.topic.Name, data.space.ID, input.SpaceIdentifier, resolver, uc.attachmentRepo)
	if err != nil {
		return nil, fmt.Errorf("本文のレンダリングに失敗しました: %w", err)
	}
	linkedPageIDs := resolver.linkedPageIDs

	// 添付ファイル参照の書き込み
	if err := applyAttachmentRefChanges(ctx, data.page.ID, data.space.ID, pd.attachmentRefsToAdd, pd.attachmentRefsToRemove, pageAttachmentRefRepo); err != nil {
		return nil, fmt.Errorf("添付ファイル参照の同期に失敗しました: %w", err)
	}

	// 競合する未公開ページが存在する場合、論理削除する
	if unpublishedConflictingPageID != nil {
		err := pageRepo.DiscardByID(ctx, *unpublishedConflictingPageID, data.space.ID, now)
		if err != nil {
			return nil, fmt.Errorf("競合する未公開ページの論理削除に失敗しました: %w", err)
		}
	}

	// Titleをポインタに変換（空文字列の場合はnil）
	var titlePtr *string
	if input.Title != "" {
		titlePtr = &input.Title
	}

	// Pageを更新（DraftPageの内容を反映 + publishedAtを更新）
	updatedPage, err := pageRepo.Update(ctx, repository.UpdatePageInput{
		ID:                        data.page.ID,
		SpaceID:                   data.space.ID,
		TopicID:                   data.page.TopicID,
		Title:                     titlePtr,
		Body:                      input.Body,
		BodyHTML:                  bodyHTML,
		LinkedPageIDs:             linkedPageIDs,
		ModifiedAt:                now,
		PublishedAt:               &now,
		FeaturedImageAttachmentID: pd.featuredImageAttachmentID,
	})
	if err != nil {
		return nil, fmt.Errorf("ページの更新に失敗しました: %w", err)
	}

	// PageRevisionを作成（スナップショット）
	_, err = pageRevisionRepo.Create(ctx, repository.CreatePageRevisionInput{
		SpaceID:       data.space.ID,
		SpaceMemberID: data.spaceMember.ID,
		PageID:        data.page.ID,
		Title:         input.Title,
		Body:          input.Body,
		BodyHTML:      bodyHTML,
	})
	if err != nil {
		return nil, fmt.Errorf("ページリビジョンの作成に失敗しました: %w", err)
	}

	// PageEditorを追加・更新
	pageEditor, err := pageEditorRepo.FindOrCreate(ctx, repository.FindOrCreateInput{
		SpaceID:            data.space.ID,
		PageID:             data.page.ID,
		SpaceMemberID:      data.spaceMember.ID,
		LastPageModifiedAt: now,
	})
	if err != nil {
		return nil, fmt.Errorf("ページ編集者の追加に失敗しました: %w", err)
	}

	_, err = pageEditorRepo.UpdateLastPageModifiedAt(ctx, repository.UpdateLastPageModifiedAtInput{
		ID:                 pageEditor.ID,
		SpaceID:            data.space.ID,
		LastPageModifiedAt: now,
	})
	if err != nil {
		return nil, fmt.Errorf("ページ編集者の更新に失敗しました: %w", err)
	}

	// TopicMemberのlast_page_modified_atを更新
	err = topicMemberRepo.UpdateLastPageModifiedAt(ctx, data.space.ID, data.page.TopicID, data.spaceMember.ID, now)
	if err != nil {
		return nil, fmt.Errorf("トピックメンバーの更新に失敗しました: %w", err)
	}

	// DraftPageRevisionとDraftPageを削除（下書きが存在する場合のみ）
	if draftPage != nil {
		err = draftPageRevisionRepo.DeleteByDraftPageID(ctx, draftPage.ID, data.space.ID)
		if err != nil {
			return nil, fmt.Errorf("下書きページリビジョンの削除に失敗しました: %w", err)
		}

		err = draftPageRepo.Delete(ctx, draftPage.ID, data.space.ID)
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

// publishData はページ公開の事前計算データ
type publishData struct {
	featuredImageAttachmentID *model.AttachmentID
	attachmentRefsToAdd       []model.AttachmentID
	attachmentRefsToRemove    []model.AttachmentID
}

// calculatePublishData computes the attachment-reference diff and the featured image.
// The body HTML itself is rendered inside the save transaction through markup.RenderHTML
// (the same path as preview / page detail), so here Markdown is rendered only once to
// extract the attachment IDs needed for the reference diff.
//
// [Ja] calculatePublishData は添付ファイル参照の差分計算とアイキャッチ画像抽出を行う。
// bodyHTML 本体のレンダリングは保存トランザクション内の markup.RenderHTML
// (プレビュー・ページ詳細と同じ経路) に一本化したため、ここでは参照差分に必要な添付 ID 抽出を
// 目的に Markdown を 1 度だけレンダリングする。
func (uc *PublishPageUsecase) calculatePublishData(ctx context.Context, body string, pageID model.PageID, spaceID model.SpaceID) (*publishData, error) {
	rawHTML := markup.RenderMarkdown(body)

	toAdd, toRemove, err := calculateAttachmentRefDiff(ctx, rawHTML, pageID, spaceID, uc.attachmentRepo, uc.pageAttachmentRefRepo)
	if err != nil {
		return nil, fmt.Errorf("添付ファイル参照の差分計算に失敗しました: %w", err)
	}

	featuredImageAttachmentID, err := extractFeaturedImageAttachmentID(ctx, body, spaceID, uc.attachmentRepo)
	if err != nil {
		return nil, fmt.Errorf("アイキャッチ画像の抽出に失敗しました: %w", err)
	}

	return &publishData{
		featuredImageAttachmentID: featuredImageAttachmentID,
		attachmentRefsToAdd:       toAdd,
		attachmentRefsToRemove:    toRemove,
	}, nil
}
