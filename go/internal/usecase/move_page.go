package usecase

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// MovePageUsecase はページ移動ユースケース
type MovePageUsecase struct {
	db              *sql.DB
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
	createValidator *validator.PageMoveCreateValidator
}

// NewMovePageUsecase は MovePageUsecase を生成する
func NewMovePageUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	createValidator *validator.PageMoveCreateValidator,
) *MovePageUsecase {
	return &MovePageUsecase{
		db:              db,
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		pageRepo:        pageRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
		createValidator: createValidator,
	}
}

// MovePageInput はページ移動の入力パラメータ
type MovePageInput struct {
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	UserID          model.UserID
	DestTopicNumber string
}

// MovePageOutput はページ移動の出力パラメータ
type MovePageOutput struct {
	Page *model.Page
}

// Execute はページを別のトピックに移動する
func (uc *MovePageUsecase) Execute(ctx context.Context, input MovePageInput) (*MovePageOutput, error) {
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
	var pageTitle string
	if data.page.Title != nil {
		pageTitle = *data.page.Title
	}

	destTopic, err := uc.createValidator.Validate(ctx, validator.PageMoveCreateValidatorInput{
		DestTopicNumber: input.DestTopicNumber,
		PageID:          data.page.ID,
		PageTitle:       pageTitle,
		CurrentTopicID:  data.page.TopicID,
		SpaceID:         data.space.ID,
		SpaceMember:     data.spaceMember,
	})
	if err != nil {
		return nil, err
	}

	// 4. 永続化
	return uc.movePage(ctx, data.page.ID, data.space.ID, destTopic.ID)
}

func (uc *MovePageUsecase) pageAccessRepos() pageAccessRepos {
	return pageAccessRepos{
		spaceRepo:       uc.spaceRepo,
		spaceMemberRepo: uc.spaceMemberRepo,
		pageRepo:        uc.pageRepo,
		topicRepo:       uc.topicRepo,
		topicMemberRepo: uc.topicMemberRepo,
	}
}

func (uc *MovePageUsecase) movePage(ctx context.Context, pageID model.PageID, spaceID model.SpaceID, destTopicID model.TopicID) (*MovePageOutput, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	pageRepo := uc.pageRepo.WithTx(tx)

	page, err := pageRepo.MoveTopic(ctx, repository.MoveTopicInput{
		ID:      pageID,
		SpaceID: spaceID,
		TopicID: destTopicID,
	})
	if err != nil {
		return nil, fmt.Errorf("ページのトピック変更に失敗しました: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	return &MovePageOutput{
		Page: page,
	}, nil
}
