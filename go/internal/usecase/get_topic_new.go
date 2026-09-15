package usecase

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetTopicNewUsecaseはトピック作成フォームが表示するものを集める。
type GetTopicNewUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
}

// NewGetTopicNewUsecaseはGetTopicNewUsecaseを生成する
func NewGetTopicNewUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
) *GetTopicNewUsecase {
	return &GetTopicNewUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
	}
}

// GetTopicNewInputはトピック作成フォームの表示に必要な入力パラメータ
type GetTopicNewInput struct {
	SpaceIdentifier model.SpaceIdentifier
	UserID          model.UserID
}

// GetTopicNewOutputはトピック作成フォームの表示に必要なデータを保持する
type GetTopicNewOutput struct {
	Space *model.Space
}

// Executeはトピックを作成するスペースを解決する
func (uc *GetTopicNewUsecase) Execute(ctx context.Context, input GetTopicNewInput) (*GetTopicNewOutput, error) {
	space, _, err := fetchTopicCreateAccess(ctx, uc.spaceRepo, uc.spaceMemberRepo, input.SpaceIdentifier, input.UserID)
	if err != nil {
		return nil, err
	}

	return &GetTopicNewOutput{Space: space}, nil
}
