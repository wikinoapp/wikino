package usecase

import (
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
)

// newAuthorizerはスペースメンバーとトピックメンバーから適切なAuthorizerを生成する。
// spaceMemberがnilの場合はGuestPolicyを返し、そうでなければMemberPolicyを返す。
// スコープはスペースのロールとトピックのロールから引く。
func newAuthorizer(spaceMember *model.SpaceMember, topicMember *model.TopicMember) policy.Authorizer {
	if spaceMember == nil {
		return policy.NewGuestPolicy()
	}
	return policy.NewMemberPolicy(spaceMember.Role.Scopes(), topicRoleScopes(topicMember))
}

// newAPIAuthorizerは公開APIのトークンの主体とトピックメンバーからAuthorizerを生成する。
// メンバーのロールから引いたスコープ (スペース + トピック) とトークンのスコープの論理積で判定する。
func newAPIAuthorizer(principal *model.APIPrincipal, topicMember *model.TopicMember) policy.Authorizer {
	var spaceScopes []model.Scope
	if principal.SpaceMember != nil {
		spaceScopes = principal.SpaceMember.Role.Scopes()
	}
	return policy.NewAPIMemberPolicy(spaceScopes, topicRoleScopes(topicMember), principal.Scopes)
}

// topicRoleScopesは、トピックメンバーのロールが持つスコープを返す。トピックメンバーでなければnil
func topicRoleScopes(topicMember *model.TopicMember) []model.Scope {
	if topicMember == nil {
		return nil
	}
	return topicMember.Role.Scopes()
}
