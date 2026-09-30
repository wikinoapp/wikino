package usecase

import (
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
)

// newAuthorizerはスペースメンバーとトピックメンバーから適切なAuthorizerを生成する。
// spaceMemberがnilの場合はGuestPolicyを返し、そうでなければMemberPolicyを返す。
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

// newAPIAuthorizerは公開APIのトークンの主体とトピックメンバーからAuthorizerを生成する。
// メンバーのスコープ (スペース + トピック) とトークンのスコープの論理積で判定する。
func newAPIAuthorizer(principal *model.APIPrincipal, topicMember *model.TopicMember) policy.Authorizer {
	var spaceScopes []model.Scope
	if principal.SpaceMember != nil {
		spaceScopes = principal.SpaceMember.Scopes
	}
	var topicScopes []model.Scope
	if topicMember != nil {
		topicScopes = topicMember.Scopes
	}
	return policy.NewAPIMemberPolicy(spaceScopes, topicScopes, principal.Scopes)
}
