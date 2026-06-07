package usecase

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// buildCanCreatePageByTopic resolves, per topic, whether the user may create a page in it. It indexes
// the user's topic memberships once, then evaluates each topic by merging the space-level and
// topic-level scopes through newAuthorizer. The applicable space membership is obtained per topic via
// spaceMemberForTopic, so callers that resolve a single space (the space page) and callers that span
// multiple spaces (the home page) share this authorization loop while differing only in how they fetch
// the inputs.
//
// [Ja] buildCanCreatePageByTopic はトピックごとに、ユーザーがそこにページを作成できるかを解決する。
// ユーザーのトピックメンバーを 1 度だけ index 化し、各トピックを newAuthorizer でスペーススコープと
// トピックスコープを統合して判定する。判定に使うスペースメンバーは spaceMemberForTopic でトピック
// ごとに取得するため、単一スペースを解決する呼び出し元 (スペース画面) と複数スペースに跨る呼び出し元
// (ホーム画面) が、入力の取得方法だけを変えてこの認可ループを共有できる。
func buildCanCreatePageByTopic(
	topics []*model.Topic,
	topicMembers []*model.TopicMember,
	spaceMemberForTopic func(topic *model.Topic) *model.SpaceMember,
) map[model.TopicID]bool {
	topicMemberByTopic := make(map[model.TopicID]*model.TopicMember, len(topicMembers))
	for _, topicMember := range topicMembers {
		topicMemberByTopic[topicMember.TopicID] = topicMember
	}

	canCreatePageByTopic := make(map[model.TopicID]bool, len(topics))
	for _, topic := range topics {
		spaceMember := spaceMemberForTopic(topic)
		canCreatePageByTopic[topic.ID] = newAuthorizer(spaceMember, topicMemberByTopic[topic.ID]).CanCreatePage()
	}
	return canCreatePageByTopic
}
