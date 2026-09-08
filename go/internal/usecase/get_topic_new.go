package usecase

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetTopicNewUsecase gathers what the topic creation form shows.
//
// [Ja] GetTopicNewUsecase はトピック作成フォームが表示するものを集める。
type GetTopicNewUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
}

// NewGetTopicNewUsecase は GetTopicNewUsecase を生成する
func NewGetTopicNewUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
) *GetTopicNewUsecase {
	return &GetTopicNewUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
	}
}

// GetTopicNewInput はトピック作成フォームの表示に必要な入力パラメータ
type GetTopicNewInput struct {
	SpaceIdentifier model.SpaceIdentifier
	UserID          model.UserID
}

// GetTopicNewOutput はトピック作成フォームの表示に必要なデータを保持する
type GetTopicNewOutput struct {
	Space *model.Space
}

// Execute はトピックを作成するスペースを解決する
func (uc *GetTopicNewUsecase) Execute(ctx context.Context, input GetTopicNewInput) (*GetTopicNewOutput, error) {
	space, _, err := fetchTopicCreateAccess(ctx, uc.spaceRepo, uc.spaceMemberRepo, input.SpaceIdentifier, input.UserID)
	if err != nil {
		return nil, err
	}

	return &GetTopicNewOutput{Space: space}, nil
}
