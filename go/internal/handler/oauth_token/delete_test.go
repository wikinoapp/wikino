package oauth_token_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// buildTokensは、fの許可に属するアクセストークンとリフレッシュトークンを作り、それぞれの値を返す
func (f tokenFixture) buildTokens(t *testing.T, tx *sql.Tx, key string) (accessToken, refreshToken string) {
	t.Helper()

	accessToken = string(auth.OAuthAccessTokenPrefix) + key + "-access"
	testutil.NewOAuthAccessTokenBuilder(t, tx).
		WithOAuthGrantID(f.grantID).
		WithSpaceID(f.spaceID).
		WithTokenDigest(auth.DigestOpaqueToken(accessToken)).
		Build()
	refreshToken = string(auth.OAuthRefreshTokenPrefix) + key + "-refresh"
	testutil.NewOAuthRefreshTokenBuilder(t, tx).
		WithOAuthGrantID(f.grantID).
		WithSpaceID(f.spaceID).
		WithTokenDigest(auth.DigestOpaqueToken(refreshToken)).
		WithExpiresAt(time.Now().Add(24 * time.Hour)).
		Build()
	return accessToken, refreshToken
}

func newRevocationRequest(form url.Values) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/oauth/revoke", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

// isAccessTokenRevokedはアクセストークンtokenが失効しているかを返す
func isAccessTokenRevoked(t *testing.T, q *query.Queries, token string) bool {
	t.Helper()

	got, err := repository.NewOAuthAccessTokenRepository(q).FindByTokenDigest(context.Background(), auth.DigestOpaqueToken(token))
	if err != nil {
		t.Fatalf("アクセストークンのFindByTokenDigest()のエラー = %v", err)
	}
	return got.IsRevoked()
}

// isRefreshTokenRevokedはリフレッシュトークンtokenが失効しているかを返す
func isRefreshTokenRevoked(t *testing.T, q *query.Queries, token string) bool {
	t.Helper()

	got, err := repository.NewOAuthRefreshTokenRepository(q).FindByTokenDigest(context.Background(), auth.DigestOpaqueToken(token))
	if err != nil {
		t.Fatalf("リフレッシュトークンのFindByTokenDigest()のエラー = %v", err)
	}
	return got.IsRevoked()
}

// assertRevokedResponseは、失効の要求が成功した応答 (本文の無い200) であることを確かめる
func assertRevokedResponse(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータス = %d、期待値 = %d (本文: %s)", rr.Code, http.StatusOK, rr.Body.String())
	}
	if rr.Body.Len() != 0 {
		t.Errorf("本文 = %q、空を期待", rr.Body.String())
	}
	if got := rr.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q、期待値 = %q", got, "no-store")
	}
}

func TestDelete_client_secret_basicでリフレッシュトークンを失効する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	f := setupTokenFixture(t, tx, "oauth-revoke-refresh", true)
	accessToken, refreshToken := f.buildTokens(t, tx, "oauth-revoke-refresh")

	req := newRevocationRequest(url.Values{"token": {refreshToken}, "token_type_hint": {"refresh_token"}})
	setBasicAuth(req, f.clientID, testClientSecret)
	rr := httptest.NewRecorder()
	setupHandler(t, q).Delete(rr, req)

	assertRevokedResponse(t, rr)
	if !isRefreshTokenRevoked(t, q, refreshToken) {
		t.Error("リフレッシュトークンが失効していない")
	}
	if !isAccessTokenRevoked(t, q, accessToken) {
		t.Error("同じ許可のアクセストークンが失効していない")
	}
}

func TestDelete_client_secret_postでアクセストークンを失効する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	f := setupTokenFixture(t, tx, "oauth-revoke-access", true)
	accessToken, refreshToken := f.buildTokens(t, tx, "oauth-revoke-access")

	// ヒントがトークンの種類と違っても、トークンの接頭辞で見分けて失効させる
	rr := httptest.NewRecorder()
	setupHandler(t, q).Delete(rr, newRevocationRequest(url.Values{
		"token":           {accessToken},
		"token_type_hint": {"refresh_token"},
		"client_id":       {f.clientID},
		"client_secret":   {testClientSecret},
	}))

	assertRevokedResponse(t, rr)
	if !isAccessTokenRevoked(t, q, accessToken) {
		t.Error("アクセストークンが失効していない")
	}
	if isRefreshTokenRevoked(t, q, refreshToken) {
		t.Error("アクセストークンの失効でリフレッシュトークンまで失効した")
	}
}

