package usecase

import (
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
)

// newAuthorizer はスペースメンバーとトピックメンバーから適切な Authorizer を生成する。
// spaceMember が nil の場合は GuestPolicy を返し、そうでなければ MemberPolicy を返す。
func newAuthorizer(spaceMember *model.SpaceMember, topicMember *model.TopicMember) policy.Authorizer {
	if spaceMember == nil {
		return policy.NewGuestPolicy()
	}
	var topicScopes []model.Scope
	if topicMember != nil {
		topicScopes = topicMember.Scopes
	}
	return policy.NewMemberPolicy(spaceMember.Scopes, topicScopes)
}
