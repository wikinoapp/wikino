package oauth_application_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestEdit_今の値を入れた編集フォームを表示する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "oauth-app-edit"
	m := setupAppMember(t, tx, identifier, "編集するメンバー", model.SpaceRoleAdmin, true)
	appID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(m.spaceID).
		WithName("連携するWebアプリ").
		WithClientID("oauth-app-edit-client").
		WithConfidentialClientSecretDigest("oauth_app_edit_secret").
		WithRedirectURIs([]string{"https://example.com/callback", "https://example.com/callback2"}).
		Build()

	path := "/s/" + identifier + "/settings/oauth_applications/" + string(appID)
	req := newRequest(t, http.MethodGet, path+"/edit", identifier, m.userID, nil)
	rr := httptest.NewRecorder()
	setupHandler(t, q).Edit(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Cache-Control"); got != "private, no-cache" {
		t.Errorf("Cache-Control = %q、期待値 = %q", got, "private, no-cache")
	}

	body := rr.Body.String()
	for _, want := range []string{
		"<title>連携するWebアプリの編集 | OAuthアプリ | アプリのスペース</title>",
		`action="` + path + `"`,
		`name="_method" value="PATCH"`,
		`name="csrf_token"`,
		`name="version" value="1"`,
		`value="連携するWebアプリ"`,
		"https://example.com/callback\nhttps://example.com/callback2",
		"サーバーで動くアプリ (confidential)",
		`href="` + path + `"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}
	// クライアントの種別は編集できないため、入力欄を出さない
	if strings.Contains(body, `name="client_type"`) {
		t.Error("レスポンスにクライアントの種別の入力欄が含まれている")
	}
	// 登録フォームと同じく、必須項目のラベルの中に必須のバッジを出す
	for _, tc := range []struct {
		id    string
		label string
	}{
		{id: "name", label: "名前"},
		{id: "redirect_uris", label: "リダイレクトURI"},
	} {
		pattern := regexp.QuoteMeta(`<label class="label" for="`+tc.id+`">`) + `\s*` + regexp.QuoteMeta(tc.label) + `\s*<span class="badge" data-variant="outline">\s*必須\s*</span>\s*</label>`
		if !regexp.MustCompile(pattern).MatchString(body) {
			t.Errorf("%sのラベルの中に必須のバッジが無い", tc.label)
		}
	}
}

func TestEdit_編集できないメンバーとアプリには404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "oauth-app-edit-denied"
	writer := setupAppMember(t, tx, identifier, "編集するメンバー", model.SpaceRoleAdmin, true)
	reader := setupAppMember(t, tx, identifier+"-reader", "編集者のメンバー", model.SpaceRoleEditor, true)
	readerAppID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(reader.spaceID).
		WithClientID("oauth-app-edit-reader-client").
		Build()

	// サブテストはフィクスチャのトランザクションを共有するため、並行に走らせない。
	for _, tt := range []struct {
		name       string
		identifier string
		userID     model.UserID
		appID      string
	}{
		{name: "oauth_application:*を持たない編集者", identifier: identifier + "-reader", userID: reader.userID, appID: string(readerAppID)},
		{name: "別のスペースのアプリ", identifier: identifier, userID: writer.userID, appID: string(readerAppID)},
		{name: "UUIDでないID", identifier: identifier, userID: writer.userID, appID: "not-a-uuid"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := "/s/" + tt.identifier + "/settings/oauth_applications/" + tt.appID + "/edit"
			req := newRequest(t, http.MethodGet, path, tt.identifier, tt.userID, nil)
			rr := httptest.NewRecorder()
			setupHandler(t, q).Edit(rr, req)

			if rr.Code != http.StatusNotFound {
				t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
			}
		})
	}
}
