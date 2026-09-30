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

// allResourceScopesはspace:adminが包括するすべてのリソーススコープ (定義の表のすべてのスコープ) を返す。
// space:admin自体は含まない。
func allResourceScopes() []model.Scope {
	scopes := make([]model.Scope, len(model.ScopeDefinitions))
	for i, d := range model.ScopeDefinitions {
		scopes[i] = d.Scope
	}
	return scopes
}

// expandScopesはスコープの含意を展開し、有効なスコープの集合を返す。
// DB保存時には展開しない。判定時にのみ使用する。
func expandScopes(scopes []model.Scope) []model.Scope {
	expanded := make([]model.Scope, 0, len(scopes)*2)
	expanded = append(expanded, scopes...)

	// リソース内の含意展開 (write → read、個人アクセストークン・OAuthの連携・OAuthアプリはdelete → readも含む)
	for _, s := range scopes {
		if implied, ok := implications[s]; ok {
			expanded = append(expanded, implied...)
		}
	}

	// space:adminは全リソーススコープを包括する (唯一の特別スコープ)
	if model.HasScope(scopes, model.ScopeSpaceAdmin) {
		expanded = append(expanded, allResourceScopes()...)
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
