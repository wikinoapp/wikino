package auth

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"
)

// fakeOAuthは、テスト用の認可サーバー。認可コードの交換・リフレッシュ (ローテーション)・失効を受け、
// 発行したアクセストークンをaliceのトークンとしてAPIで使えるようにする。
type fakeOAuth struct {
	codes         map[string]url.Values
	refreshTokens map[string]bool
	issued        []string
	revoked       []string
	// revokeStatusが0でなければ、失効の要求にそのステータスを返す。
	revokeStatus int
	// authorizationsは受けた認可要求の値。
	authorizations []url.Values
	verifiers      []string
	// deviceRequestsは受けたデバイス認可の要求の値。
	deviceRequests []url.Values
	// deviceResultsは、デバイスコードのポーリングに順に返すエラーコード。空文字なら許可済みとしてトークンを発行し、
	// 使い切った後は `authorization_pending` を返す。
	deviceResults []string
	// devicePollsは受けたデバイスコードのポーリングの数。
	devicePolls int
}

// testDeviceCodeは、テスト用の認可サーバーが発行するデバイスコード。出力に出てはならない。
const testDeviceCode = "device-code-secret"

func newFakeOAuth() *fakeOAuth {
	return &fakeOAuth{codes: map[string]url.Values{}, refreshTokens: map[string]bool{}}
}

func (o *fakeOAuth) serve(req *http.Request, owners map[string]tokenOwner) *http.Response {
	switch req.Method + " " + req.URL.Path {
	case "GET /.well-known/oauth-authorization-server":
		return jsonResponse(map[string]any{
			"issuer":                                         "https://wikino.app",
			"authorization_endpoint":                         "https://wikino.app/oauth/authorize",
			"token_endpoint":                                 "https://wikino.app/oauth/token",
			"revocation_endpoint":                            "https://wikino.app/oauth/revoke",
			"device_authorization_endpoint":                  "https://wikino.app/oauth/device_authorization",
			"scopes_supported":                               []string{"page:read", "page:write", "draft_page:read"},
			"code_challenge_methods_supported":               []string{"S256"},
			"authorization_response_iss_parameter_supported": true,
		})
	case "POST /oauth/token":
		form := oauthForm(req)
		switch form.Get("grant_type") {
		case "authorization_code":
			request, ok := o.codes[form.Get("code")]
			sum := sha256.Sum256([]byte(form.Get("code_verifier")))
			if !ok || base64.RawURLEncoding.EncodeToString(sum[:]) != request.Get("code_challenge") {
				return oauthError(http.StatusBadRequest, "invalid_grant")
			}
			delete(o.codes, form.Get("code"))
			o.verifiers = append(o.verifiers, form.Get("code_verifier"))
			return o.issue(owners, request.Get("scope"))
		case "refresh_token":
			if used, ok := o.refreshTokens[form.Get("refresh_token")]; !ok || used {
				return oauthError(http.StatusBadRequest, "invalid_grant")
			}
			o.refreshTokens[form.Get("refresh_token")] = true
			return o.issue(owners, "page:read")
		case "urn:ietf:params:oauth:grant-type:device_code":
			if form.Get("device_code") != testDeviceCode {
				return oauthError(http.StatusBadRequest, "invalid_grant")
			}
			o.devicePolls++
			result := "authorization_pending"
			if len(o.deviceResults) > 0 {
				result, o.deviceResults = o.deviceResults[0], o.deviceResults[1:]
			}
			if result != "" {
				return oauthError(http.StatusBadRequest, result)
			}
			return o.issue(owners, o.deviceRequests[len(o.deviceRequests)-1].Get("scope"))
		}
		return oauthError(http.StatusBadRequest, "unsupported_grant_type")
	case "POST /oauth/device_authorization":
		o.deviceRequests = append(o.deviceRequests, oauthForm(req))
		return jsonResponse(map[string]any{
			"device_code":               testDeviceCode,
			"user_code":                 "BCDF-GHJK",
			"verification_uri":          "https://wikino.app/oauth/device",
			"verification_uri_complete": "https://wikino.app/oauth/device?user_code=BCDF-GHJK",
			"expires_in":                600,
			"interval":                  5,
		})
	case "POST /oauth/revoke":
		if o.revokeStatus != 0 {
			return problem(o.revokeStatus, "失効できません。")
		}
		o.revoked = append(o.revoked, oauthForm(req).Get("token"))
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(nil))}
	}

	return problem(http.StatusNotFound, "見つかりません。")
}

func (o *fakeOAuth) issue(owners map[string]tokenOwner, scope string) *http.Response {
	n := len(o.issued)/2 + 1
	access := fmt.Sprintf("wikino_oauth_access-%d", n)
	refresh := fmt.Sprintf("wikino_oauth_refresh-%d", n)
	o.issued = append(o.issued, access, refresh)
	o.refreshTokens[refresh] = false
	owners[access] = owners[aliceToken]

	return jsonResponse(map[string]any{
		"access_token": access, "token_type": "Bearer", "expires_in": 3600, "refresh_token": refresh, "scope": scope,
	})
}

// secretsは、出力に出てはならない値 (発行したトークン・認可コード・コード検証子) を返す。
func (o *fakeOAuth) secrets() []string {
	secrets := append([]string{testDeviceCode}, o.issued...)
	for code := range o.codes {
		secrets = append(secrets, code)
	}

	return append(secrets, o.verifiers...)
}

func oauthForm(req *http.Request) url.Values {
	body, _ := io.ReadAll(req.Body)
	form, _ := url.ParseQuery(string(body))

	return form
}

func oauthError(status int, code string) *http.Response {
	resp := jsonResponse(map[string]any{"error": code})
	resp.StatusCode = status

	return resp
}

// approveは、ブラウザで許可した結果として認可コードを発行し、リダイレクトURIへ届ける。
// modifyがあれば、届けるクエリを書き換える。
func (e *testEnv) approve(authorizationURL string, modify func(url.Values)) string {
	u, err := url.Parse(authorizationURL)
	if err != nil {
		e.t.Errorf("認可要求のURLを解析できない: %v", err)
		return ""
	}
	request := u.Query()
	e.oauth.authorizations = append(e.oauth.authorizations, request)
	code := fmt.Sprintf("auth-code-%d", len(e.oauth.authorizations))
	e.oauth.codes[code] = request

	query := url.Values{"code": {code}, "state": {request.Get("state")}, "iss": {"https://wikino.app"}}
	if modify != nil {
		modify(query)
	}
	resp, err := http.Get(request.Get("redirect_uri") + "?" + query.Encode())
	if err != nil {
		e.t.Errorf("リダイレクトURIに届かない: %v", err)
		return ""
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		e.t.Errorf("結果画面を読めない: %v", err)
	}
	return string(body)
}

// loginWithBrowserは、ブラウザで許可してspaceにログインする。
func (e *testEnv) loginWithBrowser(space string, args ...string) {
	e.t.Helper()

	if err := e.run("", append([]string{"auth", "login", space}, args...)...); err != nil {
		e.t.Fatalf("ログインに失敗した: %v", err)
	}
}

// savedSecretは、キーチェーンに保存した唯一のトークンを返す。
func (e *testEnv) savedSecret(t *testing.T) map[string]any {
	t.Helper()

	if len(e.keyring.items) != 1 {
		t.Fatalf("キーチェーンの項目の数 = %d、期待値 = 1", len(e.keyring.items))
	}
	var secret map[string]any
	for _, v := range e.keyring.items {
		if err := json.Unmarshal([]byte(v), &secret); err != nil {
			t.Fatal(err)
		}
	}

	return secret
}
