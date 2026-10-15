package usecase

import (
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
)

// newAuthorizerはスペースメンバーとトピックメンバーから適切なAuthorizerを生成する。
// spaceMemberがnilの場合はGuestPolicyを返し、そうでなければMemberPolicyを返す。
// スコープはスペースのロールと、上限を掛けたトピックのロールから引く。
func newAuthorizer(spaceMember *model.SpaceMember, topicMember *model.TopicMember) policy.Authorizer {
	if spaceMember == nil {
		return policy.NewGuestPolicy()
	}
	return policy.NewMemberPolicy(spaceMember.Role.Scopes(), policy.TopicRoleScopes(spaceMember, topicMember))
}

// newAPIAuthorizerは公開APIのトークンの主体とトピックメンバーからAuthorizerを生成する。
// メンバーのロールから引いたスコープ (スペース + 上限を掛けたトピック) とトークンのスコープの論理積で判定する。
func newAPIAuthorizer(principal *model.APIPrincipal, topicMember *model.TopicMember) policy.Authorizer {
	var spaceScopes []model.Scope
	if principal.SpaceMember != nil {
		spaceScopes = principal.SpaceMember.Role.Scopes()
	}
	return policy.NewAPIMemberPolicy(spaceScopes, policy.TopicRoleScopes(principal.SpaceMember, topicMember), principal.Scopes)
}
