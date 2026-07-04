package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// AutoSaveDraftPageUsecase は下書きページの自動保存ユースケース
type AutoSaveDraftPageUsecase struct {
	db              *sql.DB
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	draftPageRepo   *repository.DraftPageRepository
	pageRepo        *repository.PageRepository
	pageEditorRepo  *repository.PageEditorRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
	attachmentRepo  *repository.AttachmentRepository
}

// NewAutoSaveDraftPageUsecase は AutoSaveDraftPageUsecase を生成する
func NewAutoSaveDraftPageUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	draftPageRepo *repository.DraftPageRepository,
	pageRepo *repository.PageRepository,
	pageEditorRepo *repository.PageEditorRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	attachmentRepo *repository.AttachmentRepository,
) *AutoSaveDraftPageUsecase {
	return &AutoSaveDraftPageUsecase{
		db:              db,
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		draftPageRepo:   draftPageRepo,
		pageRepo:        pageRepo,
		pageEditorRepo:  pageEditorRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
		attachmentRepo:  attachmentRepo,
	}
}

// AutoSaveDraftPageInput は下書き自動保存の入力パラメータ
type AutoSaveDraftPageInput struct {
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	UserID          model.UserID
	Title           *string
	Body            string
}

// AutoSaveDraftPageOutput は下書き自動保存の出力パラメータ
type AutoSaveDraftPageOutput struct {
	DraftPage  *model.DraftPage
	ModifiedAt time.Time
}

// Execute は下書きページを自動保存する
func (uc *AutoSaveDraftPageUsecase) Execute(ctx context.Context, input AutoSaveDraftPageInput) (*AutoSaveDraftPageOutput, error) {
	// 1. データ取得
	data, err := fetchPageAccessData(ctx, uc.pageAccessRepos(), input.SpaceIdentifier, input.PageNumber, input.UserID)
	if err != nil {
		return nil, err
	}

	// 2. 認可チェック
	if err := authorizePageUpdate(ctx, data); err != nil {
		return nil, err
	}

	// 3. 永続化
	return uc.saveDraft(ctx, data, input)
}

func (uc *AutoSaveDraftPageUsecase) pageAccessRepos() pageAccessRepos {
	return pageAccessRepos{
		spaceRepo:       uc.spaceRepo,
		spaceMemberRepo: uc.spaceMemberRepo,
		pageRepo:        uc.pageRepo,
		topicRepo:       uc.topicRepo,
		topicMemberRepo: uc.topicMemberRepo,
	}
}

func (uc *AutoSaveDraftPageUsecase) saveDraft(ctx context.Context, data *pageAccessData, input AutoSaveDraftPageInput) (*AutoSaveDraftPageOutput, error) {
	now := time.Now()

	// Before the transaction: extract the featured image only. The body HTML
	// rendering and wiki link resolution are unified into saveDraftPageContent,
	// which runs inside the transaction.
	//
	// [Ja] トランザクション前: アイキャッチ画像のみ抽出する。bodyHTML 本体のレンダリングと
	// Wiki リンクの解決はトランザクション内の saveDraftPageContent に一本化した。
	featuredImageAttachmentID, err := extractFeaturedImageAttachmentID(ctx, input.Body, data.space.ID, uc.attachmentRepo)
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

	contentInput := saveDraftPageContentInput{
		SpaceID:                   data.space.ID,
		PageID:                    data.page.ID,
		SpaceMemberID:             data.spaceMember.ID,
		TopicID:                   data.page.TopicID,
		Title:                     input.Title,
		Body:                      input.Body,
		FeaturedImageAttachmentID: featuredImageAttachmentID,
		SpaceIdentifier:           input.SpaceIdentifier,
		CurrentTopicName:          data.topic.Name,
	}

	result, err := saveDraftPageContent(ctx, contentInput, now,
		uc.draftPageRepo.WithTx(tx),
		uc.pageRepo.WithTx(tx),
		uc.pageEditorRepo.WithTx(tx),
		uc.topicRepo,
		uc.attachmentRepo,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	return &AutoSaveDraftPageOutput{
		DraftPage:  result.DraftPage,
		ModifiedAt: now,
	}, nil
}
