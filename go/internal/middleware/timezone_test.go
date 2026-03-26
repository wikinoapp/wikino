package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/timezone"
)

func TestTimeZone_ログインユーザーのタイムゾーンを優先(t *testing.T) {
	t.Parallel()

	var tzFromCtx string
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tzFromCtx = timezone.FromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// クッキーも設定して、ユーザー設定が優先されることを確認
	req.AddCookie(&http.Cookie{
		Name:  timeZoneCookieName,
		Value: "America/New_York",
	})
	ctx := SetUserToContext(req.Context(), &model.User{
		TimeZone: "Asia/Tokyo",
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	TimeZone(nextHandler).ServeHTTP(rr, req)

	if tzFromCtx != "Asia/Tokyo" {
		t.Errorf("タイムゾーンが不正: got %q, want %q", tzFromCtx, "Asia/Tokyo")
	}
}

func TestTimeZone_クッキーからタイムゾーンを取得(t *testing.T) {
	t.Parallel()

	var tzFromCtx string
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tzFromCtx = timezone.FromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  timeZoneCookieName,
		Value: "America/New_York",
	})
	rr := httptest.NewRecorder()

	TimeZone(nextHandler).ServeHTTP(rr, req)

	if tzFromCtx != "America/New_York" {
		t.Errorf("タイムゾーンが不正: got %q, want %q", tzFromCtx, "America/New_York")
	}
}

func TestTimeZone_不正なクッキー値の場合はUTCにフォールバック(t *testing.T) {
	t.Parallel()

	var tzFromCtx string
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tzFromCtx = timezone.FromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  timeZoneCookieName,
		Value: "Invalid/Timezone",
	})
	rr := httptest.NewRecorder()

	TimeZone(nextHandler).ServeHTTP(rr, req)

	if tzFromCtx != "UTC" {
		t.Errorf("タイムゾーンが不正: got %q, want %q", tzFromCtx, "UTC")
	}
}

func TestTimeZone_ユーザーもクッキーもない場合はUTC(t *testing.T) {
	t.Parallel()

	var tzFromCtx string
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tzFromCtx = timezone.FromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	TimeZone(nextHandler).ServeHTTP(rr, req)

	if tzFromCtx != "UTC" {
		t.Errorf("タイムゾーンが不正: got %q, want %q", tzFromCtx, "UTC")
	}
}

func TestTimeZone_ユーザーのTimeZoneが空の場合はクッキーを使用(t *testing.T) {
	t.Parallel()

	var tzFromCtx string
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tzFromCtx = timezone.FromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  timeZoneCookieName,
		Value: "Europe/London",
	})
	ctx := SetUserToContext(req.Context(), &model.User{
		TimeZone: "",
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	TimeZone(nextHandler).ServeHTTP(rr, req)

	if tzFromCtx != "Europe/London" {
		t.Errorf("タイムゾーンが不正: got %q, want %q", tzFromCtx, "Europe/London")
	}
}

func TestFromContext_設定されていない場合はUTC(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tz := timezone.FromContext(ctx)

	if tz != "UTC" {
		t.Errorf("タイムゾーンが不正: got %q, want %q", tz, "UTC")
	}
}
