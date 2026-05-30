package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetSpaceShowUsecase aggregates the data shown on the space detail page (GET /s/:identifier):
// the space itself, the pinned and regular pages within it, and the data the empty state needs.
//
// [Ja] GetSpaceShowUsecase はスペース詳細画面 (GET /s/:identifier) に表示するデータを集約する
// 読み取り UseCase。スペース本体、スペース内のピン留めページと通常ページ、空状態に必要なデータを取得する。
type GetSpaceShowUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// NewGetSpaceShowUsecase creates a GetSpaceShowUsecase.
// [Ja] NewGetSpaceShowUsecase は GetSpaceShowUsecase を生成する。
func NewGetSpaceShowUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
) *GetSpaceShowUsecase {
	return &GetSpaceShowUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		pageRepo:        pageRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
	}
}

// GetSpaceShowInput holds the input parameters for fetching the space detail.
// UserID is nil when the user is not logged in (the space detail is viewable even
// without logging in as long as the topic is public).
//
// [Ja] GetSpaceShowInput はスペース詳細取得の入力パラメータ。
// UserID は未ログイン時に nil になる (スペース詳細は非ログインでも公開トピックなら閲覧できる)。
type GetSpaceShowInput struct {
	SpaceIdentifier model.SpaceIdentifier
	UserID          *model.UserID
	Page            int32
	PageLimit       int32
}

// GetSpaceShowOutput is the output of fetching the space detail.
// [Ja] GetSpaceShowOutput はスペース詳細取得の出力。
type GetSpaceShowOutput struct {
	Space       *model.Space
	SpaceMember *model.SpaceMember
	PinnedPages []*model.Page
	Pages       []*model.Page
	TotalCount  int64

	// TopicMap maps each listed page's topic id to its topic, so the cards can show a topic
	// label. Pages on the space detail span multiple topics, unlike the topic detail page.
	//
	// [Ja] TopicMap は一覧する各ページの topic id をトピックへ対応付け、カードにトピックラベルを
	// 表示できるようにする。スペース詳細はトピック詳細と違いページが複数トピックに跨る。
	TopicMap map[model.TopicID]*model.Topic

	// CanEditPageByTopic reports, per topic id, whether the current user may edit pages in that
	// topic (page:write scope). It is empty for guests. The space detail spans multiple topics,
	// so edit permission is resolved per topic rather than once for the whole page.
	//
	// [Ja] CanEditPageByTopic は topic id ごとに、現在のユーザーがそのトピックのページを編集できるか
	// (page:write スコープ) を表す。ゲストでは空。スペース詳細は複数トピックに跨るため、編集権限は
	// ページ全体で一度ではなくトピックごとに判定する。
	CanEditPageByTopic map[model.TopicID]bool

	// SectionTopics are the topics shown in the topic section: the topics the member has joined,
	// or (for non-members and guests) the space's public topics. Each links to its topic detail and,
	// where CanCreatePageByTopic is true, offers a per-topic "new page" action. This replaces the
	// space-level empty-state "new page" button, which implicitly fixed the destination topic.
	//
	// [Ja] SectionTopics はトピックセクションに表示するトピック。メンバーは参加中のトピック、
	// 非メンバー・ゲストはスペースの公開トピック。各トピックはトピック詳細へリンクし、
	// CanCreatePageByTopic が true のトピックではトピックごとの「新規ページ」アクションを出す。
	// これはスペースレベルの空状態「新規ページ」ボタン (作成先トピックを暗黙に固定していた) を置き換える。
	SectionTopics []*model.Topic

	// CanCreatePageByTopic reports, per section topic id, whether the current user may create a page
	// in that topic (page:write scope). It is empty for guests. The topic section shows the per-topic
	// "new page" action only for topics whose value is true.
	//
	// [Ja] CanCreatePageByTopic はセクショントピックの id ごとに、現在のユーザーがそのトピックに
	// ページを作成できるか (page:write スコープ) を表す。ゲストでは空。トピックセクションは値が true の
	// トピックにのみトピックごとの「新規ページ」アクションを出す。
	CanCreatePageByTopic map[model.TopicID]bool

	// FirstJoinedTopic is the member's joined topic with the smallest id (nil for guests or for
	// members who have not joined any topic). Used by the empty-state "create a new page" link.
	//
	// [Ja] FirstJoinedTopic はメンバーが参加しているトピックのうち id が最小のもの (ゲスト、
	// またはどのトピックにも参加していないメンバーでは nil)。空状態の「新しいページを作る」導線で使う。
	FirstJoinedTopic *model.Topic

	JoinedSpace    bool
	CanCreateTopic bool
}

