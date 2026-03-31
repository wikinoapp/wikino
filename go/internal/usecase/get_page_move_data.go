package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetPageMoveDataUsecase はページ移動フォームのデータ取得ユースケース
type GetPageMoveDataUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// NewGetPageMoveDataUsecase は GetPageMoveDataUsecase を生成する
func NewGetPageMoveDataUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
) *GetPageMoveDataUsecase {
	return &GetPageMoveDataUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		pageRepo:        pageRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
	}
}

// GetPageMoveDataInput はページ移動データ取得の入力パラメータ
type GetPageMoveDataInput struct {
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	UserID          model.UserID
}

// GetPageMoveDataOutput はページ移動データ取得の出力
type GetPageMoveDataOutput struct {
	Space           *model.Space
	SpaceMember     *model.SpaceMember
	Page            *model.Page
	TopicMember     *model.TopicMember
	CurrentTopic    *model.Topic
	AvailableTopics []*model.Topic
}

// Execute はページ移動フォームに必要なデータを取得する
func (uc *GetPageMoveDataUsecase) Execute(ctx context.Context, input GetPageMoveDataInput) (*GetPageMoveDataOutput, error) {
	// 1. データ取得 + 認可チェック
	data, err := fetchPageAccessData(ctx, uc.pageAccessRepos(), input.SpaceIdentifier, input.PageNumber, input.UserID)
	if err != nil {
		return nil, err
	}

	if err := authorizePageUpdate(ctx, data); err != nil {
		return nil, err
	}

	// 2. 移動先候補のトピック一覧を取得
	availableTopics, err := uc.availableTopicsForMove(ctx, data.spaceMember, data.space, data.page.TopicID)
	if err != nil {
		return nil, fmt.Errorf("移動先トピック一覧の取得に失敗: %w", err)
	}

	return &GetPageMoveDataOutput{
		Space:           data.space,
		SpaceMember:     data.spaceMember,
		Page:            data.page,
		TopicMember:     data.topicMember,
		CurrentTopic:    data.topic,
		AvailableTopics: availableTopics,
	}, nil
}

func (uc *GetPageMoveDataUsecase) pageAccessRepos() pageAccessRepos {
	return pageAccessRepos{
		spaceRepo:       uc.spaceRepo,
		spaceMemberRepo: uc.spaceMemberRepo,
		pageRepo:        uc.pageRepo,
		topicRepo:       uc.topicRepo,
		topicMemberRepo: uc.topicMemberRepo,
	}
}

// availableTopicsForMove は移動先候補のトピック一覧を取得する。
// スペースオーナーは全アクティブトピック、それ以外は所属トピックのみ返す。
// 現在のトピックは除外する。
// スペースオーナーは同スペース内の全トピックにCanCreatePageが真であり、
// 非オーナーはListJoinedBySpaceMemberが所属トピックのみを返すため、
// いずれの場合もリスト取得の段階で権限が暗黙的に満たされている。
func (uc *GetPageMoveDataUsecase) availableTopicsForMove(
	ctx context.Context,
	spaceMember *model.SpaceMember,
	space *model.Space,
	currentTopicID model.TopicID,
) ([]*model.Topic, error) {
	var topics []*model.Topic
	var err error

	if spaceMember.Role == model.SpaceMemberRoleOwner {
		topics, err = uc.topicRepo.ListActiveBySpace(ctx, space.ID)
	} else {
		topics, err = uc.topicRepo.ListJoinedBySpaceMember(ctx, spaceMember.ID, space.ID)
	}
	if err != nil {
		return nil, err
	}

	var filtered []*model.Topic
	for _, t := range topics {
		if t.ID == currentTopicID {
			continue
		}
		filtered = append(filtered, t)
	}

	return filtered, nil
}
