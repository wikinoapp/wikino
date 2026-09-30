package oauth_authorization_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestNew_同意画面にアプリ名と連携先のスペースとスコープの説明を示す(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupAuthorizationFixture(t, tx, "oauthz-new", model.SpaceRoleAdmin, true)

	rr := httptest.NewRecorder()
	setupHandler(t, testutil.QueriesWithTx(tx)).New(rr, newRequest(t, http.MethodGet, f.userID, validParams(f.clientID)))

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータス = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	assertResponseHeaders(t, rr)

	body := rr.Body.String()
	for _, want := range []string{
		"連携するCLIにアクセスを許可しますか？",
		"連携先のスペース",
		"@authorizing_user",
		"トピックの読み取り",
		"ページの読み取り",
		"client.example",
		`name="decision" type="submit" value="approve"`,
		`name="decision" type="submit" value="deny"`,
		// 同意の送信へ認可要求を引き継ぐ
		`name="client_id" value="oauthz-new-client"`,
		`name="code_challenge" value="` + testCodeChallenge + `"`,
		`name="state" value="xyz"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("本文に%qが含まれていない", want)
		}
	}
	if strings.Contains(body, "ページの作成・更新") {
		t.Error("要求していないスコープの説明が含まれている")
	}
}

func TestNew_未ログインならサインインへ送りサインイン後に認可要求へ戻す(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupAuthorizationFixture(t, tx, "oauthz-new-signin", model.SpaceRoleAdmin, true)
	req := newRequest(t, http.MethodGet, "", validParams(f.clientID))

	rr := httptest.NewRecorder()
	setupHandler(t, testutil.QueriesWithTx(tx)).New(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("ステータス = %d、期待値 = %d", rr.Code, http.StatusFound)
	}
	location, err := url.Parse(rr.Header().Get("Location"))
	if err != nil {
		t.Fatalf("Locationの解析に失敗: %v", err)
	}
	if location.Path != "/sign_in" {
		t.Errorf("Locationのパス = %q、期待値 = /sign_in", location.Path)
	}
	if got, want := location.Query().Get("back"), req.URL.RequestURI(); got != want {
		t.Errorf("back = %q、期待値 = %q", got, want)
	}
}

func TestNew_許可できないユーザーには理由とクライアントへ戻るボタンを示す(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		identifier string
		role       model.SpaceRole
		outsider   bool
		wantText   string
	}{
		{
			name:       "連携先のスペースのメンバーでない",
			identifier: "oauthz-new-outsider",
			role:       model.SpaceRoleAdmin,
			outsider:   true,
			wantText:   "このスペースのメンバーではないため",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			f := setupAuthorizationFixture(t, tx, tt.identifier, tt.role, true)
			userID := f.userID
			if tt.outsider {
				outsider := setupAuthorizationFixture(t, tx, tt.identifier+"-other", model.SpaceRoleAdmin, true)
				userID = outsider.userID
			}

			rr := httptest.NewRecorder()
			setupHandler(t, testutil.QueriesWithTx(tx)).New(rr, newRequest(t, http.MethodGet, userID, validParams(f.clientID)))

			if rr.Code != http.StatusOK {
				t.Fatalf("ステータス = %d、期待値 = %d", rr.Code, http.StatusOK)
			}
			body := rr.Body.String()
			if !strings.Contains(body, tt.wantText) {
				t.Errorf("本文に%qが含まれていない", tt.wantText)
			}
			if strings.Contains(body, `value="approve"`) {
				t.Error("許可のボタンが含まれている")
			}
			if !strings.Contains(body, `name="decision" type="submit" value="deny"`) {
				t.Error("クライアントへ戻るボタンが含まれていない")
			}
			// 許可も拒否も選べない画面なので、戻るボタンに合わせた移動先の案内を出す
			if !strings.Contains(body, "アプリに戻るとclient.exampleに移動します") {
				t.Error("アプリへ戻ったときの移動先の案内が含まれていない")
			}
			if strings.Contains(body, "許可または拒否すると") {
				t.Error("許可と拒否を選ぶときの移動先の案内が含まれている")
			}
		})
	}
}

func TestNew_クライアントIDかリダイレクトURIが不正ならクライアントへ戻さない(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		identifier string
		modify     func(p url.Values)
		wantText   string
	}{
		{
			name:       "client_idが登録されていない",
			identifier: "oauthz-new-unknown-client",
			modify:     func(p url.Values) { p.Set("client_id", "unknown-client") },
			wantText:   "アプリのクライアントIDが無いか、登録されていません",
		},
		{
			name:       "redirect_uriが登録と一致しない",
			identifier: "oauthz-new-bad-redirect",
			modify:     func(p url.Values) { p.Set("redirect_uri", "https://evil.example/callback") },
			wantText:   "アプリのリダイレクトURIが無いか、登録されたものと一致しません",
		},
		{
			name:       "redirect_uriが2回送られた",
			identifier: "oauthz-new-dup-redirect",
			modify:     func(p url.Values) { p.Add("redirect_uri", "https://evil.example/callback") },
			wantText:   "アプリのリダイレクトURIが無いか、登録されたものと一致しません",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			f := setupAuthorizationFixture(t, tx, tt.identifier, model.SpaceRoleAdmin, true)
			params := validParams(f.clientID)
			tt.modify(params)

			rr := httptest.NewRecorder()
			setupHandler(t, testutil.QueriesWithTx(tx)).New(rr, newRequest(t, http.MethodGet, f.userID, params))

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("ステータス = %d、期待値 = %d", rr.Code, http.StatusBadRequest)
			}
			if location := rr.Header().Get("Location"); location != "" {
				t.Errorf("Location = %q、期待値 = 空 (クライアントへ戻さない)", location)
			}
			assertResponseHeaders(t, rr)
			if !strings.Contains(rr.Body.String(), tt.wantText) {
				t.Errorf("本文に%qが含まれていない", tt.wantText)
			}
		})
	}
}

func TestNew_その他の不正はエラーをリダイレクトURIに付けてクライアントへ戻す(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		identifier string
		modify     func(p url.Values)
		wantError  string
	}{
		{
			name:       "PKCEが無い",
			identifier: "oauthz-new-no-pkce",
			modify: func(p url.Values) {
				p.Del("code_challenge")
				p.Del("code_challenge_method")
			},
			wantError: "invalid_request",
		},
		{
			name:       "response_typeがtoken",
			identifier: "oauthz-new-token",
			modify:     func(p url.Values) { p.Set("response_type", "token") },
			wantError:  "unsupported_response_type",
		},
		{
			name:       "付与できないスコープ",
			identifier: "oauthz-new-admin-scope",
			modify:     func(p url.Values) { p.Set("scope", "space:admin") },
			wantError:  "invalid_scope",
		},
		{
			name:       "resourceが別のスペースを指す",
			identifier: "oauthz-new-other-resource",
			modify:     func(p url.Values) { p.Set("resource", "https://example.com/api/v1/spaces/no-such-space") },
			wantError:  "invalid_target",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			f := setupAuthorizationFixture(t, tx, tt.identifier, model.SpaceRoleAdmin, true)
			params := validParams(f.clientID)
			tt.modify(params)

			rr := httptest.NewRecorder()
			setupHandler(t, testutil.QueriesWithTx(tx)).New(rr, newRequest(t, http.MethodGet, f.userID, params))

			assertResponseHeaders(t, rr)
			query := assertRedirectToClient(t, rr, http.StatusFound, map[string]string{
				"error": tt.wantError,
				"state": "xyz",
			})
			if query.Has("code") {
				t.Error("エラーの応答に認可コードが付いている")
			}
		})
	}
}

func TestNew_壊れたGETクエリは認可要求のエラーにする(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		identifier string
		change     func(*http.Request)
		wantState  string
	}{
		{
			name:       "stateの値が壊れている",
			identifier: "oauthz-new-bad-state",
			change: func(r *http.Request) {
				r.URL.RawQuery = strings.Replace(r.URL.RawQuery, "state=xyz", "state=%ZZ", 1)
			},
		},
		{
			name:       "未使用のパラメーターの値が壊れている",
			identifier: "oauthz-new-bad-extra",
			change: func(r *http.Request) {
				r.URL.RawQuery += "&unused=%ZZ"
			},
			wantState: "xyz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			f := setupAuthorizationFixture(t, tx, tt.identifier, model.SpaceRoleAdmin, true)
			req := newRequest(t, http.MethodGet, f.userID, validParams(f.clientID))
			tt.change(req)

			rr := httptest.NewRecorder()
			setupHandler(t, testutil.QueriesWithTx(tx)).New(rr, req)

			query := assertRedirectToClient(t, rr, http.StatusFound, map[string]string{"error": "invalid_request"})
			if got := query.Get("state"); got != tt.wantState {
				t.Errorf("state = %q、期待値 = %q", got, tt.wantState)
			}
			if query.Has("code") {
				t.Error("不正な要求に認可コードが付いている")
			}
		})
	}
}

func TestNew_壊れたリダイレクトURIを含むGETクエリはクライアントへ戻さない(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupAuthorizationFixture(t, tx, "oauthz-new-bad-query-redirect", model.SpaceRoleAdmin, true)
	params := validParams(f.clientID)
	params.Del("redirect_uri")
	req := newRequest(t, http.MethodGet, f.userID, params)
	req.URL.RawQuery += "&redirect_uri=%ZZ"

	rr := httptest.NewRecorder()
	setupHandler(t, testutil.QueriesWithTx(tx)).New(rr, req)

	if rr.Code != http.StatusBadRequest || rr.Header().Get("Location") != "" {
		t.Errorf("ステータス = %d、Location = %q。クライアントへ戻さない400を期待", rr.Code, rr.Header().Get("Location"))
	}
}

func TestNew_フィーチャーフラグが無効なら404を返す(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupAuthorizationFixture(t, tx, "oauthz-new-noflag", model.SpaceRoleAdmin, false)

	rr := httptest.NewRecorder()
	setupHandler(t, testutil.QueriesWithTx(tx)).New(rr, newRequest(t, http.MethodGet, f.userID, validParams(f.clientID)))

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータス = %d、期待値 = %d", rr.Code, http.StatusNotFound)
	}
}
