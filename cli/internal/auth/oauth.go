package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/wikinoapp/wikino/cli/internal/apiclient"
)

// ClientIDは、公式CLIのOAuthクライアントのID。サーバーのマイグレーションで登録した値
// (model.OfficialCLIClientID) と同じで、変えない。
const ClientID = "wikino-cli"

// metadataPathは、認可サーバーのメタデータ (RFC 8414) のパス。
const metadataPath = "/.well-known/oauth-authorization-server"

// spaceAPIResourcePathは、`resource` (RFC 8707) に載せるスペースのAPIのURLの、接続先からのパス。
const spaceAPIResourcePath = "/api/v1/spaces/"

// maxOAuthResponseBytesは、認可サーバーの応答を読む上限。
const maxOAuthResponseBytes = 1 << 20

// Metadataは、認可サーバーのメタデータのうちCLIが使う項目。
type Metadata struct {
	Issuer                                     string   `json:"issuer"`
	AuthorizationEndpoint                      string   `json:"authorization_endpoint"`
	TokenEndpoint                              string   `json:"token_endpoint"`
	RevocationEndpoint                         string   `json:"revocation_endpoint"`
	DeviceAuthorizationEndpoint                string   `json:"device_authorization_endpoint"`
	ScopesSupported                            []string `json:"scopes_supported"`
	CodeChallengeMethodsSupported              []string `json:"code_challenge_methods_supported"`
	AuthorizationResponseIssParameterSupported bool     `json:"authorization_response_iss_parameter_supported"`
}

// TokenResponseは、トークンエンドポイントの応答 (RFC 6749 §5.1)。
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// OAuthErrorは、認可サーバーがエラーコード (RFC 6749 §4.1.2.1・§5.2) で失敗を返したことを表す。
type OAuthError struct {
	// Codeはエラーコード (`invalid_grant` など)。
	Code string
	// StatusCodeはトークンエンドポイントの応答のステータス。認可の応答のエラーでは0。
	StatusCode int
}

func (e *OAuthError) Error() string {
	if e.StatusCode == 0 {
		return "認可サーバーがエラーを返しました: " + e.Code
	}
	return fmt.Sprintf("認可サーバーがエラーを返しました: %s (%d)", e.Code, e.StatusCode)
}

// ErrInvalidMetadataは、認可サーバーのメタデータが接続先と食い違うか、必要な項目を欠くことを表す。
var ErrInvalidMetadata = errors.New("認可サーバーのメタデータが不正です")

// ErrUnexpectedResponseは、認可サーバーの応答を想定した形で読めないことを表す。
var ErrUnexpectedResponse = errors.New("認可サーバーの応答が不正です")

// OAuthClientは、公式クライアントとして認可サーバーと通信する。
type OAuthClient struct {
	// Hostは接続先のURL (apiclient.ValidateHostで検証済みのもの)。
	Host string
	// HTTPClientは要求を送るクライアント。リダイレクトを拒否したコピーを使う。
	HTTPClient *http.Client
	// UserAgentはUser-Agentヘッダーの値。
	UserAgent string
}

// FetchMetadataは、認可サーバーのメタデータを取得する。エンドポイントはCLIに埋め込まず、ここから得る。
// 発行者が接続先と一致し (RFC 8414 §3.3)、エンドポイントが接続先と同じオリジンにあることを確かめ、
// トークンを別のオリジンへ送らないようにする。
func (c *OAuthClient) FetchMetadata(ctx context.Context) (*Metadata, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Host+metadataPath, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: メタデータの取得が%dで失敗しました", ErrUnexpectedResponse, resp.StatusCode)
	}

	var metadata Metadata
	if err := decodeJSON(resp, &metadata); err != nil {
		return nil, err
	}
	if err := metadata.validate(c.Host); err != nil {
		return nil, err
	}

	return &metadata, nil
}

