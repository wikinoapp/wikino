package oauth_authorization_test

import (
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	oauthauthorizationhandler "github.com/wikinoapp/wikino/go/internal/handler/oauth_authorization"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// testIssuerは認可レスポンスの `iss` に載る認可サーバーの識別子 (テストの設定のオリジン)
const testIssuer = "https://example.com"

// testCodeChallengeはS256のcode_challengeの例 (RFC 7636 Appendix B)
const testCodeChallenge = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

const redirectURIWithReservedParams = "https://client.example/callback?app=cli&code=old&state=old&iss=old&error=old"

// authorizationFixtureは、連携先のスペースのメンバーのユーザーと、そのスペースのOAuthアプリ
type authorizationFixture struct {
	userID   model.UserID
	spaceID  model.SpaceID
	clientID string
}

// setupAuthorizationFixtureは、identifierを識別子に持つスペースと、scopesを持つメンバーと、
// そのスペースのOAuthアプリ「連携するCLI」を作る。flagEnabledが真ならメンバーのユーザーに
// 公開APIのフィーチャーフラグを有効にする
func setupAuthorizationFixture(t *testing.T, tx *sql.Tx, identifier string, scopes []model.Scope, flagEnabled bool) authorizationFixture {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail(identifier + "@example.com").
		WithAtname(strings.ReplaceAll(identifier, "-", "_")).
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier(identifier).
		WithName("連携先のスペース").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithScopes(scopes).
		Build()
	if flagEnabled {
		testutil.NewFeatureFlagBuilder(t, tx).
			WithUserID(userID).
			WithName(string(model.FeatureFlagPublicAPI)).
			Build()
	}
	clientID := identifier + "-client"
	testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(spaceID).
		WithName("連携するCLI").
		WithClientID(clientID).
		WithRedirectURIs([]string{"https://client.example/callback?app=cli", redirectURIWithReservedParams, "http://127.0.0.1/callback"}).
		Build()

	return authorizationFixture{userID: userID, spaceID: spaceID, clientID: clientID}
}

// validParamsは、clientIDのアプリへの検証を通る認可要求のパラメーターを返す
func validParams(clientID string) url.Values {
	return url.Values{
		"client_id":             {clientID},
		"redirect_uri":          {"https://client.example/callback?app=cli"},
		"response_type":         {"code"},
		"scope":                 {"page:read topic:read"},
		"state":                 {"xyz"},
		"code_challenge":        {testCodeChallenge},
		"code_challenge_method": {"S256"},
	}
}

// setupHandlerは渡したクエリを使うハンドラーを組み立てる
func setupHandler(t *testing.T, q *query.Queries) *oauthauthorizationhandler.Handler {
	t.Helper()

	cfg := &config.Config{Env: "test", Domain: "example.com"}
	featureFlagRepo := repository.NewFeatureFlagRepository(q)
	appRepo := repository.NewOAuthApplicationRepository(q)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)

	return oauthauthorizationhandler.NewHandler(
		cfg,
		usecase.NewGetOAuthAuthorizationNewUsecase(cfg, featureFlagRepo, appRepo, spaceRepo, spaceMemberRepo),
		usecase.NewCreateOAuthAuthorizationUsecase(cfg, featureFlagRepo, appRepo, spaceRepo, spaceMemberRepo, repository.NewOAuthAuthorizationCodeRepository(q)),
	)
}

// newRequestはログイン中のユーザーと画面を描画するロケールを載せたリクエストを組み立てる。
// GETではparamsをクエリに、POSTではフォームの本文にする。userIDが空のときはユーザーを載せず、
// 未ログインの経路へ到達できるようにする
func newRequest(t *testing.T, method string, userID model.UserID, params url.Values) *http.Request {
	t.Helper()

	target := "/oauth/authorize"
	var body io.Reader
	if method == http.MethodPost {
		body = strings.NewReader(params.Encode())
	} else {
		target += "?" + params.Encode()
	}
	req := httptest.NewRequest(method, target, body)
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	ctx := req.Context()
	if userID != "" {
		ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "authorizing_user"})
	}
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	return req.WithContext(ctx)
}

