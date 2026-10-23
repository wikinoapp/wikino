package auth

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/apiclient"
)

const testIssuer = "https://wikino.app"

// fakeAuthServerは、トークン・失効のエンドポイントとメタデータを返すテスト用の認可サーバー。
// URLを書き換えずにトランスポートとして差し込み、リフレッシュトークンのローテーションと
// 使用済みの再提示の拒否をサーバーと同じく行う。
type fakeAuthServer struct {
	mu sync.Mutex

	metadata map[string]any
	// authorizationsは、認可要求のstateごとの要求の値。
	authorizations map[string]url.Values
	// codesは、発行した認可コードごとの認可要求の値。
	codes map[string]url.Values
	// refreshTokensは、リフレッシュトークンごとの使用済みかどうか。
	refreshTokens map[string]bool
	issued        int
	// refreshCountは、受け付けたリフレッシュの数。usedRefreshCountは使用済みの再提示の数。
	refreshCount     int
	usedRefreshCount int
	revoked          []string
	requests         []*http.Request
	forms            []url.Values
	// deviceAuthorizationは、デバイス認可の要求に返す応答。deviceFormsは受けた要求の値。
	deviceAuthorization map[string]any
	deviceForms         []url.Values
	// deviceResultsは、デバイスコードのポーリングに順に返すエラーコード。空文字なら許可済みとしてトークンを発行し、
	// 使い切った後は `authorization_pending` を返す。
	deviceResults []string
}

func newFakeAuthServer() *fakeAuthServer {
	return &fakeAuthServer{
		metadata: map[string]any{
			"issuer":                                         testIssuer,
			"authorization_endpoint":                         testIssuer + "/oauth/authorize",
			"token_endpoint":                                 testIssuer + "/oauth/token",
			"revocation_endpoint":                            testIssuer + "/oauth/revoke",
			"device_authorization_endpoint":                  testIssuer + "/oauth/device_authorization",
			"scopes_supported":                               []string{"page:read", "page:write", "draft_page:read"},
			"code_challenge_methods_supported":               []string{"S256"},
			"authorization_response_iss_parameter_supported": true,
		},
		authorizations: map[string]url.Values{},
		codes:          map[string]url.Values{},
		refreshTokens:  map[string]bool{},
		deviceAuthorization: map[string]any{
			"device_code":               "device-code-secret",
			"user_code":                 "BCDF-GHJK",
			"verification_uri":          testIssuer + "/oauth/device",
			"verification_uri_complete": testIssuer + "/oauth/device?user_code=BCDF-GHJK",
			"expires_in":                600,
			"interval":                  5,
		},
	}
}

func (s *fakeAuthServer) client() *http.Client {
	return &http.Client{Transport: roundTripFunc(s.serve)}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func (s *fakeAuthServer) serve(req *http.Request) (*http.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.requests = append(s.requests, req)
	if req.URL.Host != "wikino.app" {
		return jsonResp(http.StatusNotFound, map[string]any{}), nil
	}

	switch req.Method + " " + req.URL.Path {
	case "GET /.well-known/oauth-authorization-server":
		return jsonResp(http.StatusOK, s.metadata), nil
	case "POST /oauth/token":
		form := readForm(req)
		s.forms = append(s.forms, form)
		return s.token(form), nil
	case "POST /oauth/device_authorization":
		form := readForm(req)
		s.deviceForms = append(s.deviceForms, form)
		if form.Get("client_id") != ClientID {
			return jsonResp(http.StatusUnauthorized, map[string]any{"error": "invalid_client"}), nil
		}
		return jsonResp(http.StatusOK, s.deviceAuthorization), nil
	case "POST /oauth/revoke":
		form := readForm(req)
		s.forms = append(s.forms, form)
		s.revoked = append(s.revoked, form.Get("token"))
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(""))}, nil
	}

	return jsonResp(http.StatusNotFound, map[string]any{}), nil
}