func (m *Metadata) validate(host string) error {
	if strings.TrimSuffix(m.Issuer, "/") != host {
		return fmt.Errorf("%w: 発行者 %q が接続先と一致しません", ErrInvalidMetadata, m.Issuer)
	}
	for _, endpoint := range []string{m.AuthorizationEndpoint, m.TokenEndpoint, m.RevocationEndpoint} {
		if !sameOrigin(host, endpoint) {
			return fmt.Errorf("%w: エンドポイント %q が接続先と別のオリジンにあります", ErrInvalidMetadata, endpoint)
		}
	}
	// デバイス認可に対応しない認可サーバーでも、ループバックではログインできるため省略を許す。
	if m.DeviceAuthorizationEndpoint != "" && !sameOrigin(host, m.DeviceAuthorizationEndpoint) {
		return fmt.Errorf("%w: エンドポイント %q が接続先と別のオリジンにあります", ErrInvalidMetadata, m.DeviceAuthorizationEndpoint)
	}
	// 公開クライアントの認可コードはPKCEで守るため、S256に対応しない認可サーバーとは通信しない。
	if !slices.Contains(m.CodeChallengeMethodsSupported, codeChallengeMethod) {
		return fmt.Errorf("%w: PKCEのS256に対応していません", ErrInvalidMetadata)
	}

	return nil
}

// sameOriginは、rawがhostと同じスキームとホストのURLかを返す。
func sameOrigin(host, raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || u.Fragment != "" {
		return false
	}

	return strings.ToLower(u.Scheme)+"://"+strings.ToLower(u.Host) == strings.ToLower(host)
}

// ResourceURLは、spaceのAPIを指す `resource` の値を返す。サーバーのmodel.SpaceAPIResourceURLと同じ形にする。
func ResourceURL(host, space string) string {
	return host + spaceAPIResourcePath + url.PathEscape(space)
}

// codeChallengeMethodは、PKCEのコードチャレンジの方式。
const codeChallengeMethod = "S256"

// PKCEは、認可コードの横取りを防ぐコード検証子とコードチャレンジ (RFC 7636)。
type PKCE struct {
	Verifier  string
	Challenge string
}

// NewPKCEは、32バイトの乱数からコード検証子を作り、S256のコードチャレンジを添えて返す。
func NewPKCE() PKCE {
	verifier := randomString()
	sum := sha256.Sum256([]byte(verifier))

	return PKCE{Verifier: verifier, Challenge: base64.RawURLEncoding.EncodeToString(sum[:])}
}

// NewStateは、認可要求とリダイレクトを結び付けるstateの値を作る。
func NewState() string {
	return randomString()
}

func randomString() string {
	b := make([]byte, 32)
	// crypto/randのReadは失敗しない (Go 1.24以降はエラーを返さずにプロセスを止める)。
	_, _ = rand.Read(b)

	return base64.RawURLEncoding.EncodeToString(b)
}

// AuthorizationRequestは、認可エンドポイントへの要求の値。
type AuthorizationRequest struct {
	RedirectURI string
	Scopes      []string
	State       string
	PKCE        PKCE
	Resource    string
}

// AuthorizationURLは、ブラウザで開く認可要求のURLを返す。
func (m *Metadata) AuthorizationURL(r AuthorizationRequest) string {
	query := url.Values{
		"response_type":         {"code"},
		"client_id":             {ClientID},
		"redirect_uri":          {r.RedirectURI},
		"scope":                 {strings.Join(r.Scopes, " ")},
		"state":                 {r.State},
		"code_challenge":        {r.PKCE.Challenge},
		"code_challenge_method": {codeChallengeMethod},
		"resource":              {r.Resource},
	}

	separator := "?"
	if strings.Contains(m.AuthorizationEndpoint, "?") {
		separator = "&"
	}

	return m.AuthorizationEndpoint + separator + query.Encode()
}

