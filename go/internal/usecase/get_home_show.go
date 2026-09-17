package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// homeJoinedTopicsLimitはホーム画面に表示する参加中トピック数の上限。
// ユーザーが直近で動いたトピックにセクションを絞る。
const homeJoinedTopicsLimit = 10

// homeDraftPagesLimitはホーム画面に表示する下書きページ数の上限。
// ユーザーの直近の下書きにセクションを絞る。
const homeDraftPagesLimit = 5

// GetHomeShowUsecaseはホーム画面表示用ユースケース
type GetHomeShowUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
	draftPageRepo   *repository.DraftPageRepository
}

// NewGetHomeShowUsecaseはGetHomeShowUsecaseを生成する
func NewGetHomeShowUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	draftPageRepo *repository.DraftPageRepository,
) *GetHomeShowUsecase {
	return &GetHomeShowUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
		draftPageRepo:   draftPageRepo,
	}
}

// GetHomeShowInputはホーム画面表示の入力パラメータ
type GetHomeShowInput struct {
	UserID model.UserID
}

// GetHomeShowOutputはホーム画面に表示するデータ。ユーザーが参加中のスペース一覧、
// 参加中のトピック一覧、ユーザーが作業中の下書きページを保持する。
type GetHomeShowOutput struct {
	ActiveSpaces []*model.Space
	JoinedTopics []*model.Topic
	DraftPages   []*model.DraftPage

	// CanCreatePageByTopicは参加中トピックのidごとに、現在のユーザーがそのトピックに
	// ページを作成できるか (page:writeスコープ) を表す。参加中トピックは複数スペースに跨るため、
	// 作成権限はページ全体で一度ではなくトピックごとに判定する。
	CanCreatePageByTopic map[model.TopicID]bool
}

// Executeはホーム画面に表示する参加中スペース・参加中トピック・下書きページを取得する。
func (uc *GetHomeShowUsecase) Execute(ctx context.Context, input GetHomeShowInput) (*GetHomeShowOutput, error) {
	spaces, err := uc.spaceRepo.ListActiveByUser(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("参加中スペース一覧の取得に失敗: %w", err)
	}

	joinedTopics, err := uc.topicRepo.ListJoinedByUser(ctx, input.UserID, homeJoinedTopicsLimit)
	if err != nil {
		return nil, fmt.Errorf("参加中トピック一覧の取得に失敗: %w", err)
	}

	drafts, err := uc.draftPageRepo.ListByUser(ctx, input.UserID, homeDraftPagesLimit)
	if err != nil {
		return nil, fmt.Errorf("下書きページ一覧の取得に失敗: %w", err)
	}

	// 参加中トピックについて、トピックごとのページ作成権限を解決する。
	canCreatePageByTopic, err := uc.resolveCanCreatePageByTopic(ctx, input.UserID, joinedTopics)
	if err != nil {
		return nil, err
	}

	return &GetHomeShowOutput{
		ActiveSpaces:         spaces,
		JoinedTopics:         joinedTopics,
		DraftPages:           drafts,
		CanCreatePageByTopic: canCreatePageByTopic,
	}, nil
}

// resolveCanCreatePageByTopicは参加中トピックごとに、ユーザーがそこにページを作成できるかを
// 解決する。参加中トピックは複数スペースに跨るため、ユーザーはスペースごとに別々のスペースメンバーを、
// トピックごとに別々のトピックメンバーを持つ。どちらもANY(...) によるバルククエリ1回ずつで取得し、
// トピックに対するN+1を避ける (get_space_show.goのresolveSectionTopicsと同じ構造)。スペースレベルの
// page:writeスコープ (例: space:admin) を持つユーザーは、トピックメンバーでなくてもページを作成できる。
// これはnewAuthorizerがスペーススコープとトピックスコープを統合して扱う。
func (uc *GetHomeShowUsecase) resolveCanCreatePageByTopic(
	ctx context.Context,
	userID model.UserID,
	joinedTopics []*model.Topic,
) (map[model.TopicID]bool, error) {
	if len(joinedTopics) == 0 {
		return map[model.TopicID]bool{}, nil
	}

	// 参加中トピックからスペースid (重複排除) とトピックidを集める。
	spaceIDSet := make(map[model.SpaceID]struct{}, len(joinedTopics))
	topicIDs := make([]model.TopicID, 0, len(joinedTopics))
	for _, topic := range joinedTopics {
		spaceIDSet[topic.Space.ID] = struct{}{}
		topicIDs = append(topicIDs, topic.ID)
	}
	spaceIDs := make([]model.SpaceID, 0, len(spaceIDSet))
	for spaceID := range spaceIDSet {
		spaceIDs = append(spaceIDs, spaceID)
	}

	// 全スペースにまたがるユーザーのアクティブなスペースメンバーを1クエリで取得し、スペースidで引けるようにする。
	spaceMembers, err := uc.spaceMemberRepo.ListActiveByUserAndSpaceIDs(ctx, userID, spaceIDs)
	if err != nil {
		return nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}
	spaceMemberBySpace := make(map[model.SpaceID]*model.SpaceMember, len(spaceMembers))
	for _, spaceMember := range spaceMembers {
		spaceMemberBySpace[spaceMember.SpaceID] = spaceMember
	}

	// 全トピックにまたがるユーザーのトピックメンバーを1クエリで取得する。
	topicMembers, err := uc.topicMemberRepo.ListByUserAndTopics(ctx, userID, spaceIDs, topicIDs)
	if err != nil {
		return nil, fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
	}

	// 参加中トピックは複数スペースに跨るため、各トピックをそのスペースのスペースメンバーで判定する。
	return buildCanCreatePageByTopic(joinedTopics, topicMembers, func(topic *model.Topic) *model.SpaceMember {
		return spaceMemberBySpace[topic.Space.ID]
	}), nil
}
