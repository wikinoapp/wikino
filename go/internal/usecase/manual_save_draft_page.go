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
	draftPageRepo         *repository.DraftPageRepository
	draftPageRevisionRepo *repository.DraftPageRevisionRepository
	pageRepo              *repository.PageRepository
	pageEditorRepo        *repository.PageEditorRepository
	topicRepo             *repository.TopicRepository
	attachmentRepo        *repository.AttachmentRepository
}

// NewManualSaveDraftPageUsecase は ManualSaveDraftPageUsecase を生成する
func NewManualSaveDraftPageUsecase(
	db *sql.DB,
	draftPageRepo *repository.DraftPageRepository,
	draftPageRevisionRepo *repository.DraftPageRevisionRepository,
	pageRepo *repository.PageRepository,
	pageEditorRepo *repository.PageEditorRepository,
	topicRepo *repository.TopicRepository,
	attachmentRepo *repository.AttachmentRepository,
) *ManualSaveDraftPageUsecase {
	return &ManualSaveDraftPageUsecase{
		db:                    db,
		draftPageRepo:         draftPageRepo,
		draftPageRevisionRepo: draftPageRevisionRepo,
		pageRepo:              pageRepo,
		pageEditorRepo:        pageEditorRepo,
		topicRepo:             topicRepo,
		attachmentRepo:        attachmentRepo,
	}
}

// ManualSaveDraftPageInput は下書きページの手動保存の入力パラメータ
type ManualSaveDraftPageInput struct {
	SpaceID          model.SpaceID
	PageID           model.PageID
	SpaceMemberID    model.SpaceMemberID
	TopicID          model.TopicID
	Title            *string
	Body             string
	SpaceIdentifier  model.SpaceIdentifier
	CurrentTopicName string
}

// ManualSaveDraftPageOutput は下書きページの手動保存の出力パラメータ
type ManualSaveDraftPageOutput struct {
	DraftPageRevision *model.DraftPageRevision
}

// Execute はフォームから受け取った内容でDraftPageを更新し、リビジョンを作成する
func (uc *ManualSaveDraftPageUsecase) Execute(ctx context.Context, input ManualSaveDraftPageInput) (*ManualSaveDraftPageOutput, error) {
	now := time.Now()

	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	draftPageRevisionRepo := uc.draftPageRevisionRepo.WithTx(tx)

	// DraftPageのfind_or_create・Markdown処理・更新
	result, err := saveDraftPageContent(ctx, saveDraftPageContentInput(input), now,
		uc.draftPageRepo.WithTx(tx),
		uc.pageRepo.WithTx(tx),
		uc.pageEditorRepo.WithTx(tx),
		uc.topicRepo.WithTx(tx),
		uc.attachmentRepo.WithTx(tx),
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
		SpaceID:       input.SpaceID,
		SpaceMemberID: input.SpaceMemberID,
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
		DraftPageRevision: revision,
	}, nil
}