func (s *fakeAuthServer) token(form url.Values) *http.Response {
	if form.Get("client_id") != ClientID {
		return jsonResp(http.StatusUnauthorized, map[string]any{"error": "invalid_client"})
	}

	switch form.Get("grant_type") {
	case "authorization_code":
		request, ok := s.codes[form.Get("code")]
		if !ok {
			return jsonResp(http.StatusBadRequest, map[string]any{"error": "invalid_grant"})
		}
		delete(s.codes, form.Get("code"))
		sum := sha256.Sum256([]byte(form.Get("code_verifier")))
		if base64.RawURLEncoding.EncodeToString(sum[:]) != request.Get("code_challenge") ||
			form.Get("redirect_uri") != request.Get("redirect_uri") || form.Get("resource") != request.Get("resource") {
			return jsonResp(http.StatusBadRequest, map[string]any{"error": "invalid_grant"})
		}
		return s.issue("page:read draft_page:read")
	case "refresh_token":
		used, ok := s.refreshTokens[form.Get("refresh_token")]
		if !ok {
			return jsonResp(http.StatusBadRequest, map[string]any{"error": "invalid_grant"})
		}
		if used {
			s.usedRefreshCount++
			return jsonResp(http.StatusBadRequest, map[string]any{"error": "invalid_grant"})
		}
		s.refreshTokens[form.Get("refresh_token")] = true
		s.refreshCount++
		return s.issue("page:read draft_page:read")
	case "urn:ietf:params:oauth:grant-type:device_code":
		if form.Get("device_code") != s.deviceAuthorization["device_code"] {
			return jsonResp(http.StatusBadRequest, map[string]any{"error": "invalid_grant"})
		}
		result := "authorization_pending"
		if len(s.deviceResults) > 0 {
			result, s.deviceResults = s.deviceResults[0], s.deviceResults[1:]
		}
		if result == "" {
			return s.issue("page:read")
		}
		return jsonResp(http.StatusBadRequest, map[string]any{"error": result})
	}

	return jsonResp(http.StatusBadRequest, map[string]any{"error": "unsupported_grant_type"})
}

func (s *fakeAuthServer) issue(scope string) *http.Response {
	s.issued++
	refresh := fmt.Sprintf("wikino_oauth_refresh_%d", s.issued)
	s.refreshTokens[refresh] = false

	return jsonResp(http.StatusOK, map[string]any{
		"access_token":  fmt.Sprintf("wikino_oauth_access_%d", s.issued),
		"token_type":    "Bearer",
		"expires_in":    3600,
		"refresh_token": refresh,
		"scope":         scope,
	})
}

// addRefreshTokenは、ログイン済みの状態を作るため、有効なリフレッシュトークンを登録する。
func (s *fakeAuthServer) addRefreshToken(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.refreshTokens[token] = false
}

// authorizeは、ブラウザで許可した結果として、認可要求のURLから認可コードを発行し、
// リダイレクトURIに渡すクエリを返す。
func (s *fakeAuthServer) authorize(t *testing.T, authorizationURL string) (redirectURI string, query url.Values) {
	t.Helper()

	u, err := url.Parse(authorizationURL)
	if err != nil {
		t.Fatalf("認可要求のURLを解析できない: %v", err)
	}
	request := u.Query()

	s.mu.Lock()
	defer s.mu.Unlock()
	code := fmt.Sprintf("code-%d", len(s.authorizations)+1)
	s.authorizations[request.Get("state")] = request
	s.codes[code] = request

	return request.Get("redirect_uri"), url.Values{"code": {code}, "state": {request.Get("state")}, "iss": {testIssuer}}
}

func readForm(req *http.Request) url.Values {
	body, _ := io.ReadAll(req.Body)
	form, _ := url.ParseQuery(string(body))

	return form
}

