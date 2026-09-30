package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/oauth2"

	"github.com/wikinoapp/wikino/go/api"
	"github.com/wikinoapp/wikino/go/internal/apierror"
	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/apihandler"
	apicurrentspacemember "github.com/wikinoapp/wikino/go/internal/apihandler/current_space_member"
	"github.com/wikinoapp/wikino/go/internal/apihandler/openapi_description"
	apipage "github.com/wikinoapp/wikino/go/internal/apihandler/page"
	apispace "github.com/wikinoapp/wikino/go/internal/apihandler/space"
	apitopic "github.com/wikinoapp/wikino/go/internal/apihandler/topic"
	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/api_catalog"
	"github.com/wikinoapp/wikino/go/internal/handler/api_reference"
	"github.com/wikinoapp/wikino/go/internal/handler/oauth_authorization_server_metadata"
	"github.com/wikinoapp/wikino/go/internal/handler/oauth_protected_resource_metadata"
	oauthtokenhandler "github.com/wikinoapp/wikino/go/internal/handler/oauth_token"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// oauthFlowTestRedirectURIは、ネイティブアプリがループバックで待ち受けるリダイレクトURI
const oauthFlowTestRedirectURI = "http://127.0.0.1:53123/callback"

// oauthFlowCurrentSpaceMemberPathは、トークンが使えるかを確かめるために呼ぶAPIのパス。
// スペースは、setupOAuthFlowFixtureで作るトークンの束縛先
const oauthFlowCurrentSpaceMemberPath = "/api/v1/spaces/oauth-flow/members/me"

// newOAuthFlowTestServerは、serve.goと同じくトークンエンドポイント・メタデータ・公開APIを配線したサーバーを返す。
// メタデータは発見の流れで辿るGETだけを配線する (HEADのルートは配線しない)。
// リポジトリはテストのトランザクションを使い、トークンの照合とレート制限も実物を使う
func newOAuthFlowTestServer(t *testing.T, cfg *config.Config, q *query.Queries) *httptest.Server {
	t.Helper()

	problems := apierror.NewWriter(cfg.AppURL())
	spec, err := apigen.GetSpec()
	if err != nil {
		t.Fatalf("GetSpec() error = %v", err)
	}
	validator, err := middleware.NewAPIRequestValidator(spec, problems)
	if err != nil {
		t.Fatalf("NewAPIRequestValidator() error = %v", err)
	}

	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	userRepo := repository.NewUserRepository(q)
	featureFlagRepo := repository.NewFeatureFlagRepository(q)
	oauthGrantRepo := repository.NewOAuthGrantRepository(q)
	oauthAccessTokenRepo := repository.NewOAuthAccessTokenRepository(q)
	limiter := ratelimit.NewLimiter(repository.NewRateLimitRepository(q))

	tokenAuth := middleware.NewAPITokenAuth(usecase.NewAuthenticateAPITokenUsecase(
		repository.NewPersonalAccessTokenRepository(q),
		oauthAccessTokenRepo,
		oauthGrantRepo,
		spaceRepo,
		spaceMemberRepo,
		userRepo,
		featureFlagRepo,
	), problems)
	server := apihandler.NewServer(
		apicurrentspacemember.NewHandler(usecase.NewGetAPICurrentSpaceMemberUsecase()),
		openapi_description.NewHandler(api.OpenAPIDescription),
		apipage.NewHandler(nil, nil, nil, nil),
		apispace.NewHandler(usecase.NewGetAPISpaceUsecase()),
		apitopic.NewHandler(nil, nil),
	)
	oauthApplicationRepo := repository.NewOAuthApplicationRepository(q)
	oauthRefreshTokenRepo := repository.NewOAuthRefreshTokenRepository(q)
	tokenHandler := oauthtokenhandler.NewHandler(
		usecase.NewCreateOAuthTokenUsecase(
			cfg,
			oauthApplicationRepo,
			oauthGrantRepo,
			repository.NewOAuthAuthorizationCodeRepository(q),
			oauthAccessTokenRepo,
			oauthRefreshTokenRepo,
			spaceRepo,
			spaceMemberRepo,
			userRepo,
			featureFlagRepo,
		),
		usecase.NewRevokeOAuthTokenUsecase(oauthApplicationRepo, oauthGrantRepo, oauthAccessTokenRepo, oauthRefreshTokenRepo),
	)

	r := chi.NewRouter()
	r.Use(middleware.APIResponseHeader)
	r.Mount(apiMountPath, newAPIRouter(server, api_reference.NewHandler(cfg).Show, problems, tokenAuth,
		middleware.NewAPIRateLimit(limiter, problems, middleware.APISpaceMemberRateLimitPolicy), validator))
	oauthRateLimit := middleware.NewAPIRateLimit(limiter, problems, middleware.OAuthIPRateLimitPolicy(cfg.TrustedProxyCIDRs))
	r.With(oauthRateLimit.Middleware).Post(model.OAuthTokenEndpointPath, tokenHandler.Create)
	r.With(oauthRateLimit.Middleware).Post(model.OAuthRevocationEndpointPath, tokenHandler.Delete)
	r.Get(model.OAuthAuthorizationServerMetadataPath, oauth_authorization_server_metadata.NewHandler(cfg).Show)
	r.Get(spaceAPIResourceMetadataRoute, oauth_protected_resource_metadata.NewHandler(cfg, usecase.NewGetOAuthProtectedResourceMetadataUsecase(spaceRepo)).Show)
	r.Get(model.APICatalogPath, api_catalog.NewHandler(cfg).Show)

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

// oauthFlowFixtureは、公式CLIのようなpublicクライアントと、それを許可するスペースのメンバー
type oauthFlowFixture struct {
	userID        model.UserID
	spaceID       model.SpaceID
	spaceMemberID model.SpaceMemberID
	clientID      string
}

func setupOAuthFlowFixture(t *testing.T, tx *sql.Tx) oauthFlowFixture {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).
		WithAtname("oauth_flow").WithEmail("oauth_flow@example.com").Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("oauth-flow").Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithRole(model.SpaceRoleEditor).
		Build()
	testutil.NewFeatureFlagBuilder(t, tx).WithUserID(userID).WithName(string(model.FeatureFlagPublicAPI)).Build()

	clientID := "oauth-flow-cli"
	testutil.NewOAuthApplicationBuilder(t, tx).
		AsOfficialClient().
		WithName("Wikino CLI").
		WithClientID(clientID).
		WithRedirectURIs([]string{"http://127.0.0.1/callback"}).
		Build()

	return oauthFlowFixture{userID: userID, spaceID: spaceID, spaceMemberID: spaceMemberID, clientID: clientID}
}

