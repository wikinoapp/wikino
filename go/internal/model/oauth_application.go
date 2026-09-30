package model

import (
	"time"
)

// OAuthClientTypeはOAuthアプリのクライアントの種別 (RFC 6749 §2.1)
type OAuthClientType int32

const (
	// OAuthClientTypeConfidentialはシークレットを安全に保管できるクライアント (サーバーで動くWebアプリなど)
	OAuthClientTypeConfidential OAuthClientType = 0
	// OAuthClientTypePublicはシークレットを保管できないクライアント (CLI・デスクトップなど)
	OAuthClientTypePublic OAuthClientType = 1
)

// OAuthClientTypesは登録フォームで選べるクライアントの種別で、フォームに並べる順である
var OAuthClientTypes = []OAuthClientType{OAuthClientTypeConfidential, OAuthClientTypePublic}

// Stringはクライアントの種別をRFC 6749 §2.1の名前で返す。登録フォームが送る値にも使う
func (t OAuthClientType) String() string {
	switch t {
	case OAuthClientTypeConfidential:
		return "confidential"
	case OAuthClientTypePublic:
		return "public"
	default:
		return ""
	}
}

// OAuthApplicationはOAuthアプリのドメインモデル。アプリはスペースが持ち、登録した
// スペースでだけ使える。SpaceIDがnilのアプリは、全スペース共通のWikinoの公式クライアント
// (CLI・デスクトップ) である。CreatedSpaceMemberIDは作成したメンバーで、そのメンバーが
// いなくなるとnilになる。
//
// クライアントシークレットの値そのものは保持せず、ClientSecretDigestで照合する。
// publicクライアントはシークレットを持たず、ClientSecretDigestはnilである。
type OAuthApplication struct {
	ID                   OAuthApplicationID
	Version              int64
	SpaceID              *SpaceID
	CreatedSpaceMemberID *SpaceMemberID
	Name                 string
	ClientID             string
	ClientSecretDigest   *string
	ClientType           OAuthClientType
	RedirectURIs         []string
	DiscardedAt          *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// IsConfidentialはシークレットで認証するconfidentialクライアントであるかを返す
func (a *OAuthApplication) IsConfidential() bool {
	return a.ClientType == OAuthClientTypeConfidential
}

// IsOfficialClientは、スペースに属さない全スペース共通の公式クライアントであるかを返す
func (a *OAuthApplication) IsOfficialClient() bool {
	return a.SpaceID == nil
}

// IsAvailableInは、アプリをスペースspaceIDとの連携に使えるかを返す。スペースのアプリは
// 登録したスペースでだけ、公式クライアントはどのスペースでも使える
func (a *OAuthApplication) IsAvailableIn(spaceID SpaceID) bool {
	return a.IsOfficialClient() || *a.SpaceID == spaceID
}