func jsonResp(status int, v any) *http.Response {
	body, _ := json.Marshal(v)

	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

func TestFetchMetadataValidates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		modify  func(m map[string]any)
		wantErr error
	}{
		{name: "正しいメタデータを受け付ける", modify: func(map[string]any) {}},
		{name: "発行者が接続先と違えば拒否する", modify: func(m map[string]any) { m["issuer"] = "https://evil.example" }, wantErr: ErrInvalidMetadata},
		{name: "トークンエンドポイントが別のオリジンなら拒否する", modify: func(m map[string]any) { m["token_endpoint"] = "https://evil.example/oauth/token" }, wantErr: ErrInvalidMetadata},
		{name: "HTTPのエンドポイントを拒否する", modify: func(m map[string]any) { m["authorization_endpoint"] = "http://wikino.app/oauth/authorize" }, wantErr: ErrInvalidMetadata},
		{name: "デバイス認可のエンドポイントが無くても受け付ける", modify: func(m map[string]any) { delete(m, "device_authorization_endpoint") }},
		{name: "デバイス認可のエンドポイントが別のオリジンなら拒否する", modify: func(m map[string]any) {
			m["device_authorization_endpoint"] = "https://evil.example/oauth/device_authorization"
		}, wantErr: ErrInvalidMetadata},
		{name: "S256に対応しなければ拒否する", modify: func(m map[string]any) { m["code_challenge_methods_supported"] = []string{"plain"} }, wantErr: ErrInvalidMetadata},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := newFakeAuthServer()
			tt.modify(server.metadata)
			client := &OAuthClient{Host: testIssuer, HTTPClient: server.client()}

			metadata, err := client.FetchMetadata(context.Background())
			if tt.wantErr == nil {
				if err != nil || metadata.TokenEndpoint != testIssuer+"/oauth/token" {
					t.Fatalf("メタデータ = %+v, エラー = %v", metadata, err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("エラー = %v、期待値 = %v", err, tt.wantErr)
			}
		})
	}
}

func TestFetchMetadataNotJSON(t *testing.T) {
	t.Parallel()

	client := &OAuthClient{Host: testIssuer, HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"text/html"}}, Body: io.NopCloser(strings.NewReader("<html>"))}, nil
	})}}

	if _, err := client.FetchMetadata(context.Background()); !errors.Is(err, ErrUnexpectedResponse) {
		t.Errorf("エラー = %v、期待値 = ErrUnexpectedResponse", err)
	}
}

func TestPKCE(t *testing.T) {
	t.Parallel()

	a, b := NewPKCE(), NewPKCE()
	if a.Verifier == b.Verifier || len(a.Verifier) != 43 {
		t.Errorf("コード検証子が乱数の43文字でない: %q, %q", a.Verifier, b.Verifier)
	}
	sum := sha256.Sum256([]byte(a.Verifier))
	if a.Challenge != base64.RawURLEncoding.EncodeToString(sum[:]) {
		t.Error("コードチャレンジがS256でない")
	}
	if first, second := NewState(), NewState(); first == second {
		t.Error("stateが毎回変わらない")
	}
}

func TestAuthorizationURL(t *testing.T) {
	t.Parallel()

	metadata := &Metadata{AuthorizationEndpoint: testIssuer + "/oauth/authorize"}
	got := metadata.AuthorizationURL(AuthorizationRequest{
		RedirectURI: "http://127.0.0.1:5000/callback",
		Scopes:      []string{"page:read", "draft_page:read"},
		State:       "state-value",
		PKCE:        PKCE{Verifier: "verifier-value", Challenge: "challenge-value"},
		Resource:    ResourceURL(testIssuer, "team"),
	})

	u, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	want := url.Values{
		"response_type":         {"code"},
		"client_id":             {"wikino-cli"},
		"redirect_uri":          {"http://127.0.0.1:5000/callback"},
		"scope":                 {"page:read draft_page:read"},
		"state":                 {"state-value"},
		"code_challenge":        {"challenge-value"},
		"code_challenge_method": {"S256"},
		"resource":              {"https://wikino.app/api/v1/spaces/team"},
	}
	if u.Path != "/oauth/authorize" || u.Query().Encode() != want.Encode() {
		t.Errorf("認可要求のURL = %s", got)
	}
	if strings.Contains(got, "verifier-value") {
		t.Error("認可要求のURLにコード検証子が含まれる")
	}
}