func TestDelete_無効なトークンにも200を返す(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupTokenFixture(t, tx, "oauth-revoke-unknown", false)

	rr := httptest.NewRecorder()
	setupHandler(t, testutil.QueriesWithTx(tx)).Delete(rr, newRevocationRequest(url.Values{
		"token":     {string(auth.OAuthRefreshTokenPrefix) + "unknown"},
		"client_id": {f.clientID},
	}))

	assertRevokedResponse(t, rr)
}

func TestDelete_拒否する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// prepareは、fのアクセストークンtokenの失効を求める要求を作る
		prepare          func(f tokenFixture, token string) *http.Request
		wantStatus       int
		wantError        string
		wantAuthenticate string
	}{
		{
			name: "シークレットの誤り",
			prepare: func(f tokenFixture, token string) *http.Request {
				req := newRevocationRequest(url.Values{"token": {token}})
				setBasicAuth(req, f.clientID, "wks_wrong")
				return req
			},
			wantStatus:       http.StatusUnauthorized,
			wantError:        "invalid_client",
			wantAuthenticate: `Basic realm="wikino"`,
		},
		{
			name: "confidentialクライアントがシークレットを送らない",
			prepare: func(f tokenFixture, token string) *http.Request {
				return newRevocationRequest(url.Values{"token": {token}, "client_id": {f.clientID}})
			},
			wantStatus:       http.StatusUnauthorized,
			wantError:        "invalid_client",
			wantAuthenticate: `Basic realm="wikino"`,
		},
		{
			name: "Basicと本文のシークレットを同時に使う",
			prepare: func(f tokenFixture, token string) *http.Request {
				req := newRevocationRequest(url.Values{"token": {token}, "client_secret": {testClientSecret}})
				setBasicAuth(req, f.clientID, testClientSecret)
				return req
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid_request",
		},
		{
			name: "トークンが無い",
			prepare: func(f tokenFixture, _ string) *http.Request {
				req := newRevocationRequest(url.Values{})
				setBasicAuth(req, f.clientID, testClientSecret)
				return req
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid_request",
		},
		{
			name: "トークンの重複",
			prepare: func(f tokenFixture, token string) *http.Request {
				req := newRevocationRequest(url.Values{"token": {token, token}})
				setBasicAuth(req, f.clientID, testClientSecret)
				return req
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid_request",
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			key := "oauth-revoke-reject-" + string(rune('a'+i))
			f := setupTokenFixture(t, tx, key, true)
			accessToken, _ := f.buildTokens(t, tx, key)

			rr := httptest.NewRecorder()
			setupHandler(t, q).Delete(rr, tt.prepare(f, accessToken))

			body := assertJSONResponse(t, rr, tt.wantStatus)
			if body["error"] != tt.wantError {
				t.Errorf("error = %v、期待値 = %q", body["error"], tt.wantError)
			}
			if got := rr.Header().Get("WWW-Authenticate"); got != tt.wantAuthenticate {
				t.Errorf("WWW-Authenticate = %q、期待値 = %q", got, tt.wantAuthenticate)
			}
			if isAccessTokenRevoked(t, q, accessToken) {
				t.Error("拒否した要求でアクセストークンが失効した")
			}
		})
	}
}

func TestDelete_別のスペースのクライアントはトークンを失効できない(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	f := setupTokenFixture(t, tx, "oauth-revoke-other-client", false)
	other := setupTokenFixture(t, tx, "oauth-revoke-other-client-other", false)
	accessToken, _ := f.buildTokens(t, tx, "oauth-revoke-other-client")

	rr := httptest.NewRecorder()
	setupHandler(t, q).Delete(rr, newRevocationRequest(url.Values{
		"token":     {accessToken},
		"client_id": {other.clientID},
	}))

	// クライアントの認証には成功しても、このトークンは別のクライアントに発行されている。
	body := assertJSONResponse(t, rr, http.StatusBadRequest)
	if body["error"] != "unauthorized_client" {
		t.Errorf("error = %v、期待値 = unauthorized_client", body["error"])
	}
	if isAccessTokenRevoked(t, q, accessToken) {
		t.Error("別のクライアントの要求でアクセストークンが失効した")
	}

	// 同じクライアントでも、見つからないトークンなら成功として答える。
	missingRR := httptest.NewRecorder()
	setupHandler(t, q).Delete(missingRR, newRevocationRequest(url.Values{
		"token":     {string(auth.OAuthAccessTokenPrefix) + "oauth-revoke-other-client-missing"},
		"client_id": {other.clientID},
	}))
	assertRevokedResponse(t, missingRR)
}
