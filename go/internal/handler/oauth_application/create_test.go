package oauth_application_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// clientSecretPatternは登録直後の画面に出るクライアントシークレットを取り出す
var clientSecretPattern = regexp.MustCompile(`value="(wks_[A-Za-z0-9_-]+)"`)

// clientIDPatternは画面に出るクライアントIDを取り出す
var clientIDPattern = regexp.MustCompile(`id="client_id"[^>]*value="([A-Za-z0-9_-]+)"`)

func TestCreate_confidentialクライアントのシークレットをその応答でだけ表示する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "oauth-app-create"
	m := setupAppMember(t, tx, identifier, "登録するメンバー", model.SpaceRoleAdmin, true)

	form := url.Values{
		"name":          {"連携するWebアプリ"},
		"redirect_uris": {"https://example.com/callback\r\nhttps://example.com/callback2"},
		"client_type":   {"confidential"},
	}
	req := newRequest(t, http.MethodPost, "/s/"+identifier+"/settings/oauth_applications", identifier, m.userID, form)
	rr := httptest.NewRecorder()
	setupHandler(t, q).Create(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q、期待値 = %q", got, "no-store")
	}

	body := rr.Body.String()
	secret := clientSecretPattern.FindStringSubmatch(body)
	if secret == nil {
		t.Fatal("レスポンスにクライアントシークレットが含まれていない")
	}
	clientID := clientIDPattern.FindStringSubmatch(body)
	if clientID == nil {
		t.Fatal("レスポンスにクライアントIDが含まれていない")
	}
	if !strings.Contains(body, "二度と表示できない") {
		t.Error("レスポンスにシークレットを1度だけ表示することの注意が含まれていない")
	}

	app, err := repository.NewOAuthApplicationRepository(q).FindByClientIDAndSpaceID(context.Background(), clientID[1], m.spaceID)
	if err != nil {
		t.Fatalf("FindByClientIDAndSpaceID()のエラー = %v", err)
	}
	if app == nil {
		t.Fatal("表示したクライアントIDのアプリがデータベースに無い")
	}
	if app.ClientSecretDigest == nil || *app.ClientSecretDigest != auth.DigestOpaqueToken(secret[1]) {
		t.Errorf("ClientSecretDigest = %v、期待値は表示したシークレットのダイジェスト", app.ClientSecretDigest)
	}
	if app.Name != "連携するWebアプリ" || app.CreatedSpaceMemberID == nil || *app.CreatedSpaceMemberID != m.spaceMemberID {
		t.Errorf("保存されたアプリ = %+v", app)
	}
	if want := []string{"https://example.com/callback", "https://example.com/callback2"}; !slices.Equal(app.RedirectURIs, want) {
		t.Errorf("RedirectURIs = %q、期待値 = %q", app.RedirectURIs, want)
	}
	if !strings.Contains(body, `href="/s/`+identifier+`/settings/oauth_applications/`+string(app.ID)+`"`) {
		t.Error("レスポンスに詳細へのリンクが含まれていない")
	}
}

func TestCreate_publicクライアントは詳細へリダイレクトする(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "oauth-app-create-public"
	m := setupAppMember(t, tx, identifier, "登録するメンバー", model.SpaceRoleAdmin, true)

	form := url.Values{
		"name":          {"手元のCLI"},
		"redirect_uris": {"http://127.0.0.1/callback"},
		"client_type":   {"public"},
	}
	req := newRequest(t, http.MethodPost, "/s/"+identifier+"/settings/oauth_applications", identifier, m.userID, form)
	rr := httptest.NewRecorder()
	setupHandler(t, q).Create(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusSeeOther)
	}

	apps, err := repository.NewOAuthApplicationRepository(q).ListBySpace(context.Background(), m.spaceID)
	if err != nil {
		t.Fatalf("ListBySpace()のエラー = %v", err)
	}
	if len(apps) != 1 || apps[0].IsConfidential() || apps[0].ClientSecretDigest != nil {
		t.Fatalf("登録されたアプリ = %v、期待値はシークレットを持たないpublicクライアント1件", apps)
	}
	if want := "/s/" + identifier + "/settings/oauth_applications/" + string(apps[0].ID); rr.Header().Get("Location") != want {
		t.Errorf("Location = %q、期待値 = %q", rr.Header().Get("Location"), want)
	}
	if clientSecretPattern.MatchString(rr.Body.String()) {
		t.Error("publicクライアントの応答にシークレットが含まれている")
	}
}

