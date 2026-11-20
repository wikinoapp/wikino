package usecase

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetTopicDeletionNewUsecaseはトピックの削除の確認画面が表示するものを集める。
type GetTopicDeletionNewUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// NewGetTopicDeletionNewUsecaseはGetTopicDeletionNewUsecaseを生成する。
func NewGetTopicDeletionNewUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
) *GetTopicDeletionNewUsecase {
	return &GetTopicDeletionNewUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
	}
}

// GetTopicDeletionNewInputは確認画面を描画するための入力パラメータ。
type GetTopicDeletionNewInput struct {
	SpaceIdentifier model.SpaceIdentifier
	TopicNumber     int32
	UserID          model.UserID
}

// GetTopicDeletionNewOutputは削除するトピックとそのスペースを返す。
type GetTopicDeletionNewOutput struct {
	Space *model.Space
	Topic *model.Topic
}

// Executeは削除するトピックを解決する。トピックを削除できない閲覧者は拒否する。
func (uc *GetTopicDeletionNewUsecase) Execute(ctx context.Context, input GetTopicDeletionNewInput) (*GetTopicDeletionNewOutput, error) {
	target, err := fetchTopicDeleteAccess(
		ctx,
		uc.spaceRepo,
		uc.spaceMemberRepo,
		uc.topicRepo,
		uc.topicMemberRepo,
		input.SpaceIdentifier,
		input.TopicNumber,
		input.UserID,
	)
	if err != nil {
		return nil, err
	}

	return &GetTopicDeletionNewOutput{Space: target.Space, Topic: target.Topic}, nil
}
