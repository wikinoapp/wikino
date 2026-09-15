package usecase

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// buildCanCreatePageByTopicはトピックごとに、ユーザーがそこにページを作成できるかを解決する。
// ユーザーのトピックメンバーを1度だけindex化し、各トピックをnewAuthorizerでスペーススコープと
// トピックスコープを統合して判定する。判定に使うスペースメンバーはspaceMemberForTopicでトピック
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
