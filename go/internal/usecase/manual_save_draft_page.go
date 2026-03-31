package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// ManualSaveDraftPageUsecase は下書きページの手動保存ユースケース
type ManualSaveDraftPageUsecase struct {
	db                    *sql.DB
	spaceRepo             *repository.SpaceRepository
	spaceMemberRepo       *repository.SpaceMemberRepository
	draftPageRepo         *repository.DraftPageRepository
	draftPageRevisionRepo *repository.DraftPageRevisionRepository
	pageRepo              *repository.PageRepository
	pageEditorRepo        *repository.PageEditorRepository
	topicRepo             *repository.TopicRepository
	topicMemberRepo       *repository.TopicMemberRepository
	attachmentRepo        *repository.AttachmentRepository
}

// NewManualSaveDraftPageUsecase は ManualSaveDraftPageUsecase を生成する
func NewManualSaveDraftPageUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	draftPageRepo *repository.DraftPageRepository,
	draftPageRevisionRepo *repository.DraftPageRevisionRepository,
	pageRepo *repository.PageRepository,
	pageEditorRepo *repository.PageEditorRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	attachmentRepo *repository.AttachmentRepository,
) *ManualSaveDraftPageUsecase {
	return &ManualSaveDraftPageUsecase{
		db:                    db,
		spaceRepo:             spaceRepo,
		spaceMemberRepo:       spaceMemberRepo,
		draftPageRepo:         draftPageRepo,
		draftPageRevisionRepo: draftPageRevisionRepo,
		pageRepo:              pageRepo,
		pageEditorRepo:        pageEditorRepo,
		topicRepo:             topicRepo,
		topicMemberRepo:       topicMemberRepo,
		attachmentRepo:        attachmentRepo,
	}
}

// ManualSaveDraftPageInput は下書きページの手動保存の入力パラメータ
type ManualSaveDraftPageInput struct {
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	UserID          model.UserID
	Title           *string
	Body            string
}

// ManualSaveDraftPageOutput は下書きページの手動保存の出力パラメータ
type ManualSaveDraftPageOutput struct {
	DraftPage         *model.DraftPage
	DraftPageRevision *model.DraftPageRevision
	TopicNumber       int32
}

// Execute はフォームから受け取った内容でDraftPageを更新し、リビジョンを作成する
func (uc *ManualSaveDraftPageUsecase) Execute(ctx context.Context, input ManualSaveDraftPageInput) (*ManualSaveDraftPageOutput, error) {
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

func (uc *ManualSaveDraftPageUsecase) pageAccessRepos() pageAccessRepos {
	return pageAccessRepos{
		spaceRepo:       uc.spaceRepo,
		spaceMemberRepo: uc.spaceMemberRepo,
		pageRepo:        uc.pageRepo,
		topicRepo:       uc.topicRepo,
		topicMemberRepo: uc.topicMemberRepo,
	}
}

func (uc *ManualSaveDraftPageUsecase) saveDraft(ctx context.Context, data *pageAccessData, input ManualSaveDraftPageInput) (*ManualSaveDraftPageOutput, error) {
	now := time.Now()

	// トランザクション前: 下書き保存に必要な事前計算データを取得
	saveData, err := calculateDraftPageSaveData(ctx, input.Body, data.topic.Name, data.space.ID, uc.topicRepo, uc.attachmentRepo)
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

	draftPageRevisionRepo := uc.draftPageRevisionRepo.WithTx(tx)

	contentInput := saveDraftPageContentInput{
		SpaceID:                   data.space.ID,
		PageID:                    data.page.ID,
		SpaceMemberID:             data.spaceMember.ID,
		TopicID:                   data.page.TopicID,
		Title:                     input.Title,
		Body:                      input.Body,
		BodyHTML:                  saveData.bodyHTML,
		FeaturedImageAttachmentID: saveData.featuredImageAttachmentID,
		WikilinkKeys:              saveData.wikilinkKeys,
		TopicMap:                  saveData.topicMap,
		SpaceIdentifier:           input.SpaceIdentifier,
		CurrentTopicName:          data.topic.Name,
	}

	// DraftPageのfind_or_create・リンク解決・更新
	result, err := saveDraftPageContent(ctx, contentInput, now,
		uc.draftPageRepo.WithTx(tx),
		uc.pageRepo.WithTx(tx),
		uc.pageEditorRepo.WithTx(tx),
	)
	if err != nil {
		return nil, err
	}

	// リビジョンを作成
	var title string
	if input.Title != nil {
		title = *input.Title
	}

	revision, err := draftPageRevisionRepo.Create(ctx, repository.CreateDraftPageRevisionInput{
		DraftPageID:   result.DraftPage.ID,
		SpaceID:       data.space.ID,
		SpaceMemberID: data.spaceMember.ID,
		Title:         title,
		Body:          input.Body,
		BodyHTML:      result.BodyHTML,
	})
	if err != nil {
		return nil, fmt.Errorf("下書きページリビジョンの作成に失敗しました: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	return &ManualSaveDraftPageOutput{
		DraftPage:         result.DraftPage,
		DraftPageRevision: revision,
		TopicNumber:       data.topic.Number,
	}, nil
}
