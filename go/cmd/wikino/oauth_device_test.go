package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// newOAuthDeviceFlowConfigは、テストサーバーのデバイス認可の要求とトークンのエンドポイントを使う、
// 公式CLIと同じpublicクライアントの設定を返す
func newOAuthDeviceFlowConfig(srv *httptest.Server, f oauthFlowFixture) *oauth2.Config {
	return &oauth2.Config{
		ClientID: f.clientID,
		Scopes:   []string{"page:read", "topic:read"},
		Endpoint: oauth2.Endpoint{
			DeviceAuthURL: srv.URL + model.OAuthDeviceAuthorizationEndpointPath,
			TokenURL:      srv.URL + model.OAuthTokenEndpointPath,
			AuthStyle:     oauth2.AuthStyleInParams,
		},
	}
}

// oauthDeviceFlowResourceは、フィクスチャのスペースのAPIを指す `resource`
func oauthDeviceFlowResource(t *testing.T, cfg *config.Config) oauth2.AuthCodeOption {
	t.Helper()

	return oauth2.SetAuthURLParam("resource", cfg.AppURL()+"/api/v1/spaces/"+oauthFlowSpaceIdentifier(t))
}

// requestDeviceAuthorizationは、公式CLIと同じくデバイス認可を要求する
func requestDeviceAuthorization(t *testing.T, ctx context.Context, cfg *config.Config, conf *oauth2.Config) *oauth2.DeviceAuthResponse {
	t.Helper()

	da, err := conf.DeviceAuth(ctx, oauthDeviceFlowResource(t, cfg))
	if err != nil {
		t.Fatalf("デバイス認可の要求に失敗: %v", err)
	}
	return da
}

// decideDeviceAuthorizationは、利用者がユーザーコードを入力し、同意画面でgrantedのスコープを選んで
// 許可 (approvedが偽なら拒否) したときと同じく要求を更新する。入力と同意画面のHTTPの振る舞いは
// ハンドラーのテストで確かめるため、ここではユースケースを直接呼ぶ
func decideDeviceAuthorization(t *testing.T, q *query.Queries, f oauthFlowFixture, userCode string, approved bool, granted []string) {
	t.Helper()

	deviceRepo := repository.NewOAuthDeviceAuthorizationRepository(q)
	output, err := usecase.NewUpdateOAuthDeviceAuthorizationUsecase(
		testutil.GetTestDB(),
		repository.NewFeatureFlagRepository(q),
		deviceRepo,
		repository.NewOAuthGrantRepository(q),
		repository.NewSpaceMemberRepository(q),
		validator.NewOAuthDeviceVerificationValidator(deviceRepo, repository.NewOAuthApplicationRepository(q), repository.NewSpaceRepository(q)),
		validator.NewOAuthAuthorizationCreateValidator(),
	).Execute(context.Background(), usecase.UpdateOAuthDeviceAuthorizationInput{
		UserID:        f.userID,
		UserCode:      userCode,
		Approved:      approved,
		GrantedScopes: granted,
	})
	if err != nil {
		t.Fatalf("デバイス認可の要求の更新に失敗: %v", err)
	}
	if output.DeniedReason != "" || output.Approved != approved {
		t.Fatalf("更新の結果 = %+v、Approved = %v を期待", output, approved)
	}
}

