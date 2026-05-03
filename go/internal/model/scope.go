package model

// Scope はリソースに対する権限を表すドメイン型。
// GitHub 風の "resource:action" 形式で命名する。
type Scope string

// String は Scope を文字列に変換する
func (s Scope) String() string { return string(s) }

// トピック関連スコープ
const (
	ScopeTopicRead   Scope = "topic:read"
	ScopeTopicWrite  Scope = "topic:write"
	ScopeTopicDelete Scope = "topic:delete"
)

// トピックメンバー関連スコープ
const (
	ScopeTopicMemberRead   Scope = "topic_member:read"
	ScopeTopicMemberWrite  Scope = "topic_member:write"
	ScopeTopicMemberDelete Scope = "topic_member:delete"
)

// ページ関連スコープ
const (
	ScopePageRead    Scope = "page:read"
	ScopePageWrite   Scope = "page:write"
	ScopePageTrash   Scope = "page:trash"
	ScopePageRestore Scope = "page:restore"
)

// 下書きページ関連スコープ
const (
	ScopeDraftPageRead   Scope = "draft_page:read"
	ScopeDraftPageWrite  Scope = "draft_page:write"
	ScopeDraftPageDelete Scope = "draft_page:delete"
)

// 編集提案関連スコープ
const (
	ScopeSuggestionRead  Scope = "suggestion:read"
	ScopeSuggestionWrite Scope = "suggestion:write"
	ScopeSuggestionApply Scope = "suggestion:apply"
	ScopeSuggestionClose Scope = "suggestion:close"
)

// 編集提案コメント関連スコープ
const (
	ScopeSuggestionCommentRead  Scope = "suggestion_comment:read"
	ScopeSuggestionCommentWrite Scope = "suggestion_comment:write"
)

// スペースメンバー関連スコープ
const (
	ScopeSpaceMemberRead   Scope = "space_member:read"
	ScopeSpaceMemberWrite  Scope = "space_member:write"
	ScopeSpaceMemberDelete Scope = "space_member:delete"
)

// スペース関連スコープ
const (
	ScopeSpaceRead   Scope = "space:read"
	ScopeSpaceWrite  Scope = "space:write"
	ScopeSpaceDelete Scope = "space:delete"
	// ScopeSpaceAdmin は全スコープを包括する唯一の特別スコープ
	ScopeSpaceAdmin Scope = "space:admin"
)

// 添付ファイル関連スコープ
const (
	ScopeAttachmentRead   Scope = "attachment:read"
	ScopeAttachmentWrite  Scope = "attachment:write"
	ScopeAttachmentDelete Scope = "attachment:delete"
)

// HasScope は指定のスコープがスライスに含まれているかチェックする
func HasScope(scopes []Scope, target Scope) bool {
	for _, s := range scopes {
		if s == target {
			return true
		}
	}
	return false
}

// StringsToScopes は []string を []Scope に変換する
func StringsToScopes(ss []string) []Scope {
	scopes := make([]Scope, len(ss))
	for i, s := range ss {
		scopes[i] = Scope(s)
	}
	return scopes
}
