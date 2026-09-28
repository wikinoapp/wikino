package model

import (
	"net/url"
	"strings"
)

// OAuthのエンドポイントのパス。ルーティングと認可サーバーのメタデータ (RFC 8414) が共有し、
// メタデータが指すURLと実際に受け付けるパスがずれないようにする
const (
	// OAuthAuthorizationEndpointPathは認可エンドポイント (同意画面) のパス
	OAuthAuthorizationEndpointPath = "/oauth/authorize"
	// OAuthTokenEndpointPathはトークンエンドポイントのパス
	OAuthTokenEndpointPath = "/oauth/token"
	// OAuthRevocationEndpointPathはトークンの失効のエンドポイント (RFC 7009) のパス
	OAuthRevocationEndpointPath = "/oauth/revoke"
)

// メタデータのパス (RFC 8414・RFC 9728・RFC 9727)
const (
	// OAuthAuthorizationServerMetadataPathは認可サーバーのメタデータのパス。
	// 認可サーバーの識別子 (issuer) はパスを含まないため、well-knownの名前だけになる
	OAuthAuthorizationServerMetadataPath = "/.well-known/oauth-authorization-server"
	// oauthProtectedResourceMetadataPathは保護リソースのメタデータのパスの接頭辞。
	// リソースの識別子のパスをこの後ろに続ける (RFC 9728 §3.1)
	oauthProtectedResourceMetadataPath = "/.well-known/oauth-protected-resource"
	// APICatalogPathはAPIカタログのパス
	APICatalogPath = "/.well-known/api-catalog"
)

// APIBasePathは公開Web APIのv1のパス。OpenAPI記述の `servers` と同じ値
const APIBasePath = "/api/v1"

// APIOpenAPIDescriptionPathはOpenAPI記述を配信するパス
const APIOpenAPIDescriptionPath = APIBasePath + "/openapi.yaml"

// APIReferencePathは、v1のOpenAPI記述を描画する人間向けのAPIリファレンスのパス。
// 新しいバージョンを出したときに版ごとのURLを並べられるよう、パスにバージョンを含める。
// `/api/v1` の下はAPIのリソースの名前空間 (OpenAPI記述の `servers`) のため、その外に置く
const APIReferencePath = "/api/reference/v1"

// 認可サーバーが受け付ける値。要求の検証と認可サーバーのメタデータが共有する
const (
	// OAuthResponseTypeCodeは認可コードグラントのresponse_type。ほかのresponse_typeは受け付けない
	OAuthResponseTypeCode = "code"
	// OAuthGrantTypeAuthorizationCodeは認可コードを交換するグラント (RFC 6749 §4.1.3)
	OAuthGrantTypeAuthorizationCode = "authorization_code"
	// OAuthGrantTypeRefreshTokenはリフレッシュトークンで更新するグラント (RFC 6749 §6)
	OAuthGrantTypeRefreshToken = "refresh_token"
	// OAuthCodeChallengeMethodS256はPKCEのcode_challenge_method。plainは受け付けない
	OAuthCodeChallengeMethodS256 = "S256"
)

// OAuthClientAuthMethodsは、トークンエンドポイントとトークンの失効のエンドポイントで受け付ける
// クライアントの認証の方式 (RFC 8414 §2)。confidentialクライアントはBasicか本文でシークレットを送り、
// publicクライアント (公式CLI・デスクトップ) は認証しない (`none`)
var OAuthClientAuthMethods = []string{"client_secret_basic", "client_secret_post", "none"}

// SpaceAPIResourceMetadataURLは、スペースのAPI (SpaceAPIResourceURL) の保護リソースのメタデータの
// URLを返す。well-knownの名前をオリジンとリソースのパスの間に挟む (RFC 9728 §3.1)
func SpaceAPIResourceMetadataURL(appURL string, identifier SpaceIdentifier) string {
	return appURL + oauthProtectedResourceMetadataPath + spaceAPIResourcePath + url.PathEscape(string(identifier))
}

// SpaceIdentifierFromAPIPathは、公開APIのリクエストのパス (エスケープされた形) がスペース配下の
// API (`/api/v1/spaces/{space_identifier}` とその下) を指していれば、スペースの識別子を返す
func SpaceIdentifierFromAPIPath(escapedPath string) (SpaceIdentifier, bool) {
	rest, ok := strings.CutPrefix(escapedPath, spaceAPIResourcePath)
	if !ok {
		return "", false
	}
	segment, _, _ := strings.Cut(rest, "/")
	identifier, err := url.PathUnescape(segment)
	if err != nil || identifier == "" {
		return "", false
	}
	return SpaceIdentifier(identifier), true
}
