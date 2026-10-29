package model

import (
	"net/url"
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

// OAuthRegistrationMethodはOAuthアプリのクライアントの登録の方式
type OAuthRegistrationMethod int32

const (
	// OAuthRegistrationMethodRegisteredは、Wikinoに登録したクライアント (スペースのアプリ・公式クライアント)
	OAuthRegistrationMethodRegistered OAuthRegistrationMethod = 0
	// OAuthRegistrationMethodClientIDMetadataDocumentは、Client ID Metadata Document (CIMD) のクライアント。
	// `client_id` はクライアントが公開するHTTPSのURLで、認可要求のときにそこから取得した文書でアプリを作成・更新する
	OAuthRegistrationMethodClientIDMetadataDocument OAuthRegistrationMethod = 1
)

// OfficialCLIClientIDは公式CLIのクライアントID。CLIに埋め込んで配布するため変えない。
// クライアント自体はマイグレーション (db/migrations) で登録する
const OfficialCLIClientID = "wikino-cli"

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
// (CLI・デスクトップ) か、CIMDのクライアントである。CreatedSpaceMemberIDは作成したメンバーで、
// そのメンバーがいなくなるとnilになる。
//
// CIMDのクライアントは、取得した文書の `client_name` をName、`redirect_uris` をRedirectURIsに持ち、
// MetadataExpiresAtまで文書をキャッシュする。登録したクライアントのMetadataExpiresAtはnilである。
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
	RegistrationMethod   OAuthRegistrationMethod
	MetadataExpiresAt    *time.Time
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
	return a.SpaceID == nil && a.RegistrationMethod == OAuthRegistrationMethodRegistered
}

// IsClientIDMetadataDocumentは、CIMDのクライアントであるかを返す
func (a *OAuthApplication) IsClientIDMetadataDocument() bool {
	return a.RegistrationMethod == OAuthRegistrationMethodClientIDMetadataDocument
}

// ClientIDHostは、CIMDのクライアントのクライアントID (URL) のホスト名を返す。同意画面と連携の一覧で、
// 文書から取った名前に添えてクライアントの出どころを示すために使う。CIMDのクライアントでなければ空を返す
func (a *OAuthApplication) ClientIDHost() string {
	if !a.IsClientIDMetadataDocument() {
		return ""
	}
	u, err := url.Parse(a.ClientID)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// IsMetadataFreshは、CIMDの文書のキャッシュがnowの時点で期限内であるかを返す
func (a *OAuthApplication) IsMetadataFresh(now time.Time) bool {
	return a.MetadataExpiresAt != nil && now.Before(*a.MetadataExpiresAt)
}

// IsAvailableInは、アプリをスペースspaceIDとの連携に使えるかを返す。スペースのアプリは
// 登録したスペースでだけ、スペースに属さないアプリ (公式クライアント・CIMDのクライアント) は
// どのスペースでも使える
func (a *OAuthApplication) IsAvailableIn(spaceID SpaceID) bool {
	return a.SpaceID == nil || *a.SpaceID == spaceID
}
