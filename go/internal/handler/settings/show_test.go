package settings_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/settings"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
)

func newSettingsRequest(t *testing.T, user *model.User) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	if user != nil {
		ctx = middleware.SetUserToContext(ctx, user)
	}
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	return req.WithContext(ctx)
}

func TestShow_各設定へのリンクとログアウトのボタンを表示する(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	settings.NewHandler(&config.Config{Env: "test", Domain: "localhost"}).
		Show(rr, newSettingsRequest(t, &model.User{ID: "settings-user", Atname: "settings_user"}))

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("X-Robots-Tag"); got != "noindex" {
		t.Errorf("X-Robots-Tag = %q、期待値 = %q", got, "noindex")
	}

	body := rr.Body.String()
	for _, want := range []string{
		"<title>設定 | Wikino</title>",
		`href="/settings/profile"`,
		`href="/settings/email"`,
		`href="/settings/two_factor_auth"`,
		`href="/settings/account/deletion/new"`,
		`action="/user_session"`,
		`name="_method" value="DELETE"`,
		`data-confirm="ログアウトしますか？"`,
		`name="csrf_token" value="test-csrf-token"`,
		"ログアウト",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに %q が含まれていない", want)
		}
	}
}

func TestShow_未ログインならログインへリダイレクトする(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	settings.NewHandler(&config.Config{Env: "test", Domain: "localhost"}).Show(rr, newSettingsRequest(t, nil))

	if rr.Code != http.StatusFound {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != "/sign_in" {
		t.Errorf("Location = %q、期待値 = %q", got, "/sign_in")
	}
}
