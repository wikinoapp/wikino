package model

// APITokenKindは公開APIのトークンの種類
type APITokenKind string

const (
	// APITokenKindPersonalAccessTokenは個人アクセストークン
	APITokenKindPersonalAccessToken APITokenKind = "personal_access_token"
	// APITokenKindOAuthAccessTokenはOAuthのアクセストークン
	APITokenKindOAuthAccessToken APITokenKind = "oauth_access_token"
)

// APIPrincipalはトークンで認証した公開APIの呼び出し主体。トークンの持ち主と、トークンを
// 束縛したスペースを表す
type APIPrincipal struct {
	User        *User
	Space       *Space
	SpaceMember *SpaceMember
	TokenKind   APITokenKind
	// Scopesはトークンのスコープを含意展開したもの。トークンに付与できるスコープ
	// (APITokenScopes) の範囲に限る。メンバー権限との論理積は含まない
	Scopes []Scope
}

// HasScopeは主体のトークンが指定のスコープを持つかを返す
func (p *APIPrincipal) HasScope(scope Scope) bool {
	return HasScope(p.Scopes, scope)
}

// IsBoundToは主体のトークンがidentifierのスペースに束縛されているかを返す。
// 公開APIのパスで指定されたスペースが束縛先と異なるかの判定を、HandlerとUseCaseで揃えるために使う
func (p *APIPrincipal) IsBoundTo(identifier SpaceIdentifier) bool {
	return p != nil && p.Space != nil && p.Space.Identifier == identifier
}
