package oauth_device_authorization_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/oauth_device_authorization"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// testConfigはテストで使う設定。AppURLは `https://example.com` になる
var testConfig = &config.Config{Domain: "example.com"}

// testClientSecretはconfidentialクライアントのシークレット
const testClientSecret = "wks_device_secret"

// deviceFixtureは、デバイス認可を要求するスペースのpublicクライアントとconfidentialクライアント
type deviceFixture struct {
	identifier         model.SpaceIdentifier
	clientID           string
	confidentialClient string
}

// setupDeviceFixtureは、スペースkeyと、そのスペースのpublicクライアント・confidentialクライアントを作る
func setupDeviceFixture(t *testing.T, tx *sql.Tx, key string) deviceFixture {
	t.Helper()

	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier(key).Build()
	testutil.NewOAuthApplicationBuilder(t, tx).WithSpaceID(spaceID).WithClientID(key + "-client").Build()
	testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(spaceID).
		WithClientID(key + "-confidential").
		WithConfidentialClientSecretDigest(auth.DigestOpaqueToken(testClientSecret)).
		Build()

	return deviceFixture{
		identifier:         model.SpaceIdentifier(key),
		clientID:           key + "-client",
		confidentialClient: key + "-confidential",
	}
}

func setupHandler(t *testing.T, tx *sql.Tx) *oauth_device_authorization.Handler {
	t.Helper()

	q := testutil.QueriesWithTx(tx)
	return oauth_device_authorization.NewHandler(usecase.NewCreateOAuthDeviceAuthorizationUsecase(
		testConfig,
		repository.NewOAuthApplicationRepository(q),
		repository.NewSpaceRepository(q),
		repository.NewOAuthDeviceAuthorizationRepository(q),
	))
}

