package model

import (
	"errors"
	"net/url"
	"time"
)

// OAuthAuthorizationCodeLifetimeは認可コードの有効期間。コードはリダイレクトの直後に
// トークンと交換されるもので、漏れたときに使える時間を短くする
const OAuthAuthorizationCodeLifetime = 5 * time.Minute

// spaceAPIResourcePathはスペースのAPIのURLのパスの接頭辞。OpenAPI記述の `servers`
// (`/api/v1`) にgetSpaceのパスを続けたもの
const spaceAPIResourcePath = "/api/v1/spaces/"

// SpaceAPIResourceURLは、OAuthのトークンの宛先 (RFC 8707の `resource`) としてスペースを表す
// URLを返す。値はスペースのAPIのURLで、appURLはWikinoのオリジン (例: `https://example.com`)
func SpaceAPIResourceURL(appURL string, identifier SpaceIdentifier) string {
	return appURL + spaceAPIResourcePath + url.PathEscape(string(identifier))
}

// ParseSpaceAPIResourceURLは、`resource` の値がSpaceAPIResourceURLの形であればスペースの
// 識別子を返す。SpaceAPIResourceURLが返す形と文字列で一致するものだけを受け付け、クエリ・
// フラグメント・末尾のスラッシュ・別のオリジンを含む値は受け付けない
func ParseSpaceAPIResourceURL(appURL string, resource string) (SpaceIdentifier, bool) {
	prefix := appURL + spaceAPIResourcePath
	if len(resource) <= len(prefix) || resource[:len(prefix)] != prefix {
		return "", false
	}
	identifier, err := url.PathUnescape(resource[len(prefix):])
	if err != nil {
		return "", false
	}
	if SpaceAPIResourceURL(appURL, SpaceIdentifier(identifier)) != resource {
		return "", false
	}
	return SpaceIdentifier(identifier), true
}

// OAuthAuthorizationErrorCodeは、認可要求の失敗をクライアントへ伝えるエラーコード
// (RFC 6749 §4.1.2.1、RFC 8707 §2)
type OAuthAuthorizationErrorCode string

const (
	// OAuthAuthorizationErrorInvalidRequestは必須のパラメーターの欠落・不正な値・重複を表す
	OAuthAuthorizationErrorInvalidRequest OAuthAuthorizationErrorCode = "invalid_request"
	// OAuthAuthorizationErrorAccessDeniedは利用者が許可しなかったことを表す
	OAuthAuthorizationErrorAccessDenied OAuthAuthorizationErrorCode = "access_denied"
	// OAuthAuthorizationErrorUnsupportedResponseTypeは `code` 以外のresponse_typeを表す
	OAuthAuthorizationErrorUnsupportedResponseType OAuthAuthorizationErrorCode = "unsupported_response_type"
	// OAuthAuthorizationErrorInvalidScopeは不明な・トークンに付与できないスコープを表す
	OAuthAuthorizationErrorInvalidScope OAuthAuthorizationErrorCode = "invalid_scope"
	// OAuthAuthorizationErrorInvalidTargetは `resource` の欠落・不正な値・連携できないスペースを表す
	OAuthAuthorizationErrorInvalidTarget OAuthAuthorizationErrorCode = "invalid_target"
)

// OAuthAuthorizationErrorは、認可要求をクライアントへエラーとして返すことを表す。
//
// RedirectURIが空のエラーは、クライアントIDかリダイレクトURIを確かめられなかったもので、
// クライアントへ戻してはならない (RFC 6749 §4.1.2.1)。その場合は利用者に画面で伝え、
// UserMsgにその文言を持つ。
type OAuthAuthorizationError struct {
	Code        OAuthAuthorizationErrorCode
	RedirectURI string
	State       string
	UserMsg     string
}

// Errorはerrorインターフェースを満たす
func (e *OAuthAuthorizationError) Error() string {
	if e.IsRedirectable() {
		return "OAuthの認可要求のエラー: " + string(e.Code)
	}
	return "OAuthの認可要求のエラー (クライアントへ戻さない): " + e.UserMsg
}

// IsRedirectableは、エラーをリダイレクトURIへ付けてクライアントへ戻せるかを返す
func (e *OAuthAuthorizationError) IsRedirectable() bool {
	return e.RedirectURI != ""
}

// AsOAuthAuthorizationErrorはerrがOAuthAuthorizationErrorなら取り出し、そうでなければnilを返す
func AsOAuthAuthorizationError(err error) *OAuthAuthorizationError {
	var oe *OAuthAuthorizationError
	if errors.As(err, &oe) {
		return oe
	}
	return nil
}

// AcceptsRedirectURIは、認可要求のリダイレクトURIuriが登録したものと一致するかを返す。
//
// 照合は文字列の完全一致で行い、正規化しない。ループバックのIPアドレス (127.0.0.1・[::1]) の
// HTTPのURIだけは、ポート番号を除いて一致すればよい。ネイティブアプリはリダイレクトを受ける
// ポートを起動のたびにOSから割り当てるため、登録の時点では決められない (RFC 8252 §7.3)。
func (a *OAuthApplication) AcceptsRedirectURI(uri string) bool {
	for _, registered := range a.RedirectURIs {
		if registered == uri {
			return true
		}
	}

	requested, ok := parseLoopbackRedirectURI(uri)
	if !ok {
		return false
	}
	for _, registered := range a.RedirectURIs {
		r, ok := parseLoopbackRedirectURI(registered)
		if !ok {
			continue
		}
		if r.Hostname() == requested.Hostname() &&
			r.EscapedPath() == requested.EscapedPath() &&
			r.RawQuery == requested.RawQuery {
			return true
		}
	}
	return false
}

// parseLoopbackRedirectURIは、uriがループバックのIPアドレスを指すHTTPのリダイレクトURIなら
// 解析した結果を返す。ユーザー情報・フラグメントを含むものはリダイレクトURIとして扱わない
func parseLoopbackRedirectURI(uri string) (*url.URL, bool) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, false
	}
	if u.Scheme != "http" || u.User != nil || u.Fragment != "" || u.RawFragment != "" {
		return nil, false
	}
	if host := u.Hostname(); host != "127.0.0.1" && host != "::1" {
		return nil, false
	}
	return u, true
}
