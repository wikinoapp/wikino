package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetTrashShowUsecaseは、スペースのゴミ箱の一覧 (GET /s/{space_identifier}/trash) に出すデータを集める。
type GetTrashShowUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// NewGetTrashShowUsecaseはGetTrashShowUsecaseを生成する。
func NewGetTrashShowUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
) *GetTrashShowUsecase {
	return &GetTrashShowUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		pageRepo:        pageRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
	}
}

// GetTrashShowInputはゴミ箱の一覧の取得の入力。
type GetTrashShowInput struct {
	SpaceIdentifier model.SpaceIdentifier
	UserID          model.UserID
	Page            int32
	PageLimit       int32

	// Nowは復元できる期間の起点にする現在時刻。
	Now time.Time
}

// GetTrashShowOutputはゴミ箱の一覧の取得の出力。
type GetTrashShowOutput struct {
	Space      *model.Space
	Pages      []*model.Page
	TotalCount int64

	// TopicMapは一覧の各ページのトピックを引けるようにする。ゴミ箱の一覧はスペースの
	// 複数のトピックのページを並べ、行ごとにトピック名を出すため。
	TopicMap map[model.TopicID]*model.Topic
}

// Executeは、閲覧者が開けてゴミ箱を見られる (page_trash:read) トピックの、復元できるページを返す。
//
// ゴミ箱を見られるかはトピックごとに判定する。どのトピックでも見られない閲覧者 (スペースの
// メンバーでない閲覧者を含む) にはForbiddenのAppErrorを返し、ゴミ箱があることを伝えない。
func (uc *GetTrashShowUsecase) Execute(ctx context.Context, input GetTrashShowInput) (*GetTrashShowOutput, error) {
	space, err := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
	if err != nil {
		return nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, notFoundError(ctx)
	}

	spaceMember, err := uc.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}
	if spaceMember == nil {
		return nil, forbiddenError(ctx)
	}

	access, err := loadTopicAccess(ctx, uc.topicRepo, uc.topicMemberRepo, space.ID, spaceMember, func(topicMember *model.TopicMember) policy.Authorizer {
		return newAuthorizer(spaceMember, topicMember)
	})
	if err != nil {
		return nil, err
	}

	trashTopics := access.trashVisibleTopics()
	if len(trashTopics) == 0 {
		return nil, forbiddenError(ctx)
	}

	topicIDs := make([]model.TopicID, len(trashTopics))
	topicMap := make(map[model.TopicID]*model.Topic, len(trashTopics))
	for i, topic := range trashTopics {
		topicIDs[i] = topic.ID
		topicMap[topic.ID] = topic
	}

	trashedSince := input.Now.AddDate(0, 0, -model.PageTrashRetentionDays)
	result, err := uc.pageRepo.FindRestorablePaginated(ctx, space.ID, topicIDs, trashedSince, input.Page, input.PageLimit)
	if err != nil {
		return nil, fmt.Errorf("ゴミ箱のページの取得に失敗: %w", err)
	}

	return &GetTrashShowOutput{
		Space:      space,
		Pages:      result.Pages,
		TotalCount: result.TotalCount,
		TopicMap:   topicMap,
	}, nil
}
