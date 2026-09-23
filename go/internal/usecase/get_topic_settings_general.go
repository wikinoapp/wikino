package usecase

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetTopicSettingsGeneralUsecaseはトピックの一般設定画面が表示するものを集める。
type GetTopicSettingsGeneralUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// NewGetTopicSettingsGeneralUsecaseはGetTopicSettingsGeneralUsecaseを生成する
func NewGetTopicSettingsGeneralUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
) *GetTopicSettingsGeneralUsecase {
	return &GetTopicSettingsGeneralUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
	}
}

// GetTopicSettingsGeneralInputは一般設定画面の表示に必要な入力パラメータ
type GetTopicSettingsGeneralInput struct {
	SpaceIdentifier model.SpaceIdentifier
	TopicNumber     int32
	UserID          model.UserID
}

// GetTopicSettingsGeneralOutputは一般設定画面の表示に必要なデータを保持する
type GetTopicSettingsGeneralOutput struct {
	Space *model.Space
	Topic *model.Topic
}

// Executeは一般設定を編集するトピックを解決する
func (uc *GetTopicSettingsGeneralUsecase) Execute(ctx context.Context, input GetTopicSettingsGeneralInput) (*GetTopicSettingsGeneralOutput, error) {
	access, err := fetchTopicUpdateAccess(
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

	return &GetTopicSettingsGeneralOutput{Space: access.Space, Topic: access.Topic}, nil
}
