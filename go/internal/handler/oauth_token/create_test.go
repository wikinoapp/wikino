package oauth_token_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/oauth_token"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// testCodeVerifierとtestCodeChallengeはPKCEの例 (RFC 7636 Appendix B)
const (
	testCodeVerifier  = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	testCodeChallenge = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	testRedirectURI   = "http://127.0.0.1:53123/callback"
	// testClientSecretは、URLエンコードが要る文字を含むconfidentialクライアントのシークレット
	testClientSecret = "wks_secret+/="
)

// tokenFixtureは、スペースのメンバーがOAuthアプリに与えた許可と、その許可に属する認可コード
type tokenFixture struct {
	clientID string
	code     string
	spaceID  model.SpaceID
	grantID  model.OAuthGrantID
}

// setupTokenFixtureは、スペースkeyのOAuthアプリと許可・認可コードを作る。confidentialなら
// testClientSecretで認証するアプリにする
func setupTokenFixture(t *testing.T, tx *sql.Tx, key string, confidential bool) tokenFixture {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail(key + "@example.com").
		WithAtname(strings.ReplaceAll(key, "-", "_")).
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier(key).Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithScopes([]model.Scope{model.ScopePageWrite, model.ScopeOAuthGrantWrite}).
		Build()
	testutil.NewFeatureFlagBuilder(t, tx).WithUserID(userID).WithName(string(model.FeatureFlagPublicAPI)).Build()

	clientID := key + "-client"
	appBuilder := testutil.NewOAuthApplicationBuilder(t, tx).WithSpaceID(spaceID).WithClientID(clientID)
	if confidential {
		appBuilder = appBuilder.WithConfidentialClientSecretDigest(auth.DigestOpaqueToken(testClientSecret))
	}
	appID := appBuilder.Build()
	grantID := testutil.NewOAuthGrantBuilder(t, tx).
		WithOAuthApplicationID(appID).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		WithScopes([]model.Scope{model.ScopeTopicRead, model.ScopePageRead}).
		Build()

	code := key + "-code"
	testutil.NewOAuthAuthorizationCodeBuilder(t, tx).
		WithOAuthGrantID(grantID).
		WithSpaceID(spaceID).
		WithCodeDigest(auth.DigestOpaqueToken(code)).
		WithScopes([]model.Scope{model.ScopeTopicRead, model.ScopePageRead}).
		WithRedirectURI(testRedirectURI).
		WithCodeChallenge(testCodeChallenge).
		Build()

	return tokenFixture{clientID: clientID, code: code, spaceID: spaceID, grantID: grantID}
}

// codeFormは、fの認可コードを交換する本文を返す。クライアントの資格情報は含めない
func (f tokenFixture) codeForm() url.Values {
	return url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {f.code},
		"redirect_uri":  {testRedirectURI},
		"code_verifier": {testCodeVerifier},
	}
}

func setupHandler(t *testing.T, q *query.Queries) *oauth_token.Handler {
	t.Helper()

	oauthApplicationRepo := repository.NewOAuthApplicationRepository(q)
	oauthGrantRepo := repository.NewOAuthGrantRepository(q)
	oauthAccessTokenRepo := repository.NewOAuthAccessTokenRepository(q)
	oauthRefreshTokenRepo := repository.NewOAuthRefreshTokenRepository(q)

	return oauth_token.NewHandler(
		usecase.NewCreateOAuthTokenUsecase(
			&config.Config{Domain: "example.com"},
			oauthApplicationRepo,
			oauthGrantRepo,
			repository.NewOAuthAuthorizationCodeRepository(q),
			oauthAccessTokenRepo,
			oauthRefreshTokenRepo,
			repository.NewSpaceRepository(q),
			repository.NewSpaceMemberRepository(q),
			repository.NewUserRepository(q),
			repository.NewFeatureFlagRepository(q),
		),
		usecase.NewRevokeOAuthTokenUsecase(oauthApplicationRepo, oauthGrantRepo, oauthAccessTokenRepo, oauthRefreshTokenRepo),
	)
}

func newTokenRequest(form url.Values) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

// setBasicAuthは、RFC 6749 §2.3.1のとおりURLエンコードしてからBasic認証の値にする
func setBasicAuth(req *http.Request, clientID, clientSecret string) {
	req.SetBasicAuth(url.QueryEscape(clientID), url.QueryEscape(clientSecret))
}

