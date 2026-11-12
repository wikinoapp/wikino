package usecase

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetTopicSettingsUsecaseはトピックの設定のトップが表示するものを集める。
type GetTopicSettingsUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// NewGetTopicSettingsUsecaseはGetTopicSettingsUsecaseを生成する。
func NewGetTopicSettingsUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
) *GetTopicSettingsUsecase {
	return &GetTopicSettingsUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
	}
}

// GetTopicSettingsInputは画面を描画するための入力パラメータ。
type GetTopicSettingsInput struct {
	SpaceIdentifier model.SpaceIdentifier
	TopicNumber     int32
	UserID          model.UserID
}

// TopicSettingsItemsは、トピックの設定のトップに出す項目のうち閲覧者が開けるものを表す。
type TopicSettingsItems struct {
	// CanUpdateTopicは一般設定の項目を出すかを表す (topic:write)。一般設定の画面自体を
	// topic:writeで開くため、リンクと画面が誰を通すかについて別々の答えを出さないよう同じ判定にする。
	CanUpdateTopic bool

	// CanDeleteTopicはトピックの削除の項目を出すかを表す (topic:delete)。
	CanDeleteTopic bool
}

// CanShowは閲覧者がトピックの設定のトップを開けるか (出す項目が1つでもあるか) を返す。
func (i TopicSettingsItems) CanShow() bool {
	return i.CanUpdateTopic || i.CanDeleteTopic
}

// GetTopicSettingsOutputはスペース・トピックと、閲覧者が開けるトピックの設定の項目を返す。
type GetTopicSettingsOutput struct {
	Space *model.Space
	Topic *model.Topic
	Items TopicSettingsItems
}

// Executeはトピックと、閲覧者が開けるトピックの設定の項目を解決する。
//
// 開ける項目が1つも無い場合は「見つからない」として答える。そのトピックで誰が何を変更できるかを、
// 外部から試して確かめられないようにするためである。
func (uc *GetTopicSettingsUsecase) Execute(ctx context.Context, input GetTopicSettingsInput) (*GetTopicSettingsOutput, error) {
	target, err := fetchTopicSettingsTarget(
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

	items := TopicSettingsItems{
		CanUpdateTopic: target.Authorizer.CanUpdateTopic(),
		CanDeleteTopic: target.Authorizer.CanDeleteTopic(),
	}
	if !items.CanShow() {
		return nil, forbiddenError(ctx)
	}

	return &GetTopicSettingsOutput{
		Space: target.Space,
		Topic: target.Topic,
		Items: items,
	}, nil
}