func newDeviceAuthorizationRequest(target string, form url.Values) *http.Request {
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

// assertJSONResponseは、応答の状態・ヘッダーを確かめて本文を読む
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

func TestCreate_デバイスコードとユーザーコードを返す(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupDeviceFixture(t, tx, "oauth-device-create")

	rr := httptest.NewRecorder()
	setupHandler(t, tx).Create(rr, newDeviceAuthorizationRequest("/oauth/device_authorization", url.Values{
		"client_id": {f.clientID},
		"scope":     {"page:read"},
		"resource":  {model.SpaceAPIResourceURL(testConfig.AppURL(), f.identifier)},
	}))

	body := assertJSONResponse(t, rr, http.StatusOK)
	deviceCode, _ := body["device_code"].(string)
	if deviceCode == "" {
		t.Errorf("device_code = %v、空でない文字列を期待", body["device_code"])
	}
	userCode, _ := body["user_code"].(string)
	if !regexp.MustCompile(`^[BCDFGHJKLMNPQRSTVWXZ]{4}-[BCDFGHJKLMNPQRSTVWXZ]{4}$`).MatchString(userCode) {
		t.Errorf("user_code = %v、期待値は `BCDF-GHJK` の形", body["user_code"])
	}
	if body["verification_uri"] != "https://example.com/oauth/device" {
		t.Errorf("verification_uri = %v、期待値 = %q", body["verification_uri"], "https://example.com/oauth/device")
	}
	if want := "https://example.com/oauth/device?user_code=" + userCode; body["verification_uri_complete"] != want {
		t.Errorf("verification_uri_complete = %v、期待値 = %q", body["verification_uri_complete"], want)
	}
	if body["expires_in"] != float64(600) {
		t.Errorf("expires_in = %v、期待値 = 600", body["expires_in"])
	}
	if body["interval"] != float64(5) {
		t.Errorf("interval = %v、期待値 = 5", body["interval"])
	}

	stored, err := repository.NewOAuthDeviceAuthorizationRepository(testutil.QueriesWithTx(tx)).
		FindByDeviceCodeDigest(t.Context(), auth.DigestOpaqueToken(deviceCode))
	if err != nil {
		t.Fatalf("要求の取得に失敗: %v", err)
	}
	if stored == nil {
		t.Fatal("返したデバイスコードのダイジェストで要求が見つからない")
	}
}

func TestCreate_空のシークレットは省略と同じに扱う(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupDeviceFixture(t, tx, "oauth-device-empty-secret")

	rr := httptest.NewRecorder()
	setupHandler(t, tx).Create(rr, newDeviceAuthorizationRequest("/oauth/device_authorization", url.Values{
		"client_id":     {f.clientID},
		"client_secret": {""},
		"scope":         {"page:read"},
	}))

	body := assertJSONResponse(t, rr, http.StatusOK)
	if deviceCode, _ := body["device_code"].(string); deviceCode == "" {
		t.Errorf("device_code = %v、空でない文字列を期待", body["device_code"])
	}
	if got := rr.Header().Get("WWW-Authenticate"); got != "" {
		t.Errorf("WWW-Authenticate = %q、期待値は無し", got)
	}
}

func TestCreate_受け付けない要求(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupDeviceFixture(t, tx, "oauth-device-reject")
	handler := setupHandler(t, tx)

	tests := []struct {
		name             string
		request          func() *http.Request
		wantStatus       int
		wantError        string
		wantAuthenticate bool
	}{
		{
			name: "登録の無いclient_id",
			request: func() *http.Request {
				return newDeviceAuthorizationRequest("/oauth/device_authorization", url.Values{"client_id": {"oauth-device-unknown"}, "scope": {"page:read"}})
			},
			wantStatus:       http.StatusUnauthorized,
			wantError:        "invalid_client",
			wantAuthenticate: true,
		},
		{
			name: "クエリだけで送ったclient_idは読まない",
			request: func() *http.Request {
				return newDeviceAuthorizationRequest("/oauth/device_authorization?client_id="+f.clientID, url.Values{"scope": {"page:read"}})
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid_request",
		},
		{
			name: "パラメーターの重複",
			request: func() *http.Request {
				return newDeviceAuthorizationRequest("/oauth/device_authorization", url.Values{"client_id": {f.clientID}, "scope": {"page:read", "topic:read"}})
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid_request",
		},
		{
			name: "Basicで認証するconfidentialクライアント",
			request: func() *http.Request {
				req := newDeviceAuthorizationRequest("/oauth/device_authorization", url.Values{"scope": {"page:read"}})
				req.SetBasicAuth(url.QueryEscape(f.confidentialClient), url.QueryEscape(testClientSecret))
				return req
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "unauthorized_client",
		},
		{
			name: "シークレットを送ったpublicクライアント",
			request: func() *http.Request {
				return newDeviceAuthorizationRequest("/oauth/device_authorization", url.Values{"client_id": {f.clientID}, "client_secret": {"secret"}, "scope": {"page:read"}})
			},
			wantStatus:       http.StatusUnauthorized,
			wantError:        "invalid_client",
			wantAuthenticate: true,
		},
		{
			name: "Basicで認証するpublicクライアント",
			request: func() *http.Request {
				req := newDeviceAuthorizationRequest("/oauth/device_authorization", url.Values{"scope": {"page:read"}})
				req.SetBasicAuth(url.QueryEscape(f.clientID), "")
				return req
			},
			wantStatus:       http.StatusUnauthorized,
			wantError:        "invalid_client",
			wantAuthenticate: true,
		},
		{
			name: "Basicと本文でclient_idが違う",
			request: func() *http.Request {
				req := newDeviceAuthorizationRequest("/oauth/device_authorization", url.Values{"client_id": {f.clientID}, "scope": {"page:read"}})
				req.SetBasicAuth(url.QueryEscape(f.confidentialClient), url.QueryEscape(testClientSecret))
				return req
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid_request",
		},
		{
			name: "Basic以外の認証スキーム",
			request: func() *http.Request {
				req := newDeviceAuthorizationRequest("/oauth/device_authorization", url.Values{"client_id": {f.clientID}, "scope": {"page:read"}})
				req.Header.Set("Authorization", "Bearer token")
				return req
			},
			wantStatus:       http.StatusUnauthorized,
			wantError:        "invalid_client",
			wantAuthenticate: true,
		},
		{
			name: "不明なスコープ",
			request: func() *http.Request {
				return newDeviceAuthorizationRequest("/oauth/device_authorization", url.Values{"client_id": {f.clientID}, "scope": {"page:read unknown:read"}})
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid_scope",
		},
		{
			name: "形式が不正な `resource`",
			request: func() *http.Request {
				return newDeviceAuthorizationRequest("/oauth/device_authorization", url.Values{"client_id": {f.clientID}, "scope": {"page:read"}, "resource": {"https://example.com/"}})
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid_target",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			handler.Create(rr, tt.request())

			body := assertJSONResponse(t, rr, tt.wantStatus)
			if body["error"] != tt.wantError {
				t.Errorf("error = %v、期待値 = %q", body["error"], tt.wantError)
			}
			if _, ok := body["device_code"]; ok {
				t.Error("エラーの応答にdevice_codeが含まれている")
			}
			gotAuthenticate := rr.Header().Get("WWW-Authenticate")
			if tt.wantAuthenticate && gotAuthenticate != `Basic realm="wikino"` {
				t.Errorf("WWW-Authenticate = %q、期待値 = %q", gotAuthenticate, `Basic realm="wikino"`)
			}
			if !tt.wantAuthenticate && gotAuthenticate != "" {
				t.Errorf("WWW-Authenticate = %q、期待値は無し", gotAuthenticate)
			}
		})
	}
}
