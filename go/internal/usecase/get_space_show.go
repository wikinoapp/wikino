package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetSpaceShowUsecaseはスペース詳細画面 (GET /s/:identifier) に表示するデータを集約する
// 読み取りUseCase。スペース本体、スペース内のピン留めページと通常ページ、空状態に必要なデータを取得する。
type GetSpaceShowUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// NewGetSpaceShowUsecaseはGetSpaceShowUsecaseを生成する。
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

// GetSpaceShowInputはスペース詳細取得の入力パラメータ。
// UserIDは未ログイン時にnilになる (スペース詳細は非ログインでも公開トピックなら閲覧できる)。
type GetSpaceShowInput struct {
	SpaceIdentifier model.SpaceIdentifier
	UserID          *model.UserID
	Page            int32
	PageLimit       int32
}

// GetSpaceShowOutputはスペース詳細取得の出力。
type GetSpaceShowOutput struct {
	Space       *model.Space
	SpaceMember *model.SpaceMember
	PinnedPages []*model.Page
	Pages       []*model.Page
	TotalCount  int64

	// TopicMapは一覧する各ページのtopic idをトピックへ対応付け、カードにトピックラベルを
	// 表示できるようにする。スペース詳細はトピック詳細と違いページが複数トピックに跨る。
	TopicMap map[model.TopicID]*model.Topic

	// CanEditPageByTopicはtopic idごとに、現在のユーザーがそのトピックのページを編集できるか
	// (page:writeスコープ) を表す。ゲストでは空。スペース詳細は複数トピックに跨るため、編集権限は
	// ページ全体で一度ではなくトピックごとに判定する。
	CanEditPageByTopic map[model.TopicID]bool

	// SectionTopicsはトピックセクションに表示するトピック。メンバーは参加中のトピック、
	// 非メンバー・ゲストはスペースの公開トピック。各トピックはトピック詳細へリンクし、
	// CanCreatePageByTopicがtrueのトピックではトピックごとの「新規ページ」アクションを出す。
	// これはスペースレベルの空状態「新規ページ」ボタン (作成先トピックを暗黙に固定していた) を置き換える。
	SectionTopics []*model.Topic

	// CanCreatePageByTopicはセクショントピックのidごとに、現在のユーザーがそのトピックに
	// ページを作成できるか (page:writeスコープ) を表す。ゲストでは空。トピックセクションは値がtrueの
	// トピックにのみトピックごとの「新規ページ」アクションを出す。
	CanCreatePageByTopic map[model.TopicID]bool

	// FirstJoinedTopicはメンバーが参加しているトピックのうちidが最小のもの (ゲスト、
	// またはどのトピックにも参加していないメンバーではnil)。空状態の「新しいページを作る」導線で使う。
	FirstJoinedTopic *model.Topic

	JoinedSpace    bool
	CanCreateTopic bool
}

// Executeはスペース詳細画面に表示するデータを取得する。スペースが見つからない場合は (nil, nil) を返す。
func (uc *GetSpaceShowUsecase) Execute(ctx context.Context, input GetSpaceShowInput) (*GetSpaceShowOutput, error) {
	space, err := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
	if err != nil {
		return nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, nil
	}

	// ログインユーザーのスペースメンバーを取得 (未ログインならnil)。
	var spaceMember *model.SpaceMember
	if input.UserID != nil {
		spaceMember, err = uc.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, *input.UserID)
		if err != nil {
			return nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
		}
	}

	// メンバーは全アクティブページを閲覧でき、非メンバー (ゲスト含む) は公開トピックのページのみ閲覧できる。
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

	// 一覧するページのトピックラベルとトピックごとのページ編集権限を解決する。
	topicMap, canEditPageByTopic, err := uc.resolvePageTopicViews(ctx, space.ID, spaceMember, pinnedPages, paginatedResult.Pages)
	if err != nil {
		return nil, err
	}

	// トピックセクションに表示するトピックと、トピックごとのページ作成権限を解決する。
	sectionTopics, canCreatePageByTopic, err := uc.resolveSectionTopics(ctx, space.ID, spaceMember)
	if err != nil {
		return nil, err
	}

	// 空状態で「新しいページを作る」導線を出せるよう、メンバーには常に最初の参加トピックを取得する。
	// 使うのはページが0件のときだけだが、取得を空状態判定でゲートしていない。Rails版と挙動を揃え、
	// 取得条件をテンプレートの空状態判定と結合させないため (コストはインデックス済みのLIMIT 1クエリ1回)。
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

