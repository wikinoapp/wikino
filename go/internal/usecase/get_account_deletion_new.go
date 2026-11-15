package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetAccountDeletionNewUsecaseはアカウントの削除の確認画面が表示するものを集める。
type GetAccountDeletionNewUsecase struct {
	spaceRepo *repository.SpaceRepository
}

// NewGetAccountDeletionNewUsecaseはGetAccountDeletionNewUsecaseを生成する。
func NewGetAccountDeletionNewUsecase(spaceRepo *repository.SpaceRepository) *GetAccountDeletionNewUsecase {
	return &GetAccountDeletionNewUsecase{spaceRepo: spaceRepo}
}

// GetAccountDeletionNewInputは確認画面を描画するための入力パラメータ。
type GetAccountDeletionNewInput struct {
	UserID model.UserID
}

// GetAccountDeletionNewOutputは、ユーザーが参加しているスペースを返す。
// 1つでもあれば、アカウントを削除できない。
type GetAccountDeletionNewOutput struct {
	ActiveSpaces []*model.Space
}

// Executeはユーザーが参加しているスペースの一覧を取得する。
func (uc *GetAccountDeletionNewUsecase) Execute(ctx context.Context, input GetAccountDeletionNewInput) (*GetAccountDeletionNewOutput, error) {
	spaces, err := uc.spaceRepo.ListActiveByUser(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("参加しているスペースの取得に失敗: %w", err)
	}

	return &GetAccountDeletionNewOutput{ActiveSpaces: spaces}, nil
}
