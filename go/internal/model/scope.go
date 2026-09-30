package model

// Scopeはリソースに対する権限を表すドメイン型。
// GitHub風の "resource:action" 形式で命名する。
type Scope string

// StringはScopeを文字列に変換する
func (s Scope) String() string { return string(s) }

// トピック関連スコープ
const (
	ScopeTopicRead   Scope = "topic:read"
	ScopeTopicWrite  Scope = "topic:write"
	ScopeTopicDelete Scope = "topic:delete"
)

// トピックの公開範囲関連スコープ
const (
	ScopeTopicVisibilityWrite Scope = "topic_visibility:write"
)

// トピックメンバー関連スコープ
const (
	ScopeTopicMemberRead   Scope = "topic_member:read"
	ScopeTopicMemberWrite  Scope = "topic_member:write"
	ScopeTopicMemberDelete Scope = "topic_member:delete"
)

// ページ関連スコープ
const (
	ScopePageRead  Scope = "page:read"
	ScopePageWrite Scope = "page:write"
)

// ゴミ箱関連スコープ
const (
	ScopePageTrashRead   Scope = "page_trash:read"
	ScopePageTrashWrite  Scope = "page_trash:write"
	ScopePageTrashDelete Scope = "page_trash:delete"
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
)

// 編集提案の反映・クローズ関連スコープ
const (
	ScopeSuggestionApplicationWrite Scope = "suggestion_application:write"
	ScopeSuggestionClosureWrite     Scope = "suggestion_closure:write"
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
	// ScopeSpaceAdminは全スコープを包括する唯一の特別スコープ
	ScopeSpaceAdmin Scope = "space:admin"
)

// 添付ファイル関連スコープ
const (
	ScopeAttachmentRead   Scope = "attachment:read"
	ScopeAttachmentWrite  Scope = "attachment:write"
	ScopeAttachmentDelete Scope = "attachment:delete"
)

// 個人アクセストークン関連スコープ
const (
	ScopePersonalAccessTokenRead   Scope = "personal_access_token:read"
	ScopePersonalAccessTokenWrite  Scope = "personal_access_token:write"
	ScopePersonalAccessTokenDelete Scope = "personal_access_token:delete"
)

// OAuthの連携関連スコープ
const (
	ScopeOAuthGrantRead   Scope = "oauth_grant:read"
	ScopeOAuthGrantWrite  Scope = "oauth_grant:write"
	ScopeOAuthGrantDelete Scope = "oauth_grant:delete"
)

// OAuthアプリ関連スコープ
const (
	ScopeOAuthApplicationRead   Scope = "oauth_application:read"
	ScopeOAuthApplicationWrite  Scope = "oauth_application:write"
	ScopeOAuthApplicationDelete Scope = "oauth_application:delete"
)

// ScopeLevelは、スコープを付けられる最も下の単位を表す。それより上の単位にも付けられる
type ScopeLevel int

const (
	// ScopeLevelSpaceは、スペースメンバーにだけ付けられる
	ScopeLevelSpace ScopeLevel = iota + 1
	// ScopeLevelTopicは、トピックメンバーにも付けられる。スペースメンバーに付けると、すべてのトピックで有効になる
	ScopeLevelTopic
)

// ScopeDefinitionは、1つのスコープの性質を表す
type ScopeDefinition struct {
	Scope Scope
	// LowestLevelは、このスコープを付けられる最も下の単位
	LowestLevel ScopeLevel
	// Impliesは、このスコープが含意するスコープ (同じリソースの中に限る)
	Implies []Scope
	// APITokenは、公開APIのトークンに付与できるか
	APIToken bool
}