// Execute fetches the data shown on the space detail page. It returns (nil, nil)
// when the space is not found.
//
// [Ja] Execute はスペース詳細画面に表示するデータを取得する。スペースが見つからない場合は (nil, nil) を返す。
func (uc *GetSpaceShowUsecase) Execute(ctx context.Context, input GetSpaceShowInput) (*GetSpaceShowOutput, error) {
	space, err := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
	if err != nil {
		return nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, nil
	}

	// Fetch the logged-in user's space membership (nil when not logged in).
	// [Ja] ログインユーザーのスペースメンバーを取得 (未ログインなら nil)。
	var spaceMember *model.SpaceMember
	if input.UserID != nil {
		spaceMember, err = uc.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, *input.UserID)
		if err != nil {
			return nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
		}
	}

	// Members see every active page; non-members (guests included) see only public-topic pages.
	// [Ja] メンバーは全アクティブページを閲覧でき、非メンバー (ゲスト含む) は公開トピックのページのみ閲覧できる。
	joinedSpace := spaceMember != nil
	publicOnly := !joinedSpace

	authorizer := newAuthorizer(spaceMember, nil)

	pinnedPages, err := uc.pageRepo.FindPinnedBySpace(ctx, space.ID, publicOnly)
	if err != nil {
		return nil, fmt.Errorf("ピン留めページの取得に失敗: %w", err)
	}

	paginatedResult, err := uc.pageRepo.FindRegularBySpacePaginated(ctx, space.ID, publicOnly, input.Page, input.PageLimit)
	if err != nil {
		return nil, fmt.Errorf("通常ページの取得に失敗: %w", err)
	}

	// Resolve the topic label and per-topic page-edit permission for the listed pages.
	// [Ja] 一覧するページのトピックラベルとトピックごとのページ編集権限を解決する。
	topicMap, canEditPageByTopic, err := uc.resolvePageTopicViews(ctx, space.ID, spaceMember, pinnedPages, paginatedResult.Pages)
	if err != nil {
		return nil, err
	}

	// Resolve the topics shown in the topic section and the per-topic page-create permission.
	// [Ja] トピックセクションに表示するトピックと、トピックごとのページ作成権限を解決する。
	sectionTopics, canCreatePageByTopic, err := uc.resolveSectionTopics(ctx, space.ID, spaceMember)
	if err != nil {
		return nil, err
	}

	// Fetch the first joined topic for every member so the empty state can offer a "new page"
	// link. It is consumed only when no pages are shown, but the fetch is not gated on emptiness:
	// that keeps this in step with the Rails version and avoids tying the fetch condition to the
	// template's empty-state check (the cost is one indexed LIMIT 1 lookup).
	//
	// [Ja] 空状態で「新しいページを作る」導線を出せるよう、メンバーには常に最初の参加トピックを取得する。
	// 使うのはページが 0 件のときだけだが、取得を空状態判定でゲートしていない。Rails 版と挙動を揃え、
	// 取得条件をテンプレートの空状態判定と結合させないため (コストはインデックス済みの LIMIT 1 クエリ 1 回)。
	var firstJoinedTopic *model.Topic
	if spaceMember != nil {
		firstJoinedTopic, err = uc.topicRepo.FindFirstJoinedBySpaceMember(ctx, spaceMember.ID, space.ID)
		if err != nil {
			return nil, fmt.Errorf("最初の参加トピックの取得に失敗: %w", err)
		}
	}

	return &GetSpaceShowOutput{
		Space:                space,
		SpaceMember:          spaceMember,
		PinnedPages:          pinnedPages,
		Pages:                paginatedResult.Pages,
		TotalCount:           paginatedResult.TotalCount,
		TopicMap:             topicMap,
		CanEditPageByTopic:   canEditPageByTopic,
		SectionTopics:        sectionTopics,
		CanCreatePageByTopic: canCreatePageByTopic,
		FirstJoinedTopic:     firstJoinedTopic,
		JoinedSpace:          joinedSpace,
		CanCreateTopic:       authorizer.CanCreateTopic(),
	}, nil
}