// pollDeviceTokenは、デバイスコードのグラントのトークン要求を1回送り、ステータスと `error` を返す。
// golang.org/x/oauth2のDeviceAccessTokenは `interval` を待ってからポーリングし、`slow_down`・
// `authorization_pending` では待ち続けるため、1回ごとの応答を確かめるときはこれを使う
func pollDeviceToken(t *testing.T, srv *httptest.Server, clientID, deviceCode string) (int, string) {
	t.Helper()

	resp, err := srv.Client().PostForm(srv.URL+model.OAuthTokenEndpointPath, url.Values{
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		"device_code": {deviceCode},
		"client_id":   {clientID},
	})
	if err != nil {
		t.Fatalf("トークン要求に失敗: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var body struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("トークン要求の応答をJSONとして読めない: %v", err)
	}
	return resp.StatusCode, body.Error
}

// assertDevicePollErrorは、デバイスコードのトークン要求が400とerrorCodeで拒否されることを確かめる
func assertDevicePollError(t *testing.T, srv *httptest.Server, clientID, deviceCode, errorCode string) {
	t.Helper()

	status, got := pollDeviceToken(t, srv, clientID, deviceCode)
	if status != http.StatusBadRequest || got != errorCode {
		t.Errorf("応答 = (%d, %q)、期待値 = (400, %q)", status, got, errorCode)
	}
}

// TestOAuthDeviceFlowは、golang.org/x/oauth2のデバイス認可のクライアントで、認可サーバーのメタデータに
// 書かれたエンドポイントだけを使い、要求 → 入力・同意 → ポーリング → APIの呼び出しの一連の流れを行えることと、
// 交換済みのデバイスコードの再提示を拒否して発行したトークンを失効させることを確かめる
func TestOAuthDeviceFlow(t *testing.T) {
	t.Parallel()

	q := query.New(testutil.GetTestDB())
	f := setupOAuthFlowFixture(t)
	ctx := context.Background()

	cfg := &config.Config{Domain: "example.com"}
	srv := newOAuthFlowTestServer(t, cfg, q)
	ctx = context.WithValue(ctx, oauth2.HTTPClient, srv.Client())

	// 認可サーバーのメタデータがデバイス認可のエンドポイントとグラントを示す
	var server struct {
		TokenEndpoint               string   `json:"token_endpoint"`
		DeviceAuthorizationEndpoint string   `json:"device_authorization_endpoint"`
		GrantTypesSupported         []string `json:"grant_types_supported"`
	}
	getDiscoveryJSON(t, cfg, srv, cfg.AppURL()+model.OAuthAuthorizationServerMetadataPath, &server)
	if !slices.Contains(server.GrantTypesSupported, "urn:ietf:params:oauth:grant-type:device_code") {
		t.Fatalf("grant_types_supported = %v、デバイスコードのグラントを期待", server.GrantTypesSupported)
	}
	conf := &oauth2.Config{
		ClientID: f.clientID,
		Scopes:   []string{"page:read", "topic:read"},
		Endpoint: oauth2.Endpoint{
			DeviceAuthURL: srv.URL + testServerPath(t, cfg, server.DeviceAuthorizationEndpoint),
			TokenURL:      srv.URL + testServerPath(t, cfg, server.TokenEndpoint),
			AuthStyle:     oauth2.AuthStyleInParams,
		},
	}

	// 要求
	da := requestDeviceAuthorization(t, ctx, cfg, conf)
	if da.Interval != 5 {
		t.Errorf("interval = %d、期待値 = 5", da.Interval)
	}

	// 利用者が許可する前のポーリングは許可待ち
	assertDevicePollError(t, srv, f.clientID, da.DeviceCode, "authorization_pending")

	// 入力・同意。要求したスコープのうちpage:readだけを許可する
	decideDeviceAuthorization(t, q, f, da.UserCode, true, []string{"page:read"})

	// ポーリング。クライアントは要求の時点のスコープを付けて送るが、同意で選んだ範囲のトークンが発行される
	token, err := conf.DeviceAccessToken(ctx, da, oauthDeviceFlowResource(t, cfg))
	if err != nil {
		t.Fatalf("デバイスコードの交換に失敗: %v", err)
	}
	if !strings.HasPrefix(token.AccessToken, string(auth.OAuthAccessTokenPrefix)) || !strings.HasPrefix(token.RefreshToken, string(auth.OAuthRefreshTokenPrefix)) {
		t.Fatalf("トークン = (%q, %q)、接頭辞 wka_・wkr_ を期待", token.AccessToken, token.RefreshToken)
	}
	if scope := token.Extra("scope"); scope != "page:read" {
		t.Errorf("scope = %v、期待値 = %q", scope, "page:read")
	}

	// APIの呼び出し
	if status := getCurrentSpaceMemberStatus(t, srv, token.AccessToken); status != http.StatusOK {
		t.Fatalf("交換したトークンでのステータス = %d、期待値 = 200", status)
	}

	// 発行したリフレッシュトークンで更新できる
	refreshed, err := conf.TokenSource(ctx, &oauth2.Token{RefreshToken: token.RefreshToken, Expiry: time.Now().Add(-time.Minute)}).Token()
	if err != nil {
		t.Fatalf("リフレッシュトークンでの更新に失敗: %v", err)
	}

	// 交換の再試行は拒否し、同じ許可のトークンをすべて失効させる
	assertDevicePollError(t, srv, f.clientID, da.DeviceCode, "invalid_grant")
	if status := getCurrentSpaceMemberStatus(t, srv, refreshed.AccessToken); status != http.StatusUnauthorized {
		t.Errorf("再提示の後の、更新したトークンでのステータス = %d、期待値 = 401", status)
	}
}

// 許可待ちのまま `interval` より早くポーリングすると `slow_down` を返し、以降の間隔を5秒ずつ延ばす
func TestOAuthDeviceFlow_slow_down(t *testing.T) {
	t.Parallel()

	q := query.New(testutil.GetTestDB())
	f := setupOAuthFlowFixture(t)
	cfg := &config.Config{Domain: "example.com"}
	srv := newOAuthFlowTestServer(t, cfg, q)
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, srv.Client())
	da := requestDeviceAuthorization(t, ctx, cfg, newOAuthDeviceFlowConfig(srv, f))

	assertDevicePollError(t, srv, f.clientID, da.DeviceCode, "authorization_pending")
	assertDevicePollError(t, srv, f.clientID, da.DeviceCode, "slow_down")
	assertDevicePollError(t, srv, f.clientID, da.DeviceCode, "slow_down")

	stored, err := repository.NewOAuthDeviceAuthorizationRepository(q).FindByDeviceCodeDigest(ctx, auth.DigestOpaqueToken(da.DeviceCode))
	if err != nil {
		t.Fatalf("デバイス認可の要求の取得に失敗: %v", err)
	}
	if stored.Interval != 15*time.Second {
		t.Errorf("Interval = %v、期待値 = 15s", stored.Interval)
	}

	// 延ばした間隔を空けたポーリングは、許可待ちに戻る
	testutil.CommitFixture(t, func(tx *sql.Tx) {
		if _, err := tx.ExecContext(ctx, "UPDATE oauth_device_authorizations SET last_polled_at = NOW() - interval '15 seconds' WHERE id = $1", string(stored.ID)); err != nil {
			t.Fatalf("最後のポーリングの日時の更新に失敗: %v", err)
		}
	})
	assertDevicePollError(t, srv, f.clientID, da.DeviceCode, "authorization_pending")
}