// ScopeDefinitionsは、メンバーに付けられるすべてのスコープの定義。並び順は画面でスコープを並べる順とする。
// space:adminはこの表のすべてのスコープに展開される特別なスコープのため、表には載せない
var ScopeDefinitions = []ScopeDefinition{
	// スペース
	{Scope: ScopeSpaceRead, LowestLevel: ScopeLevelSpace},
	{Scope: ScopeSpaceWrite, LowestLevel: ScopeLevelSpace, Implies: []Scope{ScopeSpaceRead}},
	{Scope: ScopeSpaceDelete, LowestLevel: ScopeLevelSpace},
	// トピック
	{Scope: ScopeTopicRead, LowestLevel: ScopeLevelTopic, APIToken: true},
	{Scope: ScopeTopicWrite, LowestLevel: ScopeLevelTopic, Implies: []Scope{ScopeTopicRead}},
	{Scope: ScopeTopicDelete, LowestLevel: ScopeLevelTopic},
	// トピックの公開範囲 (読み取りはtopic:readで足りるため、writeだけを持つ)
	{Scope: ScopeTopicVisibilityWrite, LowestLevel: ScopeLevelTopic},
	// トピックメンバー
	{Scope: ScopeTopicMemberRead, LowestLevel: ScopeLevelTopic},
	{Scope: ScopeTopicMemberWrite, LowestLevel: ScopeLevelTopic, Implies: []Scope{ScopeTopicMemberRead}},
	{Scope: ScopeTopicMemberDelete, LowestLevel: ScopeLevelTopic},
	// ページ
	{Scope: ScopePageRead, LowestLevel: ScopeLevelTopic, APIToken: true},
	{Scope: ScopePageWrite, LowestLevel: ScopeLevelTopic, Implies: []Scope{ScopePageRead}, APIToken: true},
	// ゴミ箱
	{Scope: ScopePageTrashRead, LowestLevel: ScopeLevelTopic},
	{Scope: ScopePageTrashWrite, LowestLevel: ScopeLevelTopic, Implies: []Scope{ScopePageTrashRead}},
	{Scope: ScopePageTrashDelete, LowestLevel: ScopeLevelTopic},
	// 下書きページ
	{Scope: ScopeDraftPageRead, LowestLevel: ScopeLevelTopic},
	{Scope: ScopeDraftPageWrite, LowestLevel: ScopeLevelTopic, Implies: []Scope{ScopeDraftPageRead}},
	{Scope: ScopeDraftPageDelete, LowestLevel: ScopeLevelTopic},
	// 編集提案
	{Scope: ScopeSuggestionRead, LowestLevel: ScopeLevelTopic},
	{Scope: ScopeSuggestionWrite, LowestLevel: ScopeLevelTopic, Implies: []Scope{ScopeSuggestionRead}},
	{Scope: ScopeSuggestionApplicationWrite, LowestLevel: ScopeLevelTopic},
	{Scope: ScopeSuggestionClosureWrite, LowestLevel: ScopeLevelTopic},
	// 編集提案コメント
	{Scope: ScopeSuggestionCommentRead, LowestLevel: ScopeLevelTopic},
	{Scope: ScopeSuggestionCommentWrite, LowestLevel: ScopeLevelTopic, Implies: []Scope{ScopeSuggestionCommentRead}},
	// スペースメンバー
	{Scope: ScopeSpaceMemberRead, LowestLevel: ScopeLevelSpace},
	{Scope: ScopeSpaceMemberWrite, LowestLevel: ScopeLevelSpace, Implies: []Scope{ScopeSpaceMemberRead}},
	{Scope: ScopeSpaceMemberDelete, LowestLevel: ScopeLevelSpace},
	// 添付ファイル (スペースに属し、判定でトピックを見ないため、スペースにだけ付けられる)
	{Scope: ScopeAttachmentRead, LowestLevel: ScopeLevelSpace},
	{Scope: ScopeAttachmentWrite, LowestLevel: ScopeLevelSpace, Implies: []Scope{ScopeAttachmentRead}},
	{Scope: ScopeAttachmentDelete, LowestLevel: ScopeLevelSpace},
	// 個人アクセストークン
	{Scope: ScopePersonalAccessTokenRead, LowestLevel: ScopeLevelSpace},
	{Scope: ScopePersonalAccessTokenWrite, LowestLevel: ScopeLevelSpace, Implies: []Scope{ScopePersonalAccessTokenRead}},
	{Scope: ScopePersonalAccessTokenDelete, LowestLevel: ScopeLevelSpace, Implies: []Scope{ScopePersonalAccessTokenRead}},
	// OAuthの連携
	{Scope: ScopeOAuthGrantRead, LowestLevel: ScopeLevelSpace},
	{Scope: ScopeOAuthGrantWrite, LowestLevel: ScopeLevelSpace, Implies: []Scope{ScopeOAuthGrantRead}},
	{Scope: ScopeOAuthGrantDelete, LowestLevel: ScopeLevelSpace, Implies: []Scope{ScopeOAuthGrantRead}},
	// OAuthアプリ
	{Scope: ScopeOAuthApplicationRead, LowestLevel: ScopeLevelSpace},
	{Scope: ScopeOAuthApplicationWrite, LowestLevel: ScopeLevelSpace, Implies: []Scope{ScopeOAuthApplicationRead}},
	{Scope: ScopeOAuthApplicationDelete, LowestLevel: ScopeLevelSpace, Implies: []Scope{ScopeOAuthApplicationRead}},
}

// HasScopeは指定のスコープがスライスに含まれているかチェックする
func HasScope(scopes []Scope, target Scope) bool {
	for _, s := range scopes {
		if s == target {
			return true
		}
	}
	return false
}

// StringsToScopesは []stringを []Scopeに変換する
func StringsToScopes(ss []string) []Scope {
	scopes := make([]Scope, len(ss))
	for i, s := range ss {
		scopes[i] = Scope(s)
	}
	return scopes
}

// ScopesToStringsは []Scopeを []stringに変換する
func ScopesToStrings(scopes []Scope) []string {
	ss := make([]string, len(scopes))
	for i, s := range scopes {
		ss[i] = string(s)
	}
	return ss
}

// APITokenScopesは、公開APIのトークン (個人アクセストークン・OAuthのアクセストークン) に
// 付与できるスコープ。管理系のスコープとトークン管理のスコープは含めない。トークンでトークンを
// 発行できると、権限の範囲を利用者の意図より広げられるためである。ScopeDefinitionsの順に並ぶ
var APITokenScopes = apiTokenScopes()

func apiTokenScopes() []Scope {
	var scopes []Scope
	for _, d := range ScopeDefinitions {
		if d.APIToken {
			scopes = append(scopes, d.Scope)
		}
	}
	return scopes
}

// SortAPITokenScopesは、トークンや許可のスコープをAPITokenScopesの順に並べ直した新しいスライスを
// 返す。許可のスコープは保存順が揃っていない (新規作成では要求の順、既存の許可の拡張では文字列の順)
// ため、表示の前に並べ直す。APITokenScopesに無いスコープが保存されていても隠さないよう、それらは
// 元の順のまま末尾に置く
func SortAPITokenScopes(scopes []Scope) []Scope {
	sorted := make([]Scope, 0, len(scopes))
	for _, s := range APITokenScopes {
		if HasScope(scopes, s) {
			sorted = append(sorted, s)
		}
	}
	for _, s := range scopes {
		if !HasScope(APITokenScopes, s) && !HasScope(sorted, s) {
			sorted = append(sorted, s)
		}
	}
	return sorted
}
