package model

import (
	"strings"
	"time"
)

// APITokenKindは公開APIのトークンの種類
type APITokenKind string

const (
	// APITokenKindPersonalAccessTokenは個人用アクセストークン
	APITokenKindPersonalAccessToken APITokenKind = "personal_access_token"
	// APITokenKindOAuthAccessTokenはOAuthのアクセストークン
	APITokenKindOAuthAccessToken APITokenKind = "oauth_access_token"
)

// APIPrincipalはトークンで認証した公開APIの呼び出し主体。トークンの持ち主と、トークンが
// 属するスペースを表す
type APIPrincipal struct {
	User        *User
	Space       *Space
	SpaceMember *SpaceMember
	TokenKind   APITokenKind
	// TokenExpiresAtはトークンの有効期限
	TokenExpiresAt time.Time
	// Scopesはトークンのスコープを含意展開したもの。トークンに付与できるスコープ
	// (APITokenScopes) の範囲に限る。メンバー権限との論理積は含まない
	Scopes []Scope
}

// HasScopeは主体のトークンが指定のスコープを持つかを返す
func (p *APIPrincipal) HasScope(scope Scope) bool {
	return HasScope(p.Scopes, scope)
}

// IsBoundToは主体のトークンがidentifierのスペースに属しているかを返す。
// 公開APIのパスで指定されたスペースにトークンが属さないかの判定を、HandlerとUseCaseで揃えるために使う。
// スペース識別子の列はcitextで、Web画面のスペースの検索は大文字と小文字を区別しないため、ここでも区別しない
func (p *APIPrincipal) IsBoundTo(identifier SpaceIdentifier) bool {
	return p != nil && p.Space != nil && strings.EqualFold(string(p.Space.Identifier), string(identifier))
}
