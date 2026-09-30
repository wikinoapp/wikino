package oauth_authorization_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestCreate_許可すると認可コードとstateとissを付けて303でクライアントへ戻す(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	f := setupAuthorizationFixture(t, tx, "oauthz-create", model.SpaceRoleAdmin, true)
	params := validParams(f.clientID)
	params.Set("decision", "approve")

	rr := httptest.NewRecorder()
	setupHandler(t, q).Create(rr, newRequest(t, http.MethodPost, f.userID, params))

	assertResponseHeaders(t, rr)
	query := assertRedirectToClient(t, rr, http.StatusSeeOther, map[string]string{"state": "xyz"})
	if query.Has("error") {
		t.Errorf("error = %q、期待値 = 無し", query.Get("error"))
	}

	code, err := repository.NewOAuthAuthorizationCodeRepository(q).FindByCodeDigest(context.Background(), auth.DigestOpaqueToken(query.Get("code")))
	if err != nil {
		t.Fatalf("FindByCodeDigest()のエラー = %v", err)
	}
	if code == nil || code.SpaceID != f.spaceID {
		t.Errorf("code = %+v、期待値は連携先のスペースの認可コード", code)
	}
}

func TestCreate_拒否するとaccess_deniedを付けて303でクライアントへ戻す(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupAuthorizationFixture(t, tx, "oauthz-create-deny", model.SpaceRoleAdmin, true)
	params := validParams(f.clientID)
	params.Set("decision", "deny")

	rr := httptest.NewRecorder()
	setupHandler(t, testutil.QueriesWithTx(tx)).Create(rr, newRequest(t, http.MethodPost, f.userID, params))

	query := assertRedirectToClient(t, rr, http.StatusSeeOther, map[string]string{
		"error": "access_denied",
		"state": "xyz",
	})
	if query.Has("code") {
		t.Error("拒否の応答に認可コードが付いている")
	}
}

func TestCreate_リダイレクトURIが不正ならクライアントへ戻さない(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupAuthorizationFixture(t, tx, "oauthz-create-bad-redirect", model.SpaceRoleAdmin, true)
	params := validParams(f.clientID)
	params.Set("redirect_uri", "https://evil.example/callback")
	params.Set("decision", "approve")

	rr := httptest.NewRecorder()
	setupHandler(t, testutil.QueriesWithTx(tx)).Create(rr, newRequest(t, http.MethodPost, f.userID, params))

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("ステータス = %d、期待値 = %d", rr.Code, http.StatusBadRequest)
	}
	if location := rr.Header().Get("Location"); location != "" {
		t.Errorf("Location = %q、期待値 = 空", location)
	}
}

func TestCreate_未ログインならサインインへ送る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupAuthorizationFixture(t, tx, "oauthz-create-signin", model.SpaceRoleAdmin, true)
	params := validParams(f.clientID)
	params.Set("decision", "approve")

	rr := httptest.NewRecorder()
	setupHandler(t, testutil.QueriesWithTx(tx)).Create(rr, newRequest(t, http.MethodPost, "", params))

	if rr.Code != http.StatusFound || rr.Header().Get("Location") != "/sign_in" {
		t.Errorf("ステータス = %d・Location = %q、期待値 = 302・/sign_in", rr.Code, rr.Header().Get("Location"))
	}
}
