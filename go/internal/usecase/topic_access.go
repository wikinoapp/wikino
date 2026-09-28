package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// topicAccessはスペース内のアクティブなトピックと閲覧者のトピックメンバーを保持し、
// 一覧全体と個別ページの双方について「この閲覧者が開けるか」を判定する。
//
// 関連一覧 (リンク一覧・バックリンク一覧) と、その一覧が指すページを1つの解決結果から判定する
// ため、閲覧者が開けないページがタイトルだけでも一覧に現れることがなくなる。
type topicAccess struct {
	// authorizeはトピックメンバーから、そのトピックにおける閲覧者のAuthorizerを生成する
	authorize     func(topicMember *model.TopicMember) policy.Authorizer
	topics        []*model.Topic
	topicByID     map[model.TopicID]*model.Topic
	memberByTopic map[model.TopicID]*model.TopicMember
}

// fetchTopicAccessはスペースのトピックと閲覧者のトピックメンバーを解決する。
// どちらもそれぞれ1クエリで取得するため、一覧のページ数に比例してクエリが増えることはない。
// ゲストはトピックメンバーを持たないため、その取得自体を行わない。
func fetchTopicAccess(ctx context.Context, repos pageAccessRepos, spaceID model.SpaceID, spaceMember *model.SpaceMember) (*topicAccess, error) {
	return loadTopicAccess(ctx, repos.topicRepo, repos.topicMemberRepo, spaceID, spaceMember, func(topicMember *model.TopicMember) policy.Authorizer {
		return newAuthorizer(spaceMember, topicMember)
	})
}

// fetchAPITopicAccessは公開APIのトークンの主体について、束縛先のスペースのトピックと
// 主体のトピックメンバーを解決する。判定はメンバーのスコープとトークンのスコープの論理積で行う。
func fetchAPITopicAccess(ctx context.Context, topicRepo *repository.TopicRepository, topicMemberRepo *repository.TopicMemberRepository, principal *model.APIPrincipal) (*topicAccess, error) {
	return loadTopicAccess(ctx, topicRepo, topicMemberRepo, principal.Space.ID, principal.SpaceMember, func(topicMember *model.TopicMember) policy.Authorizer {
		return newAPIAuthorizer(principal, topicMember)
	})
}

// loadTopicAccessはスペースのトピックと閲覧者のトピックメンバーを取得し、authorizeで判定する
// topicAccessを返す。
func loadTopicAccess(
	ctx context.Context,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	spaceID model.SpaceID,
	spaceMember *model.SpaceMember,
	authorize func(topicMember *model.TopicMember) policy.Authorizer,
) (*topicAccess, error) {
	topics, err := topicRepo.ListActiveBySpace(ctx, spaceID)
	if err != nil {
		return nil, fmt.Errorf("トピック一覧の取得に失敗: %w", err)
	}

	topicByID := make(map[model.TopicID]*model.Topic, len(topics))
	topicIDs := make([]model.TopicID, len(topics))
	for i, topic := range topics {
		topicByID[topic.ID] = topic
		topicIDs[i] = topic.ID
	}

	memberByTopic := make(map[model.TopicID]*model.TopicMember, len(topics))
	if spaceMember != nil {
		topicMembers, err := topicMemberRepo.ListBySpaceMemberAndTopics(ctx, spaceID, spaceMember.ID, topicIDs)
		if err != nil {
			return nil, fmt.Errorf("トピックメンバーの一括取得に失敗: %w", err)
		}
		for _, topicMember := range topicMembers {
			memberByTopic[topicMember.TopicID] = topicMember
		}
	}

	return &topicAccess{
		authorize:     authorize,
		topics:        topics,
		topicByID:     topicByID,
		memberByTopic: memberByTopic,
	}, nil
}

// visibilityは閲覧者が開けるトピックに一覧を絞り込む条件を返す。
func (a *topicAccess) visibility() repository.TopicVisibility {
	visible := a.visibleTopics()
	visibleTopicIDs := make([]model.TopicID, len(visible))
	for i, topic := range visible {
		visibleTopicIDs[i] = topic.ID
	}
	return repository.VisibleTopics(visibleTopicIDs)
}

// topicMapForPagesは指定されたページ群が参照する解決済みのアクティブトピックを返す。
// 解決済みトピックを再利用することで、表示データと認可で同じスナップショットを使い、
// トピックの再クエリを避ける。
func (a *topicAccess) topicMapForPages(pageGroups ...[]*model.Page) map[model.TopicID]*model.Topic {
	topicMap := make(map[model.TopicID]*model.Topic)
	for _, pages := range pageGroups {
		for _, page := range pages {
			if topic, ok := a.topicByID[page.TopicID]; ok {
				topicMap[page.TopicID] = topic
			}
		}
	}

	return topicMap
}

// canShowPageは閲覧者が指定ページを開けるかを、そのページのトピックとゴミ箱の
// 閲覧権限から判定する。廃棄済みトピックのページは誰からも見えない (廃棄済みトピックは
// 解決済みのトピックに含まれないため)。
func (a *topicAccess) canShowPage(pg *model.Page) bool {
	topic, ok := a.topicByID[pg.TopicID]
	if !ok {
		return false
	}

	authorizer := a.authorizer(pg.TopicID)
	if !authorizer.CanShowTopic(topic) {
		return false
	}
	if pg.TrashedAt != nil && !authorizer.CanShowTrash() {
		return false
	}

	return true
}

// authorizerは指定トピックにおける閲覧者のAuthorizerを返す。スペースのスコープと、その
// トピックのトピックメンバーが持つスコープを統合する。CanUpdatePageのようにページの所属トピック
// に依存する権限の判定に使う。
func (a *topicAccess) authorizer(topicID model.TopicID) policy.Authorizer {
	return a.authorize(a.memberByTopic[topicID])
}

// visibleTopicsは閲覧者が開けるトピックを、スペースのトピックと同じnumber順で返す。
func (a *topicAccess) visibleTopics() []*model.Topic {
	visible := make([]*model.Topic, 0, len(a.topics))
	for _, topic := range a.topics {
		if a.authorizer(topic.ID).CanShowTopic(topic) {
			visible = append(visible, topic)
		}
	}
	return visible
}