// 利用者の拒否・期限切れ・別のクライアントの送ったデバイスコードを、それぞれのエラーで拒否する
func TestOAuthDeviceFlow_交換できない要求(t *testing.T) {
	t.Parallel()

	q := query.New(testutil.GetTestDB())
	f := setupOAuthFlowFixture(t)
	cfg := &config.Config{Domain: "example.com"}
	srv := newOAuthFlowTestServer(t, cfg, q)
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, srv.Client())
	conf := newOAuthDeviceFlowConfig(srv, f)

	t.Run("利用者が拒否するとaccess_denied", func(t *testing.T) {
		da := requestDeviceAuthorization(t, ctx, cfg, conf)
		decideDeviceAuthorization(t, q, f, da.UserCode, false, nil)

		_, err := conf.DeviceAccessToken(ctx, da)
		assertRetrieveError(t, err, "access_denied")
	})

	t.Run("期限が切れるとexpired_token", func(t *testing.T) {
		da := requestDeviceAuthorization(t, ctx, cfg, conf)
		decideDeviceAuthorization(t, q, f, da.UserCode, true, []string{"page:read"})
		testutil.CommitFixture(t, func(tx *sql.Tx) {
			if _, err := tx.ExecContext(ctx, "UPDATE oauth_device_authorizations SET expires_at = NOW() - interval '1 second' WHERE device_code_digest = $1",
				auth.DigestOpaqueToken(da.DeviceCode)); err != nil {
				t.Fatalf("有効期限の更新に失敗: %v", err)
			}
		})

		assertDevicePollError(t, srv, f.clientID, da.DeviceCode, "expired_token")
	})

	t.Run("別のクライアントのデバイスコードはinvalid_grant", func(t *testing.T) {
		otherClientID := f.clientID + "-other"
		testutil.CommitFixture(t, func(tx *sql.Tx) {
			testutil.NewOAuthApplicationBuilder(t, tx).WithSpaceID(f.spaceID).WithClientID(otherClientID).Build()
		})
		da := requestDeviceAuthorization(t, ctx, cfg, conf)
		decideDeviceAuthorization(t, q, f, da.UserCode, true, []string{"page:read"})

		assertDevicePollError(t, srv, otherClientID, da.DeviceCode, "invalid_grant")

		// 要求したクライアントは、拒否された交換の後も交換できる
		status, errorCode := pollDeviceToken(t, srv, f.clientID, da.DeviceCode)
		if status != http.StatusOK {
			t.Errorf("要求したクライアントでの応答 = (%d, %q)、期待値 = 200", status, errorCode)
		}
	})
}