func TestRefreshAndRevokeRequests(t *testing.T) {
	t.Parallel()

	server := newFakeAuthServer()
	server.addRefreshToken("refresh-old")
	client := &OAuthClient{Host: testIssuer, HTTPClient: server.client(), UserAgent: "wikino-cli/test"}
	metadata, err := client.FetchMetadata(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	token, err := client.Refresh(context.Background(), metadata, "refresh-old", ResourceURL(testIssuer, "team"))
	if err != nil {
		t.Fatalf("リフレッシュに失敗した: %v", err)
	}
	if token.RefreshToken == "refresh-old" || token.AccessToken == "" {
		t.Errorf("トークン = %+v", token)
	}
	form := server.forms[0]
	if form.Get("grant_type") != "refresh_token" || form.Get("resource") != "https://wikino.app/api/v1/spaces/team" || form.Get("client_id") != ClientID {
		t.Errorf("リフレッシュの要求 = %v", form)
	}

	// 使用済みのリフレッシュトークンの再提示は、エラーコードを持つエラーになる。
	_, err = client.Refresh(context.Background(), metadata, "refresh-old", ResourceURL(testIssuer, "team"))
	var oauthErr *OAuthError
	if !errors.As(err, &oauthErr) || oauthErr.Code != "invalid_grant" {
		t.Errorf("エラー = %v、期待値 = invalid_grant", err)
	}

	if err := client.Revoke(context.Background(), metadata, token.RefreshToken); err != nil {
		t.Fatalf("失効に失敗した: %v", err)
	}
	if len(server.revoked) != 1 || server.revoked[0] != token.RefreshToken {
		t.Errorf("失効させたトークン = %v", server.revoked)
	}
	for _, req := range server.requests {
		if req.Header.Get("User-Agent") != "wikino-cli/test" || req.Header.Get("Authorization") != "" {
			t.Errorf("%s %s のヘッダー = %v", req.Method, req.URL, req.Header)
		}
	}
}

func TestOAuthRejectsRedirects(t *testing.T) {
	t.Parallel()
	operations := []struct {
		name string
		path string
		call func(*OAuthClient, *Metadata) error
	}{
		{"メタデータ", metadataPath, func(c *OAuthClient, _ *Metadata) error {
			_, err := c.FetchMetadata(context.Background())
			return err
		}},
		{"認可コードの交換", "/oauth/token", func(c *OAuthClient, m *Metadata) error {
			_, err := c.ExchangeCode(context.Background(), m, "code-secret", "http://127.0.0.1:5000/callback", "verifier-secret", ResourceURL(testIssuer, "team"))
			return err
		}},
		{"リフレッシュ", "/oauth/token", func(c *OAuthClient, m *Metadata) error {
			_, err := c.Refresh(context.Background(), m, "refresh-secret", ResourceURL(testIssuer, "team"))
			return err
		}},
		{"失効", "/oauth/revoke", func(c *OAuthClient, m *Metadata) error {
			return c.Revoke(context.Background(), m, "refresh-secret")
		}},
	}
	for _, op := range operations {
		for _, status := range []int{301, 302, 303, 307, 308} {
			for _, target := range []string{testIssuer + "/redirected", "https://other.example/capture"} {
				t.Run(fmt.Sprintf("%s/%d/%s", op.name, status, target), func(t *testing.T) {
					t.Parallel()
					requests := 0
					redirectPolicyCalled := false
					hc := apiclient.NewHTTPClient(roundTripFunc(func(req *http.Request) (*http.Response, error) {
						requests++
						if requests > 1 {
							t.Errorf("リダイレクト先へ要求した: %s %s", req.Method, req.URL)
							return jsonResp(http.StatusBadRequest, map[string]any{"error": "invalid_grant"}), nil
						}
						if req.URL.String() != testIssuer+op.path {
							t.Errorf("要求先 = %s、期待値 = %s", req.URL, testIssuer+op.path)
						}
						return &http.Response{StatusCode: status, Header: http.Header{"Location": {target}}, Body: io.NopCloser(strings.NewReader(""))}, nil
					}))
					hc.CheckRedirect = func(*http.Request, []*http.Request) error { redirectPolicyCalled = true; return nil }
					c := &OAuthClient{Host: testIssuer, HTTPClient: hc}
					m := &Metadata{TokenEndpoint: testIssuer + "/oauth/token", RevocationEndpoint: testIssuer + "/oauth/revoke"}
					if err := op.call(c, m); !errors.Is(err, ErrUnexpectedResponse) {
						t.Errorf("エラー = %v、期待値 = ErrUnexpectedResponse", err)
					}
					if requests != 1 {
						t.Errorf("送信回数 = %d、期待値 = 1", requests)
					}
					if redirectPolicyCalled {
						t.Error("元のクライアントのリダイレクトポリシーを呼び出した")
					}
					// OAuthが共有クライアントの設定を書き換えると、API通信の挙動も変わってしまう。
					if err := hc.CheckRedirect(nil, nil); err != nil || !redirectPolicyCalled {
						t.Error("共有クライアントのリダイレクトポリシーが書き換わった")
					}
				})
			}
		}
	}
}