// ExchangeCodeは、認可コードをトークンに交換する。
func (c *OAuthClient) ExchangeCode(ctx context.Context, metadata *Metadata, code, redirectURI, verifier, resource string) (*TokenResponse, error) {
	return c.requestToken(ctx, metadata, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {verifier},
		"client_id":     {ClientID},
		"resource":      {resource},
	})
}

// Refreshは、リフレッシュトークンで新しいアクセストークンとリフレッシュトークンを得る。
// サーバーはリフレッシュトークンをローテーションし、使用済みの再提示で許可のトークンをすべて失効させる。
func (c *OAuthClient) Refresh(ctx context.Context, metadata *Metadata, refreshToken, resource string) (*TokenResponse, error) {
	return c.requestToken(ctx, metadata, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {ClientID},
		"resource":      {resource},
	})
}

// Revokeは、トークンを失効させる (RFC 7009)。無効なトークンにもサーバーは成功を返す。
func (c *OAuthClient) Revoke(ctx context.Context, metadata *Metadata, token string) error {
	resp, err := c.postForm(ctx, metadata.RevocationEndpoint, url.Values{
		"token":           {token},
		"token_type_hint": {"refresh_token"},
		"client_id":       {ClientID},
	})
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return oauthResponseError(resp)
	}

	return nil
}

func (c *OAuthClient) requestToken(ctx context.Context, metadata *Metadata, form url.Values) (*TokenResponse, error) {
	resp, err := c.postForm(ctx, metadata.TokenEndpoint, form)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, oauthResponseError(resp)
	}

	var token TokenResponse
	if err := decodeJSON(resp, &token); err != nil {
		return nil, err
	}
	if token.AccessToken == "" || token.RefreshToken == "" || !strings.EqualFold(token.TokenType, "Bearer") || token.ExpiresIn <= 0 {
		return nil, fmt.Errorf("%w: トークンの応答に必要な項目がありません", ErrUnexpectedResponse)
	}

	return &token, nil
}

func (c *OAuthClient) postForm(ctx context.Context, endpoint string, form url.Values) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	return c.do(req)
}

func (c *OAuthClient) do(req *http.Request) (*http.Response, error) {
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = apiclient.NewHTTPClient(nil)
	}

	// OAuthのフォーム本文には秘密の値が入るため、同じオリジンも含め転送しない。
	// 注入されたクライアントを共有するAPI通信には影響させない。
	client := *httpClient
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	return client.Do(req) // #nosec G704 -- 接続先とメタデータのURLは検証済みで、リダイレクトも拒否する
}

// oauthResponseErrorは、トークン・失効のエンドポイントのエラー応答を読む。
// エラーコードを読めなければ (プロキシのエラーページなど)、応答が不正なことを返す。
func oauthResponseError(resp *http.Response) error {
	var body struct {
		Error string `json:"error"`
	}
	if err := decodeJSON(resp, &body); err == nil && body.Error != "" {
		return &OAuthError{Code: body.Error, StatusCode: resp.StatusCode}
	}

	return fmt.Errorf("%w: ステータス%d", ErrUnexpectedResponse, resp.StatusCode)
}

func decodeJSON(resp *http.Response, v any) error {
	mediaType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if mediaType != "application/json" {
		return fmt.Errorf("%w: JSONでない応答です (%q)", ErrUnexpectedResponse, mediaType)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxOAuthResponseBytes)).Decode(v); err != nil {
		return fmt.Errorf("%w: %w", ErrUnexpectedResponse, err)
	}

	return nil
}

// expiresAtは、応答を受け取った時刻nowとexpires_inから、アクセストークンの期限を返す。
func (t *TokenResponse) expiresAt(now time.Time) time.Time {
	return now.Add(time.Duration(t.ExpiresIn) * time.Second)
}

// scopesは、付与されたスコープを返す。
func (t *TokenResponse) scopes() []string {
	return strings.Fields(t.Scope)
}
