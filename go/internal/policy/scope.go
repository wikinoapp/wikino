// Package policyはリソースに対する権限チェックを提供する
package policy

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// implicationsはリソース内の含意ルール (上位スコープ → 下位スコープ)
var implications = map[model.Scope][]model.Scope{
	model.ScopeTopicWrite:             {model.ScopeTopicRead},
	model.ScopeTopicMemberWrite:       {model.ScopeTopicMemberRead},
	model.ScopePageWrite:              {model.ScopePageRead},
	model.ScopeDraftPageWrite:         {model.ScopeDraftPageRead},
	model.ScopeSuggestionWrite:        {model.ScopeSuggestionRead},
	model.ScopeSuggestionCommentWrite: {model.ScopeSuggestionCommentRead},
	model.ScopeSpaceWrite:             {model.ScopeSpaceRead},
	model.ScopeSpaceMemberWrite:       {model.ScopeSpaceMemberRead},
	model.ScopeAttachmentWrite:        {model.ScopeAttachmentRead},
}

// allResourceScopesはspace:adminが包括するすべてのリソーススコープを返す。
// space:admin自体は含まない。
func allResourceScopes() []model.Scope {
	return []model.Scope{
		// スペース
		model.ScopeSpaceRead,
		model.ScopeSpaceWrite,
		model.ScopeSpaceDelete,
		// トピック
		model.ScopeTopicRead,
		model.ScopeTopicWrite,
		model.ScopeTopicDelete,
		// トピックメンバー
		model.ScopeTopicMemberRead,
		model.ScopeTopicMemberWrite,
		model.ScopeTopicMemberDelete,
		// ページ
		model.ScopePageRead,
		model.ScopePageWrite,
		model.ScopePageTrash,
		model.ScopePageRestore,
		// 下書きページ
		model.ScopeDraftPageRead,
		model.ScopeDraftPageWrite,
		model.ScopeDraftPageDelete,
		// 編集提案
		model.ScopeSuggestionRead,
		model.ScopeSuggestionWrite,
		model.ScopeSuggestionApply,
		model.ScopeSuggestionClose,
		// 編集提案コメント
		model.ScopeSuggestionCommentRead,
		model.ScopeSuggestionCommentWrite,
		// スペースメンバー
		model.ScopeSpaceMemberRead,
		model.ScopeSpaceMemberWrite,
		model.ScopeSpaceMemberDelete,
		// 添付ファイル
		model.ScopeAttachmentRead,
		model.ScopeAttachmentWrite,
		model.ScopeAttachmentDelete,
	}
}

// expandScopesはスコープの含意を展開し、有効なスコープの集合を返す。
// DB保存時には展開しない。判定時にのみ使用する。
func expandScopes(scopes []model.Scope) []model.Scope {
	expanded := make([]model.Scope, 0, len(scopes)*2)
	expanded = append(expanded, scopes...)

	// リソース内の含意展開 (write → read)
	for _, s := range scopes {
		if implied, ok := implications[s]; ok {
			expanded = append(expanded, implied...)
		}
	}

	// space:adminは全リソーススコープを包括する (唯一の特別スコープ)
	if hasScope(scopes, model.ScopeSpaceAdmin) {
		expanded = append(expanded, allResourceScopes()...)
	}

	return deduplicate(expanded)
}

// hasScopeは指定のスコープがスライスに含まれているかチェックする
func hasScope(scopes []model.Scope, target model.Scope) bool {
	for _, s := range scopes {
		if s == target {
			return true
		}
	}
	return false
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