// authorizeは、同意画面で許可したときと同じく認可コードを発行する。同意画面のHTTPの振る舞いは
// 認可エンドポイントのハンドラーのテストで確かめるため、ここではユースケースを直接呼ぶ
func authorize(t *testing.T, cfg *config.Config, q *query.Queries, f oauthFlowFixture, verifier string) string {
	t.Helper()

	output, err := usecase.NewCreateOAuthAuthorizationUsecase(
		cfg,
		repository.NewFeatureFlagRepository(q),
		repository.NewOAuthApplicationRepository(q),
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewOAuthAuthorizationCodeRepository(q),
	).Execute(context.Background(), usecase.CreateOAuthAuthorizationInput{
		UserID: f.userID,
		Params: usecase.OAuthAuthorizationParams{
			ClientID:            f.clientID,
			RedirectURI:         oauthFlowTestRedirectURI,
			ResponseType:        "code",
			Scope:               "page:read topic:read",
			State:               "xyz",
			CodeChallenge:       oauth2.S256ChallengeFromVerifier(verifier),
			CodeChallengeMethod: "S256",
			Resource:            cfg.AppURL() + "/api/v1/spaces/oauth-flow",
		},
		Approved: true,
	})
	if err != nil {
		t.Fatalf("認可コードの発行に失敗: %v", err)
	}
	return output.Code
}