// assertJSONResponseは、トークン要求の応答の状態・ヘッダーを確かめて本文を読む
func assertJSONResponse(t *testing.T, rr *httptest.ResponseRecorder, wantStatus int) map[string]any {
	t.Helper()

	if rr.Code != wantStatus {
		t.Fatalf("ステータス = %d、期待値 = %d (本文: %s)", rr.Code, wantStatus, rr.Body.String())
	}
	if got := rr.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q、期待値 = %q", got, "application/json")
	}
	if got := rr.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q、期待値 = %q", got, "no-store")
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("本文をJSONとして読めない: %v (本文: %s)", err, rr.Body.String())
	}
	return body
}

func TestCreate_認可コードを交換してトークンレスポンスを返す(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupTokenFixture(t, tx, "oauth-token-create", false)
	form := f.codeForm()
	form.Set("client_id", f.clientID)

	rr := httptest.NewRecorder()
	setupHandler(t, testutil.QueriesWithTx(tx)).Create(rr, newTokenRequest(form))

	body := assertJSONResponse(t, rr, http.StatusOK)
	if token, _ := body["access_token"].(string); !strings.HasPrefix(token, string(auth.OAuthAccessTokenPrefix)) {
		t.Errorf("access_token = %v、接頭辞 %q を期待", body["access_token"], auth.OAuthAccessTokenPrefix)
	}
	if token, _ := body["refresh_token"].(string); !strings.HasPrefix(token, string(auth.OAuthRefreshTokenPrefix)) {
		t.Errorf("refresh_token = %v、接頭辞 %q を期待", body["refresh_token"], auth.OAuthRefreshTokenPrefix)
	}
	if body["token_type"] != "Bearer" {
		t.Errorf("token_type = %v、期待値 = Bearer", body["token_type"])
	}
	if body["expires_in"] != float64(time.Hour/time.Second) {
		t.Errorf("expires_in = %v、期待値 = 3600", body["expires_in"])
	}
	if body["scope"] != "topic:read page:read" {
		t.Errorf("scope = %v、期待値 = %q", body["scope"], "topic:read page:read")
	}
}

func TestCreate_クライアントの認証方式(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// prepareは、認可コードを交換する本文 (資格情報なし) にクライアントの資格情報を加える
		prepare          func(form url.Values, f tokenFixture) *http.Request
		wantStatus       int
		wantError        string
		wantAuthenticate string
	}{
		{
			name: "client_secret_basicで認証できる",
			prepare: func(form url.Values, f tokenFixture) *http.Request {
				req := newTokenRequest(form)
				setBasicAuth(req, f.clientID, testClientSecret)
				return req
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "client_secret_postで認証できる",
			prepare: func(form url.Values, f tokenFixture) *http.Request {
				form.Set("client_id", f.clientID)
				form.Set("client_secret", testClientSecret)
				return newTokenRequest(form)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "Basicと本文で同じclient_idを送るのは受け付ける",
			prepare: func(form url.Values, f tokenFixture) *http.Request {
				form.Set("client_id", f.clientID)
				req := newTokenRequest(form)
				setBasicAuth(req, f.clientID, testClientSecret)
				return req
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "Basicのシークレットが違えば401とWWW-Authenticate",
			prepare: func(form url.Values, f tokenFixture) *http.Request {
				req := newTokenRequest(form)
				setBasicAuth(req, f.clientID, "wks_wrong")
				return req
			},
			wantStatus:       http.StatusUnauthorized,
			wantError:        "invalid_client",
			wantAuthenticate: `Basic realm="wikino"`,
		},
		{
			name: "本文のシークレットが違えても401にWWW-Authenticateを付ける",
			prepare: func(form url.Values, f tokenFixture) *http.Request {
				form.Set("client_id", f.clientID)
				form.Set("client_secret", "wks_wrong")
				return newTokenRequest(form)
			},
			wantStatus:       http.StatusUnauthorized,
			wantError:        "invalid_client",
			wantAuthenticate: `Basic realm="wikino"`,
		},
		{
			name: "Basicと本文のシークレットを同時に使えばinvalid_request",
			prepare: func(form url.Values, f tokenFixture) *http.Request {
				form.Set("client_secret", testClientSecret)
				req := newTokenRequest(form)
				setBasicAuth(req, f.clientID, testClientSecret)
				return req
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid_request",
		},
		{
			name: "Basicと本文でclient_idが違えばinvalid_request",
			prepare: func(form url.Values, f tokenFixture) *http.Request {
				form.Set("client_id", "other")
				req := newTokenRequest(form)
				setBasicAuth(req, f.clientID, testClientSecret)
				return req
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid_request",
		},
		{
			name: "Basic以外の認証スキームはinvalid_client",
			prepare: func(form url.Values, f tokenFixture) *http.Request {
				form.Set("client_id", f.clientID)
				req := newTokenRequest(form)
				req.Header.Set("Authorization", "Bearer "+testClientSecret)
				return req
			},
			wantStatus:       http.StatusUnauthorized,
			wantError:        "invalid_client",
			wantAuthenticate: `Basic realm="wikino"`,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			f := setupTokenFixture(t, tx, "oauth-token-client-"+string(rune('a'+i)), true)

			rr := httptest.NewRecorder()
			setupHandler(t, testutil.QueriesWithTx(tx)).Create(rr, tt.prepare(f.codeForm(), f))

			body := assertJSONResponse(t, rr, tt.wantStatus)
			if tt.wantError != "" && body["error"] != tt.wantError {
				t.Errorf("error = %v、期待値 = %s", body["error"], tt.wantError)
			}
			if got := rr.Header().Get("WWW-Authenticate"); got != tt.wantAuthenticate {
				t.Errorf("WWW-Authenticate = %q、期待値 = %q", got, tt.wantAuthenticate)
			}
		})
	}
}

func TestCreate_Publicクライアントが空の資格情報を送ると拒否する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		prepare func(form url.Values, clientID string) *http.Request
	}{
		{
			name: "本文の空のclient_secret",
			prepare: func(form url.Values, clientID string) *http.Request {
				form.Set("client_id", clientID)
				form.Set("client_secret", "")
				return newTokenRequest(form)
			},
		},
		{
			name: "Basic認証の空のパスワード",
			prepare: func(form url.Values, clientID string) *http.Request {
				req := newTokenRequest(form)
				setBasicAuth(req, clientID, "")
				return req
			},
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			f := setupTokenFixture(t, tx, "oauth-token-empty-secret-"+string(rune('a'+i)), false)
			rr := httptest.NewRecorder()
			setupHandler(t, testutil.QueriesWithTx(tx)).Create(rr, tt.prepare(f.codeForm(), f.clientID))

			body := assertJSONResponse(t, rr, http.StatusUnauthorized)
			if body["error"] != "invalid_client" {
				t.Errorf("error = %v、期待値 = invalid_client", body["error"])
			}
			if got := rr.Header().Get("WWW-Authenticate"); got != `Basic realm="wikino"` {
				t.Errorf("WWW-Authenticate = %q、期待値 = %q", got, `Basic realm="wikino"`)
			}
		})
	}
}

