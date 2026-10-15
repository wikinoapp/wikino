package oauth_device_verification_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestNew_ユーザーコードの入力画面を表示する(t *testing.T) {
	t.Parallel()

	f := setupDeviceFixture(t, "devverify-new", model.SpaceRoleAdmin, true)

	rr := httptest.NewRecorder()
	setupHandler(t).New(rr, newRequest(t, http.MethodGet, f.userID, "198.51.100.1", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータス = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	assertResponseHeaders(t, rr)
	assertBodyContains(t, rr,
		"端末に表示されたコードを入力してください",
		`action="/oauth/device"`,
		`name="csrf_token"`,
		`name="user_code"`,
		`autocomplete="off"`,
	)
}

func TestNew_verification_uri_completeで開くとコードを入力欄に入れておく(t *testing.T) {
	t.Parallel()

	f := setupDeviceFixture(t, "devverify-new-complete", model.SpaceRoleAdmin, true)

	rr := httptest.NewRecorder()
	setupHandler(t).New(rr, newRequest(t, http.MethodGet, f.userID, "198.51.100.2", url.Values{"user_code": {"BCDF-GHJK"}}))

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータス = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	assertBodyContains(t, rr, `value="BCDF-GHJK"`)
}

func TestNew_未ログインならサインインへ送りサインイン後にこの画面へ戻す(t *testing.T) {
	t.Parallel()

	req := newRequest(t, http.MethodGet, "", "198.51.100.3", url.Values{"user_code": {"BCDF-GHJK"}})
	rr := httptest.NewRecorder()
	setupHandler(t).New(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("ステータス = %d、期待値 = %d", rr.Code, http.StatusFound)
	}
	assertResponseHeaders(t, rr)
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

func TestNew_フィーチャーフラグが無効なら404を返す(t *testing.T) {
	t.Parallel()

	f := setupDeviceFixture(t, "devverify-new-flag", model.SpaceRoleAdmin, false)

	rr := httptest.NewRecorder()
	setupHandler(t).New(rr, newRequest(t, http.MethodGet, f.userID, "198.51.100.4", nil))

	if rr.Code != http.StatusNotFound {
		t.Fatalf("ステータス = %d、期待値 = %d", rr.Code, http.StatusNotFound)
	}
	assertResponseHeaders(t, rr)
}
