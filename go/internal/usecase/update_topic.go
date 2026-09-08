package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// UpdateTopicUsecase saves the general settings of a topic.
//
// [Ja] UpdateTopicUsecase はトピックの一般設定を保存する。
type UpdateTopicUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
	updateValidator *validator.TopicUpdateValidator
}

// NewUpdateTopicUsecase は UpdateTopicUsecase を生成する
func NewUpdateTopicUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	updateValidator *validator.TopicUpdateValidator,
) *UpdateTopicUsecase {
	return &UpdateTopicUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
		updateValidator: updateValidator,
	}
}

// UpdateTopicInput はトピック更新の入力パラメータ。
// Visibility はフォームが送信した文字列で、変換はバリデーターが行う。
type UpdateTopicInput struct {
	SpaceIdentifier model.SpaceIdentifier
	TopicNumber     int32
	UserID          model.UserID
	Name            string
	Description     string
	Visibility      string
}

// UpdateTopicOutput は更新されたトピックとそれが属するスペースを保持する
type UpdateTopicOutput struct {
	Space *model.Space
	Topic *model.Topic
}

// Execute はトピックの一般設定を更新する
func (uc *UpdateTopicUsecase) Execute(ctx context.Context, input UpdateTopicInput) (*UpdateTopicOutput, error) {
	// 1. データ取得と認可チェック
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

	// 2. バリデーション
	visibility, err := uc.updateValidator.Validate(ctx, validator.TopicUpdateValidatorInput{
		Name:        input.Name,
		Description: input.Description,
		Visibility:  input.Visibility,
		TopicID:     access.Topic.ID,
		SpaceID:     access.Space.ID,
	})
	if err != nil {
		return nil, err
	}

	// 3. 永続化
	//
	// The update is a single statement, so it is issued without a transaction of its own.
	//
	// [Ja] 更新は 1 文で済むため、専用のトランザクションを開かずに実行する。
	topic, err := uc.topicRepo.Update(ctx, repository.UpdateTopicInput{
		ID:          access.Topic.ID,
		SpaceID:     access.Space.ID,
		Name:        input.Name,
		Description: input.Description,
		Visibility:  visibility,
	})
	if err != nil {
		return nil, fmt.Errorf("トピックの更新に失敗: %w", err)
	}

	return &UpdateTopicOutput{Space: access.Space, Topic: topic}, nil
}