// assertResponseHeadersは、認可エンドポイントのすべての応答に付けるヘッダーを確かめる
func assertResponseHeaders(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()

	want := map[string]string{
		"Content-Security-Policy": "frame-ancestors 'none'",
		"X-Frame-Options":         "DENY",
		"X-Robots-Tag":            "noindex",
		"Cache-Control":           "no-store",
	}
	for name, value := range want {
		if got := rr.Header().Get(name); got != value {
			t.Errorf("%s = %q、期待値 = %q", name, got, value)
		}
	}
}

// assertRedirectToClientは、応答がリダイレクトURIへのstatusのリダイレクトで、クエリが
// 登録したリダイレクトURIのクエリ (`app=cli`) を残したままwantを含み、`iss` を持つことを確かめる。
// 付いたクエリの値を返す
func assertRedirectToClient(t *testing.T, rr *httptest.ResponseRecorder, status int, want map[string]string) url.Values {
	t.Helper()

	if rr.Code != status {
		t.Fatalf("ステータス = %d、期待値 = %d。本文 = %s", rr.Code, status, rr.Body.String())
	}
	location, err := url.Parse(rr.Header().Get("Location"))
	if err != nil {
		t.Fatalf("Locationの解析に失敗: %v", err)
	}
	if location.Scheme != "https" || location.Host != "client.example" || location.Path != "/callback" {
		t.Fatalf("Location = %q、期待値はリダイレクトURI", location)
	}
	query := location.Query()
	if len(query["app"]) != 1 || query.Get("app") != "cli" {
		t.Errorf("Locationのクエリ = %q、期待値はリダイレクトURIのクエリ (app=cli) を残す", location.RawQuery)
	}
	if got := query.Get("iss"); got != testIssuer {
		t.Errorf("iss = %q、期待値 = %q", got, testIssuer)
	}
	for name, value := range want {
		if got := query.Get(name); got != value {
			t.Errorf("%s = %q、期待値 = %q", name, got, value)
		}
	}
	return query
}

func TestRedirectToClient_登録URIの予約パラメーターを認可レスポンスで置き換える(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupAuthorizationFixture(t, tx, "oauthz-response-collision", []model.Scope{model.ScopeOAuthGrantWrite}, true)
	h := setupHandler(t, testutil.QueriesWithTx(tx))
	params := validParams(f.clientID)
	params.Set("redirect_uri", redirectURIWithReservedParams)

	approved := url.Values{}
	for name, values := range params {
		approved[name] = append([]string(nil), values...)
	}
	approved.Set("decision", "approve")
	rr := httptest.NewRecorder()
	h.Create(rr, newRequest(t, http.MethodPost, f.userID, approved))
	query := assertRedirectToClient(t, rr, http.StatusSeeOther, map[string]string{"state": "xyz"})
	if len(query["code"]) != 1 || query.Get("code") == "old" {
		t.Errorf("成功応答のcode = %v、発行したコード1つを期待", query["code"])
	}
	if len(query["state"]) != 1 || len(query["iss"]) != 1 || query.Has("error") {
		t.Errorf("成功応答の予約パラメーター = %v、一意なstateとissのみを期待", query)
	}

	params.Set("response_type", "token")
	rr = httptest.NewRecorder()
	h.New(rr, newRequest(t, http.MethodGet, f.userID, params))
	query = assertRedirectToClient(t, rr, http.StatusFound, map[string]string{
		"error": "unsupported_response_type",
		"state": "xyz",
	})
	if len(query["error"]) != 1 || len(query["state"]) != 1 || len(query["iss"]) != 1 || query.Has("code") {
		t.Errorf("エラー応答の予約パラメーター = %v、一意なerror・state・issのみを期待", query)
	}
}
