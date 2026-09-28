package oauth_application_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestUpdate_名前とリダイレクトURIを更新して詳細へリダイレクトする(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "oauth-app-update"
	m := setupAppMember(t, tx, identifier, "編集するメンバー", []model.Scope{model.ScopeOAuthApplicationWrite}, true)
	appID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(m.spaceID).
		WithName("元の名前").
		WithClientID("oauth-app-update-client").
		Build()

	path := "/s/" + identifier + "/settings/oauth_applications/" + string(appID)
	form := url.Values{
		"version":       {"1"},
		"name":          {"編集したアプリ"},
		"redirect_uris": {"https://example.com/new\r\nhttp://127.0.0.1:8080/new"},
	}
	req := newRequest(t, http.MethodPatch, path, identifier, m.userID, form)
	rr := httptest.NewRecorder()
	setupHandler(t, q).Update(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusSeeOther)
	}
	if got := rr.Header().Get("Location"); got != path {
		t.Errorf("Location = %q、期待値 = %q", got, path)
	}

	app, err := repository.NewOAuthApplicationRepository(q).FindByIDAndSpaceID(context.Background(), appID, m.spaceID)
	if err != nil {
		t.Fatalf("FindByIDAndSpaceID()のエラー = %v", err)
	}
	if app.Name != "編集したアプリ" {
		t.Errorf("Name = %q、期待値 = %q", app.Name, "編集したアプリ")
	}
	if want := []string{"https://example.com/new", "http://127.0.0.1:8080/new"}; !slices.Equal(app.RedirectURIs, want) {
		t.Errorf("RedirectURIs = %q、期待値 = %q", app.RedirectURIs, want)
	}
}