// resolvePageTopicViews builds the topic map (for card labels) and the per-topic page-edit
// permission map for the given page groups. It runs a constant number of queries regardless of
// the number of topics: one to fetch the topics by id, and (for members) one to fetch the
// member's topic memberships in bulk, avoiding an N+1 over topics. Edit permission is resolved
// per distinct topic and skipped entirely for guests since they cannot edit. A member with a
// space-level page:write scope (e.g. space:admin) can edit pages in every topic even without a
// per-topic membership, which newAuthorizer handles by merging space and topic scopes.
//
// [Ja] resolvePageTopicViews は与えられたページ群について、カードラベル用のトピックマップと
// トピックごとのページ編集権限マップを構築する。クエリ回数はトピック数に依らず一定で、トピックの
// 一括取得に 1 回、(メンバーの場合) トピックメンバーの一括取得に 1 回だけ実行し、トピックに対する
// N+1 を避ける。編集権限はトピックごとに判定し、編集できないゲストではスキップする。スペースレベルの
// page:write スコープ (例: space:admin) を持つメンバーは、トピックメンバーでなくても全トピックの
// ページを編集できる。これは newAuthorizer がスペーススコープとトピックスコープを統合して扱う。
func (uc *GetSpaceShowUsecase) resolvePageTopicViews(
	ctx context.Context,
	spaceID model.SpaceID,
	spaceMember *model.SpaceMember,
	pageGroups ...[]*model.Page,
) (map[model.TopicID]*model.Topic, map[model.TopicID]bool, error) {
	// Collect the distinct topic ids across all page groups.
	// [Ja] 全ページ群から重複のないトピック id を集める。
	topicIDSet := make(map[model.TopicID]struct{})
	for _, pages := range pageGroups {
		for _, pg := range pages {
			topicIDSet[pg.TopicID] = struct{}{}
		}
	}

	topicMap := make(map[model.TopicID]*model.Topic, len(topicIDSet))
	canEditPageByTopic := make(map[model.TopicID]bool, len(topicIDSet))
	if len(topicIDSet) == 0 {
		return topicMap, canEditPageByTopic, nil
	}

	topicIDs := make([]model.TopicID, 0, len(topicIDSet))
	for topicID := range topicIDSet {
		topicIDs = append(topicIDs, topicID)
	}

	topics, err := uc.topicRepo.FindByIDsAndSpace(ctx, topicIDs, spaceID)
	if err != nil {
		return nil, nil, fmt.Errorf("ページのトピックの取得に失敗: %w", err)
	}
	for _, topic := range topics {
		topicMap[topic.ID] = topic
	}

	// Guests cannot edit, so leave the permission map empty (lookups default to false).
	// [Ja] ゲストは編集できないため、権限マップは空のままにする (参照は false になる)。
	if spaceMember == nil {
		return topicMap, canEditPageByTopic, nil
	}

	// Fetch the member's topic memberships for all listed topics in one query, then resolve the
	// page-edit permission per topic. Topics where the member has no membership get a nil
	// topicMember, which still grants edit access when the space scope alone includes page:write.
	//
	// [Ja] 一覧トピック全てのトピックメンバーを 1 クエリで取得し、トピックごとに編集権限を判定する。
	// メンバーシップが無いトピックは topicMember が nil になるが、スペーススコープだけで page:write を
	// 含む場合は nil でも編集可能になる。
	topicMembers, err := uc.topicMemberRepo.ListBySpaceMemberAndTopics(ctx, spaceID, spaceMember.ID, topicIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
	}
	topicMemberByTopic := make(map[model.TopicID]*model.TopicMember, len(topicMembers))
	for _, topicMember := range topicMembers {
		topicMemberByTopic[topicMember.TopicID] = topicMember
	}

	for _, topicID := range topicIDs {
		canEditPageByTopic[topicID] = newAuthorizer(spaceMember, topicMemberByTopic[topicID]).CanUpdatePage()
	}

	return topicMap, canEditPageByTopic, nil
}

