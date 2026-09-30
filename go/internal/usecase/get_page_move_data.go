package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetPageMoveDataUsecaseはページ移動フォームのデータ取得ユースケース
type GetPageMoveDataUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// NewGetPageMoveDataUsecaseはGetPageMoveDataUsecaseを生成する
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

// GetPageMoveDataInputはページ移動データ取得の入力パラメータ
type GetPageMoveDataInput struct {
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	UserID          model.UserID
}

// GetPageMoveDataOutputはページ移動データ取得の出力
type GetPageMoveDataOutput struct {
	Space           *model.Space
	SpaceMember     *model.SpaceMember
	Page            *model.Page
	TopicMember     *model.TopicMember
	CurrentTopic    *model.Topic
	AvailableTopics []*model.Topic
}

// Executeはページ移動フォームに必要なデータを取得する
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

// availableTopicsForMoveは移動先候補のトピック一覧を取得する。
// 候補はページを作成できるトピックに限り、現在のトピックは除外する。
// スペースのロールでページを作成できるメンバーは全アクティブトピックを、それ以外は所属トピックの
// うちトピックのロールでページを作成できるものを返す。移動の確定時もvalidatorが同じ判定をする。
func (uc *GetPageMoveDataUsecase) availableTopicsForMove(
	ctx context.Context,
	spaceMember *model.SpaceMember,
	space *model.Space,
	currentTopicID model.TopicID,
) ([]*model.Topic, error) {
	var topics []*model.Topic
	var err error

	if newAuthorizer(spaceMember, nil).CanCreatePage() {
		topics, err = uc.topicRepo.ListActiveBySpace(ctx, space.ID)
	} else {
		topics, err = uc.writableJoinedTopics(ctx, spaceMember, space)
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

// writableJoinedTopicsは、所属トピックのうち、トピックのロールでページを作成できるものを返す
func (uc *GetPageMoveDataUsecase) writableJoinedTopics(ctx context.Context, spaceMember *model.SpaceMember, space *model.Space) ([]*model.Topic, error) {
	joined, err := uc.topicRepo.ListJoinedBySpaceMember(ctx, spaceMember.ID, space.ID)
	if err != nil {
		return nil, err
	}
	if len(joined) == 0 {
		return nil, nil
	}

	topicIDs := make([]model.TopicID, len(joined))
	for i, t := range joined {
		topicIDs[i] = t.ID
	}
	topicMembers, err := uc.topicMemberRepo.ListBySpaceMemberAndTopics(ctx, space.ID, spaceMember.ID, topicIDs)
	if err != nil {
		return nil, err
	}
	topicMemberByTopic := make(map[model.TopicID]*model.TopicMember, len(topicMembers))
	for _, tm := range topicMembers {
		topicMemberByTopic[tm.TopicID] = tm
	}

	var topics []*model.Topic
	for _, t := range joined {
		if newAuthorizer(spaceMember, topicMemberByTopic[t.ID]).CanCreatePage() {
			topics = append(topics, t)
		}
	}
	return topics, nil
}
