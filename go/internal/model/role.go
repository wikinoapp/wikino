package model

import "slices"

// SpaceRoleは、スペースメンバーに付けるロール (名前を付けたスコープの集合)。
// メンバーにはロールのキーだけを保存し、判定のたびにロールからスコープを引く
type SpaceRole string

const (
	// SpaceRoleAdminは、定義の表のすべてのスコープを持つ管理者
	SpaceRoleAdmin SpaceRole = "admin"
	// SpaceRoleEditorは、トピックを作り、ページを書ける編集者
	SpaceRoleEditor SpaceRole = "editor"
	// SpaceRoleViewerは、ページを読み、編集提案とそのコメントを書ける閲覧者
	SpaceRoleViewer SpaceRole = "viewer"
)

// SpaceRolesは、スペース用のすべてのロール。権限の広い順に並ぶ
var SpaceRoles = []SpaceRole{SpaceRoleAdmin, SpaceRoleEditor, SpaceRoleViewer}

// TopicRoleは、トピックメンバーに付けるロール。トピックまで付けられるスコープだけで作る
type TopicRole string

const (
	// TopicRoleAdminは、トピックの設定・公開範囲・メンバーを管理できるトピック管理者
	TopicRoleAdmin TopicRole = "admin"
	// TopicRoleEditorは、トピックのページを書けるトピック編集者
	TopicRoleEditor TopicRole = "editor"
	// TopicRoleViewerは、トピックのページを読み、編集提案とそのコメントを書けるトピック閲覧者
	TopicRoleViewer TopicRole = "viewer"
)

// TopicRolesは、トピック用のすべてのロール。権限の広い順に並ぶ
var TopicRoles = []TopicRole{TopicRoleAdmin, TopicRoleEditor, TopicRoleViewer}

// ロールのスコープは含意を展開する前の値で持つ (writeは同じリソースのreadを含意する)
var (
	spaceEditorScopes = []Scope{
		// スペースにしか付けられないスコープ
		ScopeSpaceMemberRead,
		ScopeAttachmentWrite,
		ScopePersonalAccessTokenWrite,
		ScopePersonalAccessTokenDelete,
		ScopeOAuthGrantWrite,
		ScopeOAuthGrantDelete,
		// トピックまで付けられるスコープ。topic:writeはトピックを作るために持つ
		ScopeTopicWrite,
		ScopeTopicMemberRead,
		ScopePageWrite,
		ScopePageTrashWrite,
		ScopePageTrashDelete,
		ScopeDraftPageWrite,
		ScopeDraftPageDelete,
		ScopeSuggestionWrite,
		ScopeSuggestionApplicationWrite,
		ScopeSuggestionClosureWrite,
		ScopeSuggestionCommentWrite,
	}

	spaceViewerScopes = []Scope{
		// スペースにしか付けられないスコープ
		ScopeSpaceMemberRead,
		ScopeAttachmentRead,
		ScopePersonalAccessTokenWrite,
		ScopePersonalAccessTokenDelete,
		ScopeOAuthGrantWrite,
		ScopeOAuthGrantDelete,
		// トピックまで付けられるスコープ
		ScopeTopicRead,
		ScopeTopicMemberRead,
		ScopePageRead,
		ScopeSuggestionWrite,
		ScopeSuggestionCommentWrite,
	}

	topicAdminScopes = []Scope{
		ScopeTopicWrite,
		ScopeTopicDelete,
		ScopeTopicVisibilityWrite,
		ScopeTopicMemberWrite,
		ScopeTopicMemberDelete,
		ScopePageWrite,
		ScopePageTrashWrite,
		ScopePageTrashDelete,
		ScopeDraftPageWrite,
		ScopeDraftPageDelete,
		ScopeSuggestionWrite,
		ScopeSuggestionApplicationWrite,
		ScopeSuggestionClosureWrite,
		ScopeSuggestionCommentWrite,
	}

	topicEditorScopes = []Scope{
		ScopeTopicRead,
		ScopeTopicMemberRead,
		ScopePageWrite,
		ScopePageTrashWrite,
		ScopePageTrashDelete,
		ScopeDraftPageWrite,
		ScopeDraftPageDelete,
		ScopeSuggestionWrite,
		ScopeSuggestionApplicationWrite,
		ScopeSuggestionClosureWrite,
		ScopeSuggestionCommentWrite,
	}

	topicViewerScopes = []Scope{
		ScopeTopicRead,
		ScopeTopicMemberRead,
		ScopePageRead,
		ScopeSuggestionWrite,
		ScopeSuggestionCommentWrite,
	}
)

// Scopesは、ロールが持つスコープ (含意の展開前) を返す。管理者は定義の表のすべてのスコープを持つため、
// スコープを表に足すと管理者にも行き渡る。未知のロールはスコープを持たない
func (r SpaceRole) Scopes() []Scope {
	switch r {
	case SpaceRoleAdmin:
		scopes := make([]Scope, len(ScopeDefinitions))
		for i, d := range ScopeDefinitions {
			scopes[i] = d.Scope
		}
		return scopes
	case SpaceRoleEditor:
		return slices.Clone(spaceEditorScopes)
	case SpaceRoleViewer:
		return slices.Clone(spaceViewerScopes)
	default:
		return nil
	}
}

// LowestLevelは、ロールを付けられる最も下の単位を、含むスコープの単位から求める
func (r SpaceRole) LowestLevel() ScopeLevel {
	return lowestLevelOf(r.Scopes())
}

// RailsScopesは、Rails版が判定に使うspace_members.scopesへ書く値を返す。
// Rails版はspace:adminを全スコープに展開し、知らないスコープを無視するため、
// 管理者はspace:adminだけを、ほかのロールはロールのスコープをそのまま書く
func (r SpaceRole) RailsScopes() []Scope {
	if r == SpaceRoleAdmin {
		return []Scope{ScopeSpaceAdmin}
	}
	return r.Scopes()
}

// Scopesは、ロールが持つスコープ (含意の展開前) を返す。未知のロールはスコープを持たない
func (r TopicRole) Scopes() []Scope {
	switch r {
	case TopicRoleAdmin:
		return slices.Clone(topicAdminScopes)
	case TopicRoleEditor:
		return slices.Clone(topicEditorScopes)
	case TopicRoleViewer:
		return slices.Clone(topicViewerScopes)
	default:
		return nil
	}
}

// LowestLevelは、ロールを付けられる最も下の単位を、含むスコープの単位から求める
func (r TopicRole) LowestLevel() ScopeLevel {
	return lowestLevelOf(r.Scopes())
}

// lowestLevelOfは、スコープの集合を付けられる最も下の単位を返す。
// 1つでもスペースにしか付けられないスコープを含めば、集合もスペースにしか付けられない
func lowestLevelOf(scopes []Scope) ScopeLevel {
	level := ScopeLevelTopic
	for _, d := range ScopeDefinitions {
		if HasScope(scopes, d.Scope) && d.LowestLevel < level {
			level = d.LowestLevel
		}
	}
	return level
}
