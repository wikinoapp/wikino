package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetAPITopicUsecaseは、公開APIでトークンを束縛したスペースのトピックを取得するユースケース
type GetAPITopicUsecase struct {
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// NewGetAPITopicUsecaseはGetAPITopicUsecaseを生成する
func NewGetAPITopicUsecase(topicRepo *repository.TopicRepository, topicMemberRepo *repository.TopicMemberRepository) *GetAPITopicUsecase {
	return &GetAPITopicUsecase{
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
	}
}

// GetAPITopicInputはトピックの取得に必要な入力パラメータ
type GetAPITopicInput struct {
	Principal       *model.APIPrincipal
	SpaceIdentifier model.SpaceIdentifier
	TopicNumber     int32
}

// GetAPITopicOutputはトピックの取得結果
type GetAPITopicOutput struct {
	Topic *model.Topic
}

// Executeはトークンを束縛したスペースのトピックを返す。
// トピックが存在しない場合と、閲覧者が開けない非公開トピックの場合は、存在を秘匿するため未存在として扱う
func (uc *GetAPITopicUsecase) Execute(ctx context.Context, input GetAPITopicInput) (*GetAPITopicOutput, error) {
	space, err := resolveAPISpace(ctx, input.Principal, input.SpaceIdentifier)
	if err != nil {
		return nil, err
	}

	notFound := &model.AppError{
		Code:    model.AppErrCodeResourceNotFound,
		UserMsg: i18n.T(ctx, "error_not_found_message"),
	}

	topic, err := uc.topicRepo.FindBySpaceAndNumber(ctx, space.ID, input.TopicNumber)
	if err != nil {
		return nil, fmt.Errorf("トピックの取得に失敗: %w", err)
	}
	if topic == nil {
		return nil, notFound
	}

	var topicMember *model.TopicMember
	if spaceMember := input.Principal.SpaceMember; spaceMember != nil {
		topicMember, err = uc.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, topic.ID)
		if err != nil {
			return nil, fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
		}
	}

	if !newAPIAuthorizer(input.Principal, topicMember).CanShowTopic(topic) {
		return nil, notFound
	}

	return &GetAPITopicOutput{Topic: topic}, nil
}