func TestCreate_要求の誤りは400とエラーコードで返す(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupTokenFixture(t, tx, "oauth-token-error", false)

	tests := []struct {
		name      string
		form      url.Values
		wantError string
	}{
		{
			name:      "パラメーターの重複",
			form:      url.Values{"grant_type": {"authorization_code"}, "code": {f.code, f.code}, "client_id": {f.clientID}},
			wantError: "invalid_request",
		},
		{
			name:      "対応していないgrant_type",
			form:      url.Values{"grant_type": {"client_credentials"}, "client_id": {f.clientID}},
			wantError: "unsupported_grant_type",
		},
		{
			name:      "登録されていない認可コード",
			form:      url.Values{"grant_type": {"authorization_code"}, "code": {"unknown"}, "redirect_uri": {testRedirectURI}, "code_verifier": {testCodeVerifier}, "client_id": {f.clientID}},
			wantError: "invalid_grant",
		},
	}

	for _, tt := range tests {
		rr := httptest.NewRecorder()
		setupHandler(t, testutil.QueriesWithTx(tx)).Create(rr, newTokenRequest(tt.form))

		body := assertJSONResponse(t, rr, http.StatusBadRequest)
		if body["error"] != tt.wantError {
			t.Errorf("%s: error = %v、期待値 = %s", tt.name, body["error"], tt.wantError)
		}
		if _, ok := body["error_description"]; ok {
			t.Errorf("%s: error_descriptionが付いている", tt.name)
		}
	}

	t.Run("クエリのパラメーターは読まない", func(t *testing.T) {
		form := f.codeForm()
		form.Set("client_id", f.clientID)
		req := httptest.NewRequest(http.MethodPost, "/oauth/token?"+form.Encode(), nil)

		rr := httptest.NewRecorder()
		setupHandler(t, testutil.QueriesWithTx(tx)).Create(rr, req)

		body := assertJSONResponse(t, rr, http.StatusBadRequest)
		if body["error"] != "invalid_request" {
			t.Errorf("error = %v、期待値 = invalid_request", body["error"])
		}
	})
}