func TestUpdate_入力が不正なら422で送信内容を戻したフォームを再描画する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "oauth-app-update-invalid"
	m := setupAppMember(t, tx, identifier, "編集するメンバー", []model.Scope{model.ScopeOAuthApplicationWrite}, true)
	appID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(m.spaceID).
		WithName("元の名前").
		WithClientID("oauth-app-update-invalid-client").
		Build()

	path := "/s/" + identifier + "/settings/oauth_applications/" + string(appID)
	form := url.Values{
		"version":       {"1"},
		"name":          {"送信した名前"},
		"redirect_uris": {"http://example.com/callback"},
	}
	req := newRequest(t, http.MethodPatch, path, identifier, m.userID, form)
	rr := httptest.NewRecorder()
	setupHandler(t, q).Update(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusUnprocessableEntity)
	}
	body := rr.Body.String()
	for _, want := range []string{
		// 見出しとタイトルは送信した名前ではなく、保存されている名前を使う
		"<title>元の名前の編集 | OAuthアプリ | アプリのスペース</title>",
		`value="送信した名前"`,
		"http://example.com/callback",
		`aria-invalid="true"`,
		`name="_method" value="PATCH"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}

	app, err := repository.NewOAuthApplicationRepository(q).FindByIDAndSpaceID(context.Background(), appID, m.spaceID)
	if err != nil {
		t.Fatalf("FindByIDAndSpaceID()のエラー = %v", err)
	}
	if app.Name != "元の名前" {
		t.Errorf("Name = %q、期待値 = %q (拒否された送信で更新された)", app.Name, "元の名前")
	}
}

func TestUpdate_更新できないメンバーには404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "oauth-app-update-denied"
	m := setupAppMember(t, tx, identifier, "削除だけのメンバー", []model.Scope{model.ScopeOAuthApplicationDelete}, true)
	appID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(m.spaceID).
		WithName("元の名前").
		WithClientID("oauth-app-update-denied-client").
		Build()

	path := "/s/" + identifier + "/settings/oauth_applications/" + string(appID)
	form := url.Values{
		"version":       {"1"},
		"name":          {"編集したアプリ"},
		"redirect_uris": {"https://example.com/new"},
	}
	req := newRequest(t, http.MethodPatch, path, identifier, m.userID, form)
	rr := httptest.NewRecorder()
	setupHandler(t, q).Update(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
	}
}

func TestUpdate_古いフォームは409で現在の値を示す(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "oauth-app-update-conflict"
	m := setupAppMember(t, tx, identifier, "編集するメンバー", []model.Scope{model.ScopeOAuthApplicationWrite}, true)
	appID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(m.spaceID).
		WithName("元の名前").
		WithClientID("oauth-app-update-conflict-client").
		Build()

	path := "/s/" + identifier + "/settings/oauth_applications/" + string(appID)
	h := setupHandler(t, q)
	first := newRequest(t, http.MethodPatch, path, identifier, m.userID, url.Values{
		"version":       {"1"},
		"name":          {"先に保存した名前"},
		"redirect_uris": {"https://example.com/first"},
	})
	firstResponse := httptest.NewRecorder()
	h.Update(firstResponse, first)
	if firstResponse.Code != http.StatusSeeOther {
		t.Fatalf("先の保存のステータスコード = %d、期待値 = %d", firstResponse.Code, http.StatusSeeOther)
	}

	second := newRequest(t, http.MethodPatch, path, identifier, m.userID, url.Values{
		"version":       {"1"},
		"name":          {"古いフォームの名前"},
		"redirect_uris": {"https://example.com/second"},
	})
	secondResponse := httptest.NewRecorder()
	h.Update(secondResponse, second)
	if secondResponse.Code != http.StatusConflict {
		t.Fatalf("古いフォームのステータスコード = %d、期待値 = %d", secondResponse.Code, http.StatusConflict)
	}
	body := secondResponse.Body.String()
	for _, want := range []string{
		"他の操作でアプリが更新されました",
		`name="version" value="2"`,
		`value="先に保存した名前"`,
		"https://example.com/first",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("競合応答に%qが含まれていない", want)
		}
	}

	// 送信の値は、保存されるフォームには戻さず、フォームの外の読み取り専用の欄にだけ示す
	formEnd := strings.Index(body, "</form>")
	if formEnd < 0 {
		t.Fatal("競合応答に編集フォームが無い")
	}
	form, rest := body[:formEnd], body[formEnd:]
	if strings.Contains(form, "古いフォームの名前") || strings.Contains(form, "https://example.com/second") {
		t.Error("編集フォームに古いフォームの値が入っている")
	}
	for _, want := range []string{
		"保存されなかった入力内容",
		`id="conflicted_name" readonly type="text" value="古いフォームの名前"`,
		"https://example.com/second",
	} {
		if !strings.Contains(rest, want) {
			t.Errorf("フォームの外に%qが含まれていない", want)
		}
	}
	if strings.Contains(rest, `name="name"`) || strings.Contains(rest, `name="redirect_uris"`) {
		t.Error("保存されなかった値の欄が送信される項目になっている")
	}

	app, err := repository.NewOAuthApplicationRepository(q).FindByIDAndSpaceID(context.Background(), appID, m.spaceID)
	if err != nil {
		t.Fatalf("FindByIDAndSpaceID()のエラー = %v", err)
	}
	if app.Name != "先に保存した名前" || !slices.Equal(app.RedirectURIs, []string{"https://example.com/first"}) || app.Version != 2 {
		t.Errorf("競合後のアプリ = %+v、期待値は先に保存した内容と版2", app)
	}
}

func TestUpdate_版を指定しない送信は400で更新しない(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "oauth-app-update-no-version"
	m := setupAppMember(t, tx, identifier, "編集するメンバー", []model.Scope{model.ScopeOAuthApplicationWrite}, true)
	appID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(m.spaceID).
		WithName("元の名前").
		WithClientID("oauth-app-update-no-version-client").
		Build()

	path := "/s/" + identifier + "/settings/oauth_applications/" + string(appID)
	req := newRequest(t, http.MethodPatch, path, identifier, m.userID, url.Values{
		"name":          {"編集したアプリ"},
		"redirect_uris": {"https://example.com/new"},
	})
	rr := httptest.NewRecorder()
	setupHandler(t, q).Update(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusBadRequest)
	}
	app, err := repository.NewOAuthApplicationRepository(q).FindByIDAndSpaceID(context.Background(), appID, m.spaceID)
	if err != nil {
		t.Fatalf("FindByIDAndSpaceID()のエラー = %v", err)
	}
	if app.Name != "元の名前" || app.Version != 1 {
		t.Errorf("版を指定しない送信の後のアプリ = %+v、期待値は変更前のアプリ", app)
	}
}
