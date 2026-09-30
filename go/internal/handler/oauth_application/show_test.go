package oauth_application_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestShow_アプリの詳細を表示する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "oauth-app-show"
	m := setupAppMember(t, tx, identifier, "閲覧するメンバー", []model.Scope{model.ScopeOAuthApplicationRead}, true)

	appID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(m.spaceID).
		WithCreatedSpaceMemberID(m.spaceMemberID).
		WithName("連携するWebアプリ").
		WithClientID("oauth-app-show-client").
		WithConfidentialClientSecretDigest("oauth_app_show_secret").
		WithRedirectURIs([]string{"https://example.com/callback", "https://example.com/callback2"}).
		Build()

	path := "/s/" + identifier + "/settings/oauth_applications/" + string(appID)
	req := newRequest(t, http.MethodGet, path, identifier, m.userID, nil)
	rr := httptest.NewRecorder()
	setupHandler(t, q).Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Cache-Control"); got != "private, no-cache" {
		t.Errorf("Cache-Control = %q、期待値 = %q", got, "private, no-cache")
	}

	body := rr.Body.String()
	for _, want := range []string{
		"<title>連携するWebアプリ | OAuthアプリ | アプリのスペース</title>",
		`value="oauth-app-show-client"`,
		"サーバーで動くアプリ (confidential)",
		"発行したときにだけ表示しました",
		"https://example.com/callback2",
		"閲覧するメンバー",
		`href="/s/` + identifier + `/settings/oauth_applications"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}
	if strings.Contains(body, "oauth_app_show_secret") {
		t.Error("レスポンスにシークレットのダイジェストが含まれている")
	}
	// oauth_application:readだけでは編集・再発行・削除の操作を出さない
	for _, unwanted := range []string{
		`href="` + path + `/edit"`,
		`action="` + path + `/client_secret"`,
		`name="_method" value="DELETE"`,
	} {
		if strings.Contains(body, unwanted) {
			t.Errorf("レスポンスに%qが含まれている", unwanted)
		}
	}
}

func TestShow_権限に応じて編集と再発行と削除の操作を出す(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "oauth-app-show-actions"
	m := setupAppMember(t, tx, identifier, "管理するメンバー", []model.Scope{model.ScopeOAuthApplicationWrite, model.ScopeOAuthApplicationDelete}, true)

	confidentialID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(m.spaceID).
		WithClientID("oauth-app-show-actions-confidential").
		WithConfidentialClientSecretDigest("oauth_app_show_actions_secret").
		Build()
	publicID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(m.spaceID).
		WithClientID("oauth-app-show-actions-public").
		Build()

	t.Run("confidentialクライアントには再発行を含むすべての操作を出す", func(t *testing.T) {
		path := "/s/" + identifier + "/settings/oauth_applications/" + string(confidentialID)
		req := newRequest(t, http.MethodGet, path, identifier, m.userID, nil)
		rr := httptest.NewRecorder()
		setupHandler(t, q).Show(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
		}
		body := rr.Body.String()
		for _, want := range []string{
			`href="` + path + `/edit"`,
			`action="` + path + `/client_secret"`,
			`action="` + path + `"`,
			`name="_method" value="DELETE"`,
			"このアプリを削除する",
		} {
			if !strings.Contains(body, want) {
				t.Errorf("レスポンスに%qが含まれていない", want)
			}
		}
	})

	t.Run("publicクライアントには再発行を出さない", func(t *testing.T) {
		path := "/s/" + identifier + "/settings/oauth_applications/" + string(publicID)
		req := newRequest(t, http.MethodGet, path, identifier, m.userID, nil)
		rr := httptest.NewRecorder()
		setupHandler(t, q).Show(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
		}
		body := rr.Body.String()
		if strings.Contains(body, `action="`+path+`/client_secret"`) {
			t.Error("publicクライアントの詳細に再発行のフォームが含まれている")
		}
		if !strings.Contains(body, `href="`+path+`/edit"`) {
			t.Error("publicクライアントの詳細に編集へのリンクが含まれていない")
		}
	})
}

func TestShow_見られないアプリは404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "oauth-app-show-missing"
	m := setupAppMember(t, tx, identifier, "閲覧するメンバー", []model.Scope{model.ScopeOAuthApplicationRead}, true)
	other := setupAppMember(t, tx, "oauth-app-show-missing-other", "別のスペースのメンバー", []model.Scope{model.ScopeOAuthApplicationRead}, true)
	otherAppID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(other.spaceID).
		WithClientID("oauth-app-show-other-client").
		Build()

	// サブテストはフィクスチャのトランザクションを共有するため、並行に走らせない。
	for _, tt := range []struct {
		name  string
		appID string
	}{
		{name: "別のスペースのアプリ", appID: string(otherAppID)},
		{name: "UUIDでないID", appID: "not-a-uuid"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := "/s/" + identifier + "/settings/oauth_applications/" + tt.appID
			req := newRequest(t, http.MethodGet, path, identifier, m.userID, nil)
			rr := httptest.NewRecorder()
			setupHandler(t, q).Show(rr, req)

			if rr.Code != http.StatusNotFound {
				t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
			}
		})
	}
}

func TestShow_単独の操作権限だけを持つメンバーに許可された操作を出す(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		identifier   string
		scope        model.Scope
		confidential bool
		wantEdit     bool
		wantSecret   bool
		wantDelete   bool
	}{
		{name: "writeのみのconfidentialクライアント", identifier: "oauth-app-write-only", scope: model.ScopeOAuthApplicationWrite, confidential: true, wantEdit: true, wantSecret: true},
		{name: "writeのみのpublicクライアント", identifier: "oauth-app-write-public", scope: model.ScopeOAuthApplicationWrite, wantEdit: true},
		{name: "deleteのみのconfidentialクライアント", identifier: "oauth-app-delete-only", scope: model.ScopeOAuthApplicationDelete, confidential: true, wantDelete: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			m := setupAppMember(t, tx, tt.identifier, "操作するメンバー", []model.Scope{tt.scope}, true)
			builder := testutil.NewOAuthApplicationBuilder(t, tx).
				WithSpaceID(m.spaceID).
				WithClientID(tt.identifier + "-client")
			if tt.confidential {
				builder = builder.WithConfidentialClientSecretDigest(tt.identifier + "-digest")
			}
			appID := builder.Build()

			path := "/s/" + tt.identifier + "/settings/oauth_applications/" + string(appID)
			req := newRequest(t, http.MethodGet, path, tt.identifier, m.userID, nil)
			rr := httptest.NewRecorder()
			setupHandler(t, q).Show(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
			}
			for _, check := range []struct {
				name string
				text string
				want bool
			}{
				{name: "編集", text: `href="` + path + `/edit"`, want: tt.wantEdit},
				{name: "シークレット再発行", text: `action="` + path + `/client_secret"`, want: tt.wantSecret},
				{name: "削除", text: `name="_method" value="DELETE"`, want: tt.wantDelete},
			} {
				if got := strings.Contains(rr.Body.String(), check.text); got != check.want {
					t.Errorf("%sの操作の表示 = %v、期待値 = %v", check.name, got, check.want)
				}
			}
		})
	}
}
