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
	space, err := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
	if err != nil {
		return nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, nil
	}

	spaceMember, err := uc.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}
	if spaceMember == nil {
		return nil, nil
	}

	pg, err := uc.pageRepo.FindBySpaceAndNumber(ctx, space.ID, model.PageNumber(input.PageNumber))
	if err != nil {
		return nil, fmt.Errorf("ページの取得に失敗: %w", err)
	}
	if pg == nil {
		return nil, nil
	}

	topicMember, err := uc.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, pg.TopicID)
	if err != nil {
		return nil, fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
	}

	currentTopic, err := uc.topicRepo.FindBySpaceAndID(ctx, space.ID, pg.TopicID)
	if err != nil {
		return nil, fmt.Errorf("トピックの取得に失敗: %w", err)
	}
	if currentTopic == nil {
		return nil, fmt.Errorf("ページのトピックが見つかりません: page_id=%s, topic_id=%s", pg.ID, pg.TopicID)
	}

	availableTopics, err := uc.availableTopicsForMove(ctx, spaceMember, space, pg.TopicID)
	if err != nil {
		return nil, fmt.Errorf("移動先トピック一覧の取得に失敗: %w", err)
	}

	return &GetPageMoveDataOutput{
		Space:           space,
		SpaceMember:     spaceMember,
		Page:            pg,
		TopicMember:     topicMember,
		CurrentTopic:    currentTopic,
		AvailableTopics: availableTopics,
	}, nil
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