// getCurrentSpaceMemberStatusは、トークンで `GET /api/v1/spaces/oauth-flow/members/me` を呼んだときのステータスを返す
func getCurrentSpaceMemberStatus(t *testing.T, srv *httptest.Server, accessToken string) int {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, srv.URL+oauthFlowCurrentSpaceMemberPath, nil)
	if err != nil {
		t.Fatalf("リクエストの作成に失敗: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("APIの呼び出しに失敗: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode
}

// assertRetrieveErrorは、トークン要求がerrorCodeで拒否されたことを確かめる
func assertRetrieveError(t *testing.T, err error, errorCode string) {
	t.Helper()

	var re *oauth2.RetrieveError
	if !errors.As(err, &re) {
		t.Fatalf("エラー = %v、期待値 = %s のRetrieveError", err, errorCode)
	}
	if re.ErrorCode != errorCode {
		t.Errorf("ErrorCode = %q、期待値 = %q", re.ErrorCode, errorCode)
	}
}

// TestOAuthFlowは、golang.org/x/oauth2をクライアントにして、認可 → 交換 → APIの呼び出し → 更新の
// 一連の流れと、リフレッシュトークンの再利用・期限切れ・権限の喪失でトークンが使えなくなることを確かめる
func TestOAuthFlow(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	f := setupOAuthFlowFixture(t, tx)
	ctx := context.Background()

	// 認可サーバーのURL (`iss`・`resource`) とテストサーバーのURLは一致しなくてよい
	cfg := &config.Config{Domain: "example.com"}
	srv := newOAuthFlowTestServer(t, cfg, q)
	ctx = context.WithValue(ctx, oauth2.HTTPClient, srv.Client())
	conf := &oauth2.Config{
		ClientID:    f.clientID,
		RedirectURL: oauthFlowTestRedirectURI,
		Scopes:      []string{"page:read", "topic:read"},
		Endpoint: oauth2.Endpoint{
			TokenURL:  srv.URL + "/oauth/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}

	// 認可 → 交換
	verifier := oauth2.GenerateVerifier()
	code := authorize(t, cfg, q, f, verifier)
	token, err := conf.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		t.Fatalf("認可コードの交換に失敗: %v", err)
	}
	if !strings.HasPrefix(token.AccessToken, string(auth.OAuthAccessTokenPrefix)) || !strings.HasPrefix(token.RefreshToken, string(auth.OAuthRefreshTokenPrefix)) {
		t.Fatalf("トークン = (%q, %q)、接頭辞 wka_・wkr_ を期待", token.AccessToken, token.RefreshToken)
	}
	if token.TokenType != "Bearer" {
		t.Errorf("TokenType = %q、期待値 = Bearer", token.TokenType)
	}
	if remaining := time.Until(token.Expiry); remaining <= 59*time.Minute || remaining > time.Hour {
		t.Errorf("有効期限までの時間 = %v、期待値 = 約1時間", remaining)
	}
	if scope := token.Extra("scope"); scope != "topic:read page:read" {
		t.Errorf("scope = %v、期待値 = %q", scope, "topic:read page:read")
	}

	// 同じコードは2回交換できない
	_, err = conf.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	assertRetrieveError(t, err, "invalid_grant")

	// コードの再提示で1回目に発行したトークンは失効するため、改めて認可する
	verifier = oauth2.GenerateVerifier()
	token, err = conf.Exchange(ctx, authorize(t, cfg, q, f, verifier), oauth2.VerifierOption(verifier))
	if err != nil {
		t.Fatalf("2回目の認可コードの交換に失敗: %v", err)
	}

	// APIの呼び出し
	resp, err := conf.Client(ctx, token).Get(srv.URL + oauthFlowCurrentSpaceMemberPath)
	if err != nil {
		t.Fatalf("APIの呼び出しに失敗: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %sのステータス = %d、期待値 = 200", oauthFlowCurrentSpaceMemberPath, resp.StatusCode)
	}

	// 更新 (アクセストークンの期限が切れたものとしてTokenSourceに更新させる)
	expired := &oauth2.Token{AccessToken: token.AccessToken, RefreshToken: token.RefreshToken, Expiry: time.Now().Add(-time.Minute)}
	refreshed, err := conf.TokenSource(ctx, expired).Token()
	if err != nil {
		t.Fatalf("リフレッシュトークンでの更新に失敗: %v", err)
	}
	if refreshed.AccessToken == token.AccessToken || refreshed.RefreshToken == token.RefreshToken {
		t.Error("更新で新しいアクセストークン・リフレッシュトークンが発行されていない")
	}
	if status := getCurrentSpaceMemberStatus(t, srv, refreshed.AccessToken); status != http.StatusOK {
		t.Fatalf("更新したトークンでのステータス = %d、期待値 = 200", status)
	}

	t.Run("アクセストークンの期限が切れると401", func(t *testing.T) {
		if _, err := tx.ExecContext(ctx, "UPDATE oauth_access_tokens SET expires_at = NOW() - interval '1 second' WHERE token_digest = $1",
			auth.DigestOpaqueToken(token.AccessToken)); err != nil {
			t.Fatalf("有効期限の更新に失敗: %v", err)
		}
		if status := getCurrentSpaceMemberStatus(t, srv, token.AccessToken); status != http.StatusUnauthorized {
			t.Errorf("期限切れのトークンでのステータス = %d、期待値 = 401", status)
		}
	})

	t.Run("使用済みのリフレッシュトークンの再提示で、許可のトークンがすべて失効する", func(t *testing.T) {
		replayed := &oauth2.Token{RefreshToken: token.RefreshToken, Expiry: time.Now().Add(-time.Minute)}
		_, err := conf.TokenSource(ctx, replayed).Token()
		assertRetrieveError(t, err, "invalid_grant")

		if status := getCurrentSpaceMemberStatus(t, srv, refreshed.AccessToken); status != http.StatusUnauthorized {
			t.Errorf("再提示の後の、更新したアクセストークンでのステータス = %d、期待値 = 401", status)
		}
		_, err = conf.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshed.RefreshToken, Expiry: time.Now().Add(-time.Minute)}).Token()
		assertRetrieveError(t, err, "invalid_grant")
	})
}

