package usecase

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// MovePageUsecase はページ移動ユースケース
type MovePageUsecase struct {
	db       *sql.DB
	pageRepo *repository.PageRepository
}

// NewMovePageUsecase は MovePageUsecase を生成する
func NewMovePageUsecase(
	db *sql.DB,
	pageRepo *repository.PageRepository,
) *MovePageUsecase {
	return &MovePageUsecase{
		db:       db,
		pageRepo: pageRepo,
	}
}

// MovePageInput はページ移動の入力パラメータ
type MovePageInput struct {
	PageID      model.PageID
	SpaceID     model.SpaceID
	DestTopicID model.TopicID
}

// MovePageOutput はページ移動の出力パラメータ
type MovePageOutput struct {
	Page *model.Page
}

// Execute はページを別のトピックに移動する
func (uc *MovePageUsecase) Execute(ctx context.Context, input MovePageInput) (*MovePageOutput, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	pageRepo := uc.pageRepo.WithTx(tx)

	page, err := pageRepo.MoveTopic(ctx, repository.MoveTopicInput{
		ID:      input.PageID,
		SpaceID: input.SpaceID,
		TopicID: input.DestTopicID,
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
