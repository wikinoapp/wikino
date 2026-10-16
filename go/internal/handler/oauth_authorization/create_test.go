package oauth_authorization_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestCreate_許可すると選んだスコープの認可コードとstateとissを付けて303でクライアントへ戻す(t *testing.T) {
	t.Parallel()

	q := query.New(testutil.GetTestDB())
	var f authorizationFixture
	testutil.CommitFixture(t, func(tx *sql.Tx) {
		f = setupAuthorizationFixture(t, tx, "oauthz-create", model.SpaceRoleAdmin, true)
	})
	params := validParams(f.clientID)
	params.Set("decision", "approve")
	params["granted_scopes"] = []string{"page:read"}

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
		t.Fatalf("code = %+v、期待値は連携先のスペースの認可コード", code)
	}
	if want := []model.Scope{model.ScopePageRead}; !slices.Equal(code.Scopes, want) {
		t.Errorf("code.Scopes = %v、期待値 = %v (要求のうち選んだスコープだけ)", code.Scopes, want)
	}
}

func TestCreate_選んだスコープに含まれるスコープを補った認可コードを発行する(t *testing.T) {
	t.Parallel()

	q := query.New(testutil.GetTestDB())
	var f authorizationFixture
	testutil.CommitFixture(t, func(tx *sql.Tx) {
		f = setupAuthorizationFixture(t, tx, "oauthz-create-implied", model.SpaceRoleAdmin, true)
	})
	params := validParams(f.clientID)
	params.Set("scope", "draft_page:read draft_page:write")
	params.Set("decision", "approve")
	// 同意画面は、選んだdraft_page:writeに含まれるdraft_page:readを無効にして送信しない
	params["granted_scopes"] = []string{"draft_page:write"}

	rr := httptest.NewRecorder()
	setupHandler(t, q).Create(rr, newRequest(t, http.MethodPost, f.userID, params))

	query := assertRedirectToClient(t, rr, http.StatusSeeOther, map[string]string{"state": "xyz"})
	code, err := repository.NewOAuthAuthorizationCodeRepository(q).FindByCodeDigest(context.Background(), auth.DigestOpaqueToken(query.Get("code")))
	if err != nil {
		t.Fatalf("FindByCodeDigest()のエラー = %v", err)
	}
	if code == nil {
		t.Fatal("認可コードが保存されていない")
	}
	// 認可コードのスコープが、許可とトークン応答のscopeになる
	if want := []model.Scope{model.ScopeDraftPageRead, model.ScopeDraftPageWrite}; !slices.Equal(code.Scopes, want) {
		t.Errorf("code.Scopes = %v、期待値 = %v", code.Scopes, want)
	}
}

func TestCreate_選んだスコープが不正なら同意画面を422で再表示する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		granted []string
		wantMsg string
		// wantCheckedは、再表示でチェックしたまま戻すスコープのチェックボックスのid
		wantChecked []string
	}{
		{
			name:    "1つも選ばなかった",
			granted: nil,
			wantMsg: "許可する操作を1つ以上選んでください",
		},
		{
			name:        "要求に無いスコープを送信した",
			granted:     []string{"page:read", "page:write"},
			wantMsg:     "要求されていない操作が含まれています",
			wantChecked: []string{"granted_scopes_page_read"},
		},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			f := setupAuthorizationFixture(t, tx, "oauthz-create-invalid-"+strconv.Itoa(i), model.SpaceRoleAdmin, true)
			params := validParams(f.clientID)
			params.Set("decision", "approve")
			params["granted_scopes"] = tt.granted

			rr := httptest.NewRecorder()
			setupHandler(t, testutil.QueriesWithTx(tx)).Create(rr, newRequest(t, http.MethodPost, f.userID, params))

			if rr.Code != http.StatusUnprocessableEntity {
				t.Fatalf("ステータス = %d、期待値 = %d", rr.Code, http.StatusUnprocessableEntity)
			}
			assertResponseHeaders(t, rr)
			if location := rr.Header().Get("Location"); location != "" {
				t.Errorf("Location = %q、期待値 = 空", location)
			}

			body := rr.Body.String()
			for _, want := range []string{tt.wantMsg, `aria-describedby="granted_scopes-error"`, `name="state" value="xyz"`} {
				if !strings.Contains(body, want) {
					t.Errorf("本文に%qが含まれていない", want)
				}
			}
			for _, id := range []string{"granted_scopes_topic_read", "granted_scopes_page_read"} {
				checked := regexp.MustCompile(`id="` + id + `"[^>]*checked`).MatchString(body)
				if want := slices.Contains(tt.wantChecked, id); checked != want {
					t.Errorf("%sのチェック = %v、期待値 = %v", id, checked, want)
				}
			}

			var count int
			if err := tx.QueryRowContext(context.Background(), "SELECT count(*) FROM oauth_grants WHERE space_id = $1", string(f.spaceID)).Scan(&count); err != nil {
				t.Fatalf("許可の数え上げに失敗: %v", err)
			}
			if count != 0 {
				t.Errorf("許可 = %d件、期待値 = 0件", count)
			}
		})
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