// resolvePageTopicViewsは与えられたページ群について、カードラベル用のトピックマップと
// トピックごとのページ編集権限マップを構築する。クエリ回数はトピック数に依らず一定で、トピックの
// 一括取得に1回、(メンバーの場合) トピックメンバーの一括取得に1回だけ実行し、トピックに対する
// N+1を避ける。編集権限はトピックごとに判定し、編集できないゲストではスキップする。スペースレベルの
// page:writeスコープ (例: space:admin) を持つメンバーは、トピックメンバーでなくても全トピックの
// ページを編集できる。これはnewAuthorizerがスペーススコープとトピックスコープを統合して扱う。
func (uc *GetSpaceShowUsecase) resolvePageTopicViews(
	ctx context.Context,
	spaceID model.SpaceID,
	spaceMember *model.SpaceMember,
	pageGroups ...[]*model.Page,
) (map[model.TopicID]*model.Topic, map[model.TopicID]bool, error) {
	// 全ページ群から重複のないトピックidを集める。
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

	// ゲストは編集できないため、権限マップは空のままにする (参照はfalseになる)。
	if spaceMember == nil {
		return topicMap, canEditPageByTopic, nil
	}

	// 一覧トピック全てのトピックメンバーを1クエリで取得し、トピックごとに編集権限を判定する。
	// メンバーシップが無いトピックはtopicMemberがnilになるが、スペーススコープだけでpage:writeを
	// 含む場合はnilでも編集可能になる。
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

// resolveSectionTopicsはスペース詳細のトピックセクションに表示するトピックを取得し、
// トピックごとに現在のユーザーがそこにページを作成できるかを解決する。メンバーは参加中のトピックを、
// 非メンバー (ゲスト含む) は公開トピックのみを見る。トピックごとの作成権限はトピックメンバーの
// 一括取得1回で解決し、トピックに対するN+1を避ける。ゲストはページを作成できないため空になる。
// スペースレベルのpage:writeスコープ (例: space:admin) を持つメンバーは、トピックメンバーで
// なくてもページを作成できる。これはnewAuthorizerがスペーススコープとトピックスコープを統合して扱う。
func (uc *GetSpaceShowUsecase) resolveSectionTopics(
	ctx context.Context,
	spaceID model.SpaceID,
	spaceMember *model.SpaceMember,
) ([]*model.Topic, map[model.TopicID]bool, error) {
	// ゲスト (およびログイン済み非メンバー) は公開トピックのみを見てページを作成できないため、
	// 権限マップは空のままにする (参照はfalseになる)。
	if spaceMember == nil {
		topics, err := uc.topicRepo.ListPublicBySpace(ctx, spaceID)
		if err != nil {
			return nil, nil, fmt.Errorf("公開トピックの取得に失敗: %w", err)
		}
		return topics, map[model.TopicID]bool{}, nil
	}

	// メンバーは参加中のトピックを見る。
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

	// セクションの全トピックのトピックメンバーを1クエリで取得し、トピックごとに
	// ページ作成権限を判定する。
	topicMembers, err := uc.topicMemberRepo.ListBySpaceMemberAndTopics(ctx, spaceID, spaceMember.ID, topicIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
	}

	// セクションのトピックはすべて同一スペースに属するため、各トピックは同じメンバーで判定する。
	canCreatePageByTopic := buildCanCreatePageByTopic(topics, topicMembers, func(*model.Topic) *model.SpaceMember {
		return spaceMember
	})

	return topics, canCreatePageByTopic, nil
}
