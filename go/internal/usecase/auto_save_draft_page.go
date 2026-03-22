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
	db             *sql.DB
	draftPageRepo  *repository.DraftPageRepository
	pageRepo       *repository.PageRepository
	pageEditorRepo *repository.PageEditorRepository
	topicRepo      *repository.TopicRepository
	attachmentRepo *repository.AttachmentRepository
}

// NewAutoSaveDraftPageUsecase は AutoSaveDraftPageUsecase を生成する
func NewAutoSaveDraftPageUsecase(
	db *sql.DB,
	draftPageRepo *repository.DraftPageRepository,
	pageRepo *repository.PageRepository,
	pageEditorRepo *repository.PageEditorRepository,
	topicRepo *repository.TopicRepository,
	attachmentRepo *repository.AttachmentRepository,
) *AutoSaveDraftPageUsecase {
	return &AutoSaveDraftPageUsecase{
		db:             db,
		draftPageRepo:  draftPageRepo,
		pageRepo:       pageRepo,
		pageEditorRepo: pageEditorRepo,
		topicRepo:      topicRepo,
		attachmentRepo: attachmentRepo,
	}
}

// AutoSaveDraftPageInput は下書き自動保存の入力パラメータ
type AutoSaveDraftPageInput struct {
	SpaceID          model.SpaceID
	PageID           model.PageID
	SpaceMemberID    model.SpaceMemberID
	TopicID          model.TopicID
	Title            *string
	Body             string
	SpaceIdentifier  model.SpaceIdentifier
	CurrentTopicName string
}

// AutoSaveDraftPageOutput は下書き自動保存の出力パラメータ
type AutoSaveDraftPageOutput struct {
	DraftPage  *model.DraftPage
	ModifiedAt time.Time
}

// Execute は下書きページを自動保存する
func (uc *AutoSaveDraftPageUsecase) Execute(ctx context.Context, input AutoSaveDraftPageInput) (*AutoSaveDraftPageOutput, error) {
	now := time.Now()

	// トランザクション前: 下書き保存に必要な事前計算データを取得
	saveData, err := calculateDraftPageSaveData(ctx, input.Body, input.CurrentTopicName, input.SpaceID, uc.topicRepo, uc.attachmentRepo)
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
		SpaceID:                   input.SpaceID,
		PageID:                    input.PageID,
		SpaceMemberID:             input.SpaceMemberID,
		TopicID:                   input.TopicID,
		Title:                     input.Title,
		Body:                      input.Body,
		BodyHTML:                  saveData.bodyHTML,
		FeaturedImageAttachmentID: saveData.featuredImageAttachmentID,
		WikilinkKeys:              saveData.wikilinkKeys,
		TopicMap:                  saveData.topicMap,
		SpaceIdentifier:           input.SpaceIdentifier,
		CurrentTopicName:          input.CurrentTopicName,
	}

	result, err := saveDraftPageContent(ctx, contentInput, now,
		uc.draftPageRepo.WithTx(tx),
		uc.pageRepo.WithTx(tx),
		uc.pageEditorRepo.WithTx(tx),
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
