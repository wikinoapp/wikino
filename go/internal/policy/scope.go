// Package policyはリソースに対する権限チェックを提供する
package policy

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// implicationsはリソース内の含意ルール (上位スコープ → 下位スコープ)
var implications = buildImplications()

func buildImplications() map[model.Scope][]model.Scope {
	m := make(map[model.Scope][]model.Scope)
	for _, d := range model.ScopeDefinitions {
		if len(d.Implies) > 0 {
			m[d.Scope] = d.Implies
		}
	}
	return m
}

// expandScopesはスコープの含意を展開し、有効なスコープの集合を返す。
// DB保存時には展開しない。判定時にのみ使用する。
func expandScopes(scopes []model.Scope) []model.Scope {
	expanded := make([]model.Scope, 0, len(scopes)*2)
	expanded = append(expanded, scopes...)

	// リソース内の含意展開 (write → read、個人用アクセストークン・OAuthの連携・OAuthアプリはdelete → readも含む)
	for _, s := range scopes {
		if implied, ok := implications[s]; ok {
			expanded = append(expanded, implied...)
		}
	}

	return deduplicate(expanded)
}

// deduplicateはスコープのスライスから重複を除去する
func deduplicate(scopes []model.Scope) []model.Scope {
	seen := make(map[model.Scope]bool, len(scopes))
	result := make([]model.Scope, 0, len(scopes))
	for _, s := range scopes {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// ExpandAPITokenScopesは公開APIのトークンに付与できるスコープだけを残してから含意展開する。
// 発行時の検証をすり抜けた `space:admin` などが保存されていても、含意から権限が
// 生まれないようにする
func ExpandAPITokenScopes(scopes []model.Scope) []model.Scope {
	allowed := make([]model.Scope, 0, len(scopes))
	for _, s := range scopes {
		if model.HasScope(model.APITokenScopes, s) {
			allowed = append(allowed, s)
		}
	}
	expanded := expandScopes(allowed)
	result := make([]model.Scope, 0, len(expanded))
	for _, s := range expanded {
		if model.HasScope(model.APITokenScopes, s) {
			result = append(result, s)
		}
	}
	return result
}

// TopicRoleScopesは、トピックメンバーのロールのスコープに、スペースメンバーの上限を掛けて返す。
// 上限があれば、含意を展開したトピックのロールのスコープのうち、展開した上限のスコープにあるものだけを残す。
// スペースのロールのスコープは上限で削らないため、ここでは扱わない。トピックメンバーでなければnil
func TopicRoleScopes(spaceMember *model.SpaceMember, topicMember *model.TopicMember) []model.Scope {
	if topicMember == nil {
		return nil
	}
	scopes := topicMember.Role.Scopes()
	if spaceMember == nil || spaceMember.MaxTopicRole == nil {
		return scopes
	}

	maxScopes := expandScopes(spaceMember.MaxTopicRole.Scopes())
	expanded := expandScopes(scopes)
	result := make([]model.Scope, 0, len(expanded))
	for _, s := range expanded {
		if model.HasScope(maxScopes, s) {
			result = append(result, s)
		}
	}
	return result
}

// HeldAPITokenScopesは、トークンに付与できるスコープ (model.APITokenScopes) のうち、メンバーが
// スペースのロールかいずれかのトピックのロールで持つもの (含意の展開後) をAPITokenScopesの順に返す。
// トークンで効く範囲はメンバーの有効スコープとの積で決まる (NewAPIMemberPolicy) ため、ここに無い
// スコープはトークンに付与しても今の権限では効かない。topicMembersはspaceMemberのトピックメンバー
func HeldAPITokenScopes(spaceMember *model.SpaceMember, topicMembers []*model.TopicMember) []model.Scope {
	merged := spaceMember.Role.Scopes()
	for _, tm := range topicMembers {
		merged = append(merged, TopicRoleScopes(spaceMember, tm)...)
	}
	expanded := expandScopes(merged)

	held := make([]model.Scope, 0, len(model.APITokenScopes))
	for _, s := range model.APITokenScopes {
		if model.HasScope(expanded, s) {
			held = append(held, s)
		}
	}
	return held
}
