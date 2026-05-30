package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// homeJoinedTopicsLimit is the upper bound on the number of joined topics surfaced on the
// home page. Kept aligned with viewmodel.SidebarJoinedTopicsLimit (= 10) so home and the
// sidebar show the same "topics the user has been active in" set; the value is duplicated
// here rather than imported because Application layer cannot depend on Presentation layer.
//
// [Ja] homeJoinedTopicsLimit はホーム画面に表示する参加中トピック数の上限。
// viewmodel.SidebarJoinedTopicsLimit (= 10) と揃えてあり、ホームとサイドバーで
// 「ユーザーが直近で動いたトピック」の同じセットを表示できるようにしている。
// Application 層は Presentation 層に依存できないため、import せずに値を二重定義している。
const homeJoinedTopicsLimit = 10

// homeDraftPagesLimit is the upper bound on the number of draft pages surfaced on the home
// page. Kept aligned with viewmodel.SidebarDraftPagesLimit (= 5) so home and the sidebar
// show the same set of recent drafts; duplicated here rather than imported because the
// Application layer cannot depend on the Presentation layer.
//
// [Ja] homeDraftPagesLimit はホーム画面に表示する下書きページ数の上限。
// viewmodel.SidebarDraftPagesLimit (= 5) と揃えてあり、ホームとサイドバーで同じ
// 「直近の下書き」を表示できるようにしている。Application 層は Presentation 層に依存できないため、
// import せずに値を二重定義している。
const homeDraftPagesLimit = 5

// GetHomeShowUsecase はホーム画面表示用ユースケース
type GetHomeShowUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
	draftPageRepo   *repository.DraftPageRepository
}

// NewGetHomeShowUsecase は GetHomeShowUsecase を生成する
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

// GetHomeShowInput はホーム画面表示の入力パラメータ
type GetHomeShowInput struct {
	UserID model.UserID
}

// GetHomeShowOutput contains the data shown on the home page: the user's active spaces,
// the topics the user is joined to, and the draft pages the user is working on.
//
// [Ja] GetHomeShowOutput はホーム画面に表示するデータ。ユーザーが参加中のスペース一覧、
// 参加中のトピック一覧、ユーザーが作業中の下書きページを保持する。
type GetHomeShowOutput struct {
	ActiveSpaces []*model.Space
	JoinedTopics []*model.Topic
	DraftPages   []*model.DraftPage

	// CanCreatePageByTopic reports, per joined topic id, whether the current user may create a page
	// in that topic (page:write scope). Joined topics span multiple spaces, so create permission is
	// resolved per topic rather than once for the whole page.
	//
	// [Ja] CanCreatePageByTopic は参加中トピックの id ごとに、現在のユーザーがそのトピックに
	// ページを作成できるか (page:write スコープ) を表す。参加中トピックは複数スペースに跨るため、
	// 作成権限はページ全体で一度ではなくトピックごとに判定する。
	CanCreatePageByTopic map[model.TopicID]bool
}

// Execute はホーム画面に表示する参加中スペース・参加中トピック・下書きページを取得する。
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

	// Resolve the per-topic page-create permission for the joined topics.
	// [Ja] 参加中トピックについて、トピックごとのページ作成権限を解決する。
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

// resolveCanCreatePageByTopic resolves, per joined topic, whether the user may create a page in it.
// Joined topics span multiple spaces, so the user holds a separate space membership per space and a
// separate topic membership per topic. Both are fetched with one bulk query each (keyed by ANY(...))
// to avoid an N+1 over topics, mirroring resolveSectionTopics in get_space_show.go. A user with a
// space-level page:write scope (e.g. space:admin) can create pages even in a topic without a
// per-topic membership, which newAuthorizer handles by merging space and topic scopes.
//
// [Ja] resolveCanCreatePageByTopic は参加中トピックごとに、ユーザーがそこにページを作成できるかを
// 解決する。参加中トピックは複数スペースに跨るため、ユーザーはスペースごとに別々のスペースメンバーを、
// トピックごとに別々のトピックメンバーを持つ。どちらも ANY(...) によるバルククエリ 1 回ずつで取得し、
// トピックに対する N+1 を避ける (get_space_show.go の resolveSectionTopics と同じ構造)。スペースレベルの
// page:write スコープ (例: space:admin) を持つユーザーは、トピックメンバーでなくてもページを作成できる。
// これは newAuthorizer がスペーススコープとトピックスコープを統合して扱う。
func (uc *GetHomeShowUsecase) resolveCanCreatePageByTopic(
	ctx context.Context,
	userID model.UserID,
	joinedTopics []*model.Topic,
) (map[model.TopicID]bool, error) {
	if len(joinedTopics) == 0 {
		return map[model.TopicID]bool{}, nil
	}

	// Collect the distinct space ids and the topic ids across the joined topics.
	// [Ja] 参加中トピックからスペース id (重複排除) とトピック id を集める。
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

	// Fetch the user's active space memberships across all spaces in one query, keyed by space id.
	// [Ja] 全スペースにまたがるユーザーのアクティブなスペースメンバーを 1 クエリで取得し、スペース id で引けるようにする。
	spaceMembers, err := uc.spaceMemberRepo.ListActiveByUserAndSpaceIDs(ctx, userID, spaceIDs)
	if err != nil {
		return nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}
	spaceMemberBySpace := make(map[model.SpaceID]*model.SpaceMember, len(spaceMembers))
	for _, spaceMember := range spaceMembers {
		spaceMemberBySpace[spaceMember.SpaceID] = spaceMember
	}

	// Fetch the user's topic memberships across all topics in one query.
	// [Ja] 全トピックにまたがるユーザーのトピックメンバーを 1 クエリで取得する。
	topicMembers, err := uc.topicMemberRepo.ListByUserAndTopics(ctx, userID, spaceIDs, topicIDs)
	if err != nil {
		return nil, fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
	}

	// Resolve each topic against the space membership for its own space, since joined topics span spaces.
	// [Ja] 参加中トピックは複数スペースに跨るため、各トピックをそのスペースのスペースメンバーで判定する。
	return buildCanCreatePageByTopic(joinedTopics, topicMembers, func(topic *model.Topic) *model.SpaceMember {
		return spaceMemberBySpace[topic.Space.ID]
	}), nil
}