// resolveSectionTopics fetches the topics shown in the space detail's topic section and resolves,
// per topic, whether the current user may create a page in it. Members see the topics they have
// joined; non-members (guests included) see only public topics. The per-topic create permission is
// resolved with one bulk fetch of the member's topic memberships to avoid an N+1 over topics, and
// is empty for guests since they cannot create pages. A member with a space-level page:write scope
// (e.g. space:admin) can create pages even in a topic without a per-topic membership, which
// newAuthorizer handles by merging space and topic scopes.
//
// [Ja] resolveSectionTopics はスペース詳細のトピックセクションに表示するトピックを取得し、
// トピックごとに現在のユーザーがそこにページを作成できるかを解決する。メンバーは参加中のトピックを、
// 非メンバー (ゲスト含む) は公開トピックのみを見る。トピックごとの作成権限はトピックメンバーの
// 一括取得 1 回で解決し、トピックに対する N+1 を避ける。ゲストはページを作成できないため空になる。
// スペースレベルの page:write スコープ (例: space:admin) を持つメンバーは、トピックメンバーで
// なくてもページを作成できる。これは newAuthorizer がスペーススコープとトピックスコープを統合して扱う。
func (uc *GetSpaceShowUsecase) resolveSectionTopics(
	ctx context.Context,
	spaceID model.SpaceID,
	spaceMember *model.SpaceMember,
) ([]*model.Topic, map[model.TopicID]bool, error) {
	// Guests (and logged-in non-members) see only public topics and cannot create pages, so leave
	// the permission map empty (lookups default to false).
	//
	// [Ja] ゲスト (およびログイン済み非メンバー) は公開トピックのみを見てページを作成できないため、
	// 権限マップは空のままにする (参照は false になる)。
	if spaceMember == nil {
		topics, err := uc.topicRepo.ListPublicBySpace(ctx, spaceID)
		if err != nil {
			return nil, nil, fmt.Errorf("公開トピックの取得に失敗: %w", err)
		}
		return topics, map[model.TopicID]bool{}, nil
	}

	// Members see the topics they have joined.
	// [Ja] メンバーは参加中のトピックを見る。
	topics, err := uc.topicRepo.ListJoinedBySpaceMember(ctx, spaceMember.ID, spaceID)
	if err != nil {
		return nil, nil, fmt.Errorf("参加トピックの取得に失敗: %w", err)
	}

	if len(topics) == 0 {
		return topics, map[model.TopicID]bool{}, nil
	}

	topicIDs := make([]model.TopicID, len(topics))
	for i, topic := range topics {
		topicIDs[i] = topic.ID
	}

	// Fetch the member's topic memberships for all section topics in one query, then resolve the
	// page-create permission per topic.
	//
	// [Ja] セクションの全トピックのトピックメンバーを 1 クエリで取得し、トピックごとに
	// ページ作成権限を判定する。
	topicMembers, err := uc.topicMemberRepo.ListBySpaceMemberAndTopics(ctx, spaceID, spaceMember.ID, topicIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
	}

	// All section topics belong to this single space, so every topic resolves against the same member.
	// [Ja] セクションのトピックはすべて同一スペースに属するため、各トピックは同じメンバーで判定する。
	canCreatePageByTopic := buildCanCreatePageByTopic(topics, topicMembers, func(*model.Topic) *model.SpaceMember {
		return spaceMember
	})

	return topics, canCreatePageByTopic, nil
}