func TestCreate_入力が不正なら422で送信内容を戻したフォームを再描画する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "oauth-app-create-invalid"
	m := setupAppMember(t, tx, identifier, "登録するメンバー", model.SpaceRoleAdmin, true)

	form := url.Values{
		"name":          {"連携するWebアプリ"},
		"redirect_uris": {"http://example.com/callback"},
		"client_type":   {"public"},
	}
	req := newRequest(t, http.MethodPost, "/s/"+identifier+"/settings/oauth_applications", identifier, m.userID, form)
	rr := httptest.NewRecorder()
	setupHandler(t, q).Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusUnprocessableEntity)
	}
	if got := rr.Header().Get("Cache-Control"); got != "private, no-cache" {
		t.Errorf("Cache-Control = %q、期待値 = %q", got, "private, no-cache")
	}

	body := rr.Body.String()
	for _, want := range []string{
		"「http://example.com/callback」はHTTPSのURLにしてください",
		`id="redirect_uris-error"`,
		`value="連携するWebアプリ"`,
		"http://example.com/callback</textarea>",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}
	if !regexp.MustCompile(`id="client_type_public"[^>]*checked`).MatchString(body) {
		t.Error("送信したクライアントの種別の選択が戻っていない")
	}
	if !regexp.MustCompile(`<textarea\b[^>]*\bautofocus\b[^>]*id="redirect_uris"`).MatchString(body) {
		t.Error("エラーのあるリダイレクトURIの欄にフォーカスしていない")
	}

	apps, err := repository.NewOAuthApplicationRepository(q).ListBySpace(context.Background(), m.spaceID)
	if err != nil {
		t.Fatalf("ListBySpace()のエラー = %v", err)
	}
	if len(apps) != 0 {
		t.Errorf("作られたアプリの件数 = %d、期待値 = 0", len(apps))
	}
}

func TestCreate_エラーのある項目は補足とエラーの両方を指す(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "oauth-app-create-describedby"
	m := setupAppMember(t, tx, identifier, "登録するメンバー", model.SpaceRoleAdmin, true)

	form := url.Values{
		"name":          {""},
		"redirect_uris": {"https://example.com/callback"},
		"client_type":   {"native"},
	}
	req := newRequest(t, http.MethodPost, "/s/"+identifier+"/settings/oauth_applications", identifier, m.userID, form)
	rr := httptest.NewRecorder()
	setupHandler(t, q).Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusUnprocessableEntity)
	}

	body := rr.Body.String()
	for _, tc := range []struct {
		inputID     string
		describedBy string
	}{
		{inputID: "name", describedBy: "name-hint name-error"},
		{inputID: "redirect_uris", describedBy: "redirect_uris-hint"},
		{inputID: "client_type_confidential", describedBy: "client_type_confidential-hint client_type-error"},
		{inputID: "client_type_public", describedBy: "client_type_public-hint client_type-error"},
	} {
		if !regexp.MustCompile(`id="` + tc.inputID + `"[^>]*aria-describedby="` + tc.describedBy + `"`).MatchString(body) {
			t.Errorf("%sのaria-describedbyが%qになっていない", tc.inputID, tc.describedBy)
		}
	}
}

func TestCreate_登録の権限が無ければ404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "oauth-app-create-readonly"
	m := setupAppMember(t, tx, identifier, "編集者のメンバー", model.SpaceRoleEditor, true)

	form := url.Values{
		"name":          {"連携するWebアプリ"},
		"redirect_uris": {"https://example.com/callback"},
		"client_type":   {"confidential"},
	}
	req := newRequest(t, http.MethodPost, "/s/"+identifier+"/settings/oauth_applications", identifier, m.userID, form)
	rr := httptest.NewRecorder()
	setupHandler(t, q).Create(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
	}
}