// CLIのログアウトのように、クライアントがリフレッシュトークンを失効させると、同じ許可の
// アクセストークンでAPIを呼べなくなり、リフレッシュトークンでも更新できなくなる
func TestOAuthFlow_トークンの失効(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	f := setupOAuthFlowFixture(t, tx)
	ctx := context.Background()

	cfg := &config.Config{Domain: "example.com"}
	srv := newOAuthFlowTestServer(t, cfg, q)
	ctx = context.WithValue(ctx, oauth2.HTTPClient, srv.Client())
	conf := &oauth2.Config{
		ClientID:    f.clientID,
		RedirectURL: oauthFlowTestRedirectURI,
		Scopes:      []string{"page:read"},
		Endpoint: oauth2.Endpoint{
			TokenURL:  srv.URL + "/oauth/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}

	verifier := oauth2.GenerateVerifier()
	token, err := conf.Exchange(ctx, authorize(t, cfg, q, f, verifier), oauth2.VerifierOption(verifier))
	if err != nil {
		t.Fatalf("認可コードの交換に失敗: %v", err)
	}
	if status := getCurrentSpaceMemberStatus(t, srv, token.AccessToken); status != http.StatusOK {
		t.Fatalf("失効前のステータス = %d、期待値 = 200", status)
	}

	resp, err := srv.Client().PostForm(srv.URL+"/oauth/revoke", url.Values{
		"token":           {token.RefreshToken},
		"token_type_hint": {"refresh_token"},
		"client_id":       {f.clientID},
	})
	if err != nil {
		t.Fatalf("POST /oauth/revokeに失敗: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /oauth/revokeのステータス = %d、期待値 = 200", resp.StatusCode)
	}

	if status := getCurrentSpaceMemberStatus(t, srv, token.AccessToken); status != http.StatusUnauthorized {
		t.Errorf("失効後のアクセストークンでのステータス = %d、期待値 = 401", status)
	}
	_, err = conf.TokenSource(ctx, &oauth2.Token{RefreshToken: token.RefreshToken, Expiry: time.Now().Add(-time.Minute)}).Token()
	assertRetrieveError(t, err, "invalid_grant")
}

// newPersonalAccessTokenは、spaceMemberIDのメンバーの個人アクセストークンを作り、平文を返す
func newPersonalAccessToken(t *testing.T, tx *sql.Tx, spaceID model.SpaceID, spaceMemberID model.SpaceMemberID) string {
	t.Helper()

	token, err := auth.GenerateOpaqueToken(auth.PersonalAccessTokenPrefix)
	if err != nil {
		t.Fatalf("個人アクセストークンの生成に失敗: %v", err)
	}
	testutil.NewPersonalAccessTokenBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		WithTokenDigest(auth.DigestOpaqueToken(token)).
		Build()
	return token
}

// getRateLimitは、トークンでpathのAPIを呼び、`RateLimit` ヘッダーの値を返す
func getRateLimit(t *testing.T, srv *httptest.Server, path, accessToken string) string {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, srv.URL+path, nil)
	if err != nil {
		t.Fatalf("リクエストの作成に失敗: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("APIの呼び出しに失敗: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %sのステータス = %d、期待値 = 200", path, resp.StatusCode)
	}
	return resp.Header.Get("RateLimit")
}

// TestAPIRateLimit_スペースのメンバー単位は、公開APIのレート制限が、同じスペースの個人アクセストークンと
// OAuthのアクセストークンを合わせて数え、同じ人でもスペースが違えば別に数えることを、実際のカウンターで確かめる
func TestAPIRateLimit_スペースのメンバー単位(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	f := setupOAuthFlowFixture(t, tx)
	ctx := context.Background()

	cfg := &config.Config{Domain: "example.com"}
	srv := newOAuthFlowTestServer(t, cfg, q)
	ctx = context.WithValue(ctx, oauth2.HTTPClient, srv.Client())
	conf := &oauth2.Config{
		ClientID:    f.clientID,
		RedirectURL: oauthFlowTestRedirectURI,
		Scopes:      []string{"page:read"},
		Endpoint: oauth2.Endpoint{
			TokenURL:  srv.URL + "/oauth/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}
	verifier := oauth2.GenerateVerifier()
	oauthToken, err := conf.Exchange(ctx, authorize(t, cfg, q, f, verifier), oauth2.VerifierOption(verifier))
	if err != nil {
		t.Fatalf("認可コードの交換に失敗: %v", err)
	}

	// 編集者はトークンの発行の権限を持つため、同じメンバーで個人アクセストークンも使える
	pat := newPersonalAccessToken(t, tx, f.spaceID, f.spaceMemberID)

	// 同じ人が参加している、別のスペース
	otherSpaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("oauth-flow-other").Build()
	otherSpaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(otherSpaceID).
		WithUserID(f.userID).
		WithRole(model.SpaceRoleEditor).
		Build()
	otherPAT := newPersonalAccessToken(t, tx, otherSpaceID, otherSpaceMemberID)
	const otherSpacePath = "/api/v1/spaces/oauth-flow-other/members/me"

	// `t` は時間枠が切り替わるまでの秒数で呼び出しの時刻によって変わるため、残量 `r` までを比べる
	tests := []struct {
		name  string
		path  string
		token string
		want  string
	}{
		{name: "個人アクセストークンでの1回目", path: oauthFlowCurrentSpaceMemberPath, token: pat, want: `"space_member";r=4999;`},
		{name: "同じスペースのOAuthのトークンは個人アクセストークンと合わせて数える", path: oauthFlowCurrentSpaceMemberPath, token: oauthToken.AccessToken, want: `"space_member";r=4998;`},
		{name: "同じ人でもスペースが違えば別に数える", path: otherSpacePath, token: otherPAT, want: `"space_member";r=4999;`},
	}
	for _, tt := range tests {
		if got := getRateLimit(t, srv, tt.path, tt.token); !strings.HasPrefix(got, tt.want) {
			t.Errorf("%s: RateLimit = %q、期待値は %q で始まる値", tt.name, got, tt.want)
		}
	}
}

// getDiscoveryJSONは、メタデータが指すURL (認可サーバーのオリジン) のパスをテストサーバーで取得し、
// 本文をJSONとしてvに読み込む。メタデータのURLのオリジンは設定のものと一致しなければならない
func getDiscoveryJSON(t *testing.T, cfg *config.Config, srv *httptest.Server, rawURL string, v any) {
	t.Helper()

	path := testServerPath(t, cfg, rawURL)
	resp, err := srv.Client().Get(srv.URL + path)
	if err != nil {
		t.Fatalf("GET %sに失敗: %v", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %sのステータス = %d、期待値 = 200", path, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("GET %sの本文をJSONとして読めない: %v", path, err)
	}
}

// testServerPathは、メタデータが指す絶対URLのオリジンが設定のものであることを確かめ、パスを返す
func testServerPath(t *testing.T, cfg *config.Config, rawURL string) string {
	t.Helper()

	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("URL %qを解析できない: %v", rawURL, err)
	}
	if origin := u.Scheme + "://" + u.Host; origin != cfg.AppURL() {
		t.Fatalf("URL %qのオリジン = %q、期待値 = %q", rawURL, origin, cfg.AppURL())
	}
	return u.EscapedPath()
}

// TestOAuthFlow_メタデータからの発見は、クライアントが手書きの設定なしに、APIの401から保護リソースの
// メタデータ (RFC 9728)・認可サーバーのメタデータ (RFC 8414) を辿り、そこに書かれた値だけで
// 認可 → 交換 → APIの呼び出し → 失効を行えることを確かめる。メタデータの値と実際の挙動のずれを検出する
func TestOAuthFlow_メタデータからの発見(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	f := setupOAuthFlowFixture(t, tx)
	ctx := context.Background()

	cfg := &config.Config{Domain: "example.com"}
	srv := newOAuthFlowTestServer(t, cfg, q)

	// 1. トークンなしでスペースのAPIを呼び、401の `resource_metadata` を得る
	resp, err := srv.Client().Get(srv.URL + "/api/v1/spaces/oauth-flow")
	if err != nil {
		t.Fatalf("APIの呼び出しに失敗: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("トークンなしのステータス = %d、期待値 = 401", resp.StatusCode)
	}
	challenge := resp.Header.Get("WWW-Authenticate")
	_, resourceMetadataURL, ok := strings.Cut(challenge, `resource_metadata="`)
	if !ok {
		t.Fatalf("WWW-Authenticate = %q、resource_metadataを期待", challenge)
	}
	resourceMetadataURL = strings.TrimSuffix(resourceMetadataURL, `"`)

	// 2. 保護リソースのメタデータから、リソースの識別子と認可サーバーを得る
	var resource struct {
		Resource               string   `json:"resource"`
		AuthorizationServers   []string `json:"authorization_servers"`
		ScopesSupported        []string `json:"scopes_supported"`
		BearerMethodsSupported []string `json:"bearer_methods_supported"`
	}
	getDiscoveryJSON(t, cfg, srv, resourceMetadataURL, &resource)
	if want := cfg.AppURL() + "/api/v1/spaces/oauth-flow"; resource.Resource != want {
		t.Fatalf("resource = %q、期待値 = %q", resource.Resource, want)
	}
	if len(resource.AuthorizationServers) != 1 {
		t.Fatalf("authorization_servers = %v、1つを期待", resource.AuthorizationServers)
	}
	if !slices.Equal(resource.BearerMethodsSupported, []string{"header"}) {
		t.Errorf("bearer_methods_supported = %v、期待値 = [header]", resource.BearerMethodsSupported)
	}

	// 3. 認可サーバーのメタデータを、issuerにwell-knownの名前を続けたURLから得る (RFC 8414 §3)
	var server struct {
		Issuer                            string   `json:"issuer"`
		AuthorizationEndpoint             string   `json:"authorization_endpoint"`
		TokenEndpoint                     string   `json:"token_endpoint"`
		RevocationEndpoint                string   `json:"revocation_endpoint"`
		ScopesSupported                   []string `json:"scopes_supported"`
		ResponseTypesSupported            []string `json:"response_types_supported"`
		GrantTypesSupported               []string `json:"grant_types_supported"`
		TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
		CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
	}
	issuer := resource.AuthorizationServers[0]
	getDiscoveryJSON(t, cfg, srv, issuer+"/.well-known/oauth-authorization-server", &server)
	if server.Issuer != issuer {
		t.Fatalf("issuer = %q、期待値 = %q", server.Issuer, issuer)
	}
	if !slices.Equal(server.ScopesSupported, resource.ScopesSupported) {
		t.Errorf("認可サーバーのscopes_supported = %v、保護リソースの値 = %v", server.ScopesSupported, resource.ScopesSupported)
	}
	if !slices.Contains(server.TokenEndpointAuthMethodsSupported, "none") || !slices.Contains(server.GrantTypesSupported, "refresh_token") {
		t.Fatalf("メタデータ = %+v、publicクライアントの認証とリフレッシュトークンを期待", server)
	}
	if path := testServerPath(t, cfg, server.AuthorizationEndpoint); path != model.OAuthAuthorizationEndpointPath {
		t.Errorf("authorization_endpointのパス = %q、期待値 = %q", path, model.OAuthAuthorizationEndpointPath)
	}

	// 4. メタデータの値で認可する。scopes_supportedのスコープはすべて要求でき、発行の `iss` はissuerと一致する
	verifier := oauth2.GenerateVerifier()
	authorization, err := usecase.NewCreateOAuthAuthorizationUsecase(
		cfg,
		repository.NewFeatureFlagRepository(q),
		repository.NewOAuthApplicationRepository(q),
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewOAuthAuthorizationCodeRepository(q),
	).Execute(ctx, usecase.CreateOAuthAuthorizationInput{
		UserID: f.userID,
		Params: usecase.OAuthAuthorizationParams{
			ClientID:            f.clientID,
			RedirectURI:         oauthFlowTestRedirectURI,
			ResponseType:        server.ResponseTypesSupported[0],
			Scope:               strings.Join(server.ScopesSupported, " "),
			State:               "xyz",
			CodeChallenge:       oauth2.S256ChallengeFromVerifier(verifier),
			CodeChallengeMethod: server.CodeChallengeMethodsSupported[0],
			Resource:            resource.Resource,
		},
		Approved: true,
	})
	if err != nil {
		t.Fatalf("メタデータの値での認可に失敗: %v", err)
	}

	// 5. token_endpointで交換し、束縛したスペースのAPIを呼ぶ
	ctx = context.WithValue(ctx, oauth2.HTTPClient, srv.Client())
	conf := &oauth2.Config{
		ClientID:    f.clientID,
		RedirectURL: oauthFlowTestRedirectURI,
		Endpoint: oauth2.Endpoint{
			TokenURL:  srv.URL + testServerPath(t, cfg, server.TokenEndpoint),
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}
	token, err := conf.Exchange(ctx, authorization.Code, oauth2.VerifierOption(verifier))
	if err != nil {
		t.Fatalf("token_endpointでの交換に失敗: %v", err)
	}
	if scope := token.Extra("scope"); scope != strings.Join(server.ScopesSupported, " ") {
		t.Errorf("scope = %v、期待値 = %q", scope, strings.Join(server.ScopesSupported, " "))
	}
	spaceStatus := func() int {
		req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/spaces/oauth-flow", nil)
		if err != nil {
			t.Fatalf("リクエストの作成に失敗: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+token.AccessToken)
		resp, err := srv.Client().Do(req)
		if err != nil {
			t.Fatalf("APIの呼び出しに失敗: %v", err)
		}
		_ = resp.Body.Close()
		return resp.StatusCode
	}
	if status := spaceStatus(); status != http.StatusOK {
		t.Fatalf("交換したトークンでのステータス = %d、期待値 = 200", status)
	}

	// 6. revocation_endpointで失効させると、APIを呼べなくなる
	resp, err = srv.Client().PostForm(srv.URL+testServerPath(t, cfg, server.RevocationEndpoint), url.Values{
		"token":     {token.AccessToken},
		"client_id": {f.clientID},
	})
	if err != nil {
		t.Fatalf("revocation_endpointの呼び出しに失敗: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("revocation_endpointのステータス = %d、期待値 = 200", resp.StatusCode)
	}
	if status := spaceStatus(); status != http.StatusUnauthorized {
		t.Errorf("失効後のステータス = %d、期待値 = 401", status)
	}

	// 7. APIカタログのservice-descからOpenAPI記述を、service-docからAPIリファレンスを取得できる
	var catalog struct {
		Linkset []struct {
			Anchor      string `json:"anchor"`
			ServiceDesc []struct {
				Href string `json:"href"`
			} `json:"service-desc"`
			ServiceDoc []struct {
				Href string `json:"href"`
			} `json:"service-doc"`
		} `json:"linkset"`
	}
	getDiscoveryJSON(t, cfg, srv, cfg.AppURL()+"/.well-known/api-catalog", &catalog)
	if len(catalog.Linkset) != 1 || len(catalog.Linkset[0].ServiceDesc) != 1 || len(catalog.Linkset[0].ServiceDoc) != 1 {
		t.Fatalf("APIカタログ = %+v、1つのAPIと記述・ドキュメントを期待", catalog)
	}
	for rel, href := range map[string]string{
		"service-desc": catalog.Linkset[0].ServiceDesc[0].Href,
		"service-doc":  catalog.Linkset[0].ServiceDoc[0].Href,
	} {
		resp, err = srv.Client().Get(srv.URL + testServerPath(t, cfg, href))
		if err != nil {
			t.Fatalf("%sの取得に失敗: %v", rel, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%sのステータス = %d、期待値 = 200", rel, resp.StatusCode)
		}
	}
}
