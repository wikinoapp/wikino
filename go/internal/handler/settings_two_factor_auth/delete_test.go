package settings_two_factor_auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// newDeleteRequestは、Method Overrideを通ったあとの無効にするリクエストを組み立てる。
// 本番ではPOSTの本文をMethod Overrideが読んでからDELETEに書き換えるため、同じ順で組み立てる。
func newDeleteRequest(t *testing.T, user *model.User, form url.Values) *http.Request {
	t.Helper()

	req := newRequest(t, http.MethodPost, "/settings/two_factor_auth", user, form)
	if err := req.ParseForm(); err != nil {
		t.Fatalf("フォームのパースに失敗: %v", err)
	}
	req.Method = http.MethodDelete
	return req
}

func TestDelete_無効にして設定の画面へ送る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	user := newUser(t, tx, "settings_2fa_delete", testSecret, true)
	setup := setupHandler(t, tx)

	rr := httptest.NewRecorder()
	setup.handler.Delete(rr, newDeleteRequest(t, user, url.Values{"password": {testPassword}}))

	if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/settings/two_factor_auth" {
		t.Fatalf("ステータスコード = %d、Location = %q、設定の画面へのリダイレクトを期待", rr.Code, rr.Header().Get("Location"))
	}
	current, err := setup.repo.FindByUserID(context.Background(), user.ID)
	if err != nil || current == nil || current.Enabled || len(current.RecoveryCodes) != 0 {
		t.Errorf("設定 = %+v、無効でリカバリーコードの無い設定を期待 (エラー = %v)", current, err)
	}
}

func TestDelete_パスワードを誤ったら有効なまま描き直す(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	user := newUser(t, tx, "settings_2fa_delete_ng", testSecret, true)
	setup := setupHandler(t, tx)

	rr := httptest.NewRecorder()
	setup.handler.Delete(rr, newDeleteRequest(t, user, url.Values{"password": {"wrong-password"}}))

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusUnprocessableEntity)
	}
	assertSettingsHeaders(t, rr)
	body := rr.Body.String()
	for _, want := range []string{"パスワードが正しくありません", `aria-invalid="true"`, "二要素認証は有効です"} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに %q が含まれていない", want)
		}
	}
	current, err := setup.repo.FindByUserID(context.Background(), user.ID)
	if err != nil || current == nil || !current.Enabled {
		t.Errorf("設定 = %+v、有効なままを期待 (エラー = %v)", current, err)
	}
}

func TestDelete_有効でなければ設定の画面へ送る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	user := newUser(t, tx, "settings_2fa_delete_off", "", false)

	rr := httptest.NewRecorder()
	setupHandler(t, tx).handler.Delete(rr, newDeleteRequest(t, user, url.Values{"password": {testPassword}}))

	if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/settings/two_factor_auth" {
		t.Errorf("ステータスコード = %d、Location = %q、設定の画面へのリダイレクトを期待", rr.Code, rr.Header().Get("Location"))
	}
}

func TestDelete_試行が多すぎたら照合せずに描き直す(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	user := newUser(t, tx, "settings_2fa_delete_rl", testSecret, true)
	setup := setupHandler(t, tx)

	for i := range 5 {
		rr := httptest.NewRecorder()
		setup.handler.Delete(rr, newDeleteRequest(t, user, url.Values{"password": {"wrong-password"}}))
		if rr.Code != http.StatusUnprocessableEntity {
			t.Fatalf("%d回目のステータスコード = %d、期待値 = %d", i+1, rr.Code, http.StatusUnprocessableEntity)
		}
	}

	// 上限を超えたら、正しいパスワードでも無効にしない
	rr := httptest.NewRecorder()
	setup.handler.Delete(rr, newDeleteRequest(t, user, url.Values{"password": {testPassword}}))
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusUnprocessableEntity)
	}
	if !strings.Contains(rr.Body.String(), "リクエストが多すぎます") {
		t.Error("レート制限のエラーが表示されていない")
	}
	current, err := setup.repo.FindByUserID(context.Background(), user.ID)
	if err != nil || current == nil || !current.Enabled {
		t.Errorf("設定 = %+v、有効なままを期待 (エラー = %v)", current, err)
	}
}

func TestDelete_ログインしていなければログインの画面へ送る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	rr := httptest.NewRecorder()
	setupHandler(t, tx).handler.Delete(rr, newDeleteRequest(t, nil, url.Values{"password": {testPassword}}))

	if rr.Code != http.StatusFound || rr.Header().Get("Location") != "/sign_in" {
		t.Errorf("ステータスコード = %d、Location = %q、/sign_inへのリダイレクトを期待", rr.Code, rr.Header().Get("Location"))
	}
}
