package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/apierror"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
)

func TestAPIResponseHeader(t *testing.T) {
	t.Parallel()

	apiHeaders := map[string]string{
		"X-Robots-Tag":            "noindex",
		"Content-Security-Policy": "frame-ancestors 'none'",
		"X-Frame-Options":         "DENY",
		"Cache-Control":           "no-store",
		"X-Content-Type-Options":  "nosniff",
	}

	tests := []struct {
		name    string
		path    string
		wantAPI bool
	}{
		{name: "APIのパス", path: "/api/v1/openapi.yaml", wantAPI: true},
		{name: "ルートの無いAPIのパス", path: "/api/v2/unknown", wantAPI: true},
		{name: "Web画面", path: "/s/example/pages/1", wantAPI: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rr := httptest.NewRecorder()
			middleware.APIResponseHeader(next).ServeHTTP(rr, req)

			for key, want := range apiHeaders {
				got := rr.Header().Get(key)
				if !tt.wantAPI {
					want = ""
				}
				if got != want {
					t.Errorf("%s = %q、期待値 = %q", key, got, want)
				}
			}
		})
	}
}

func TestAPIResponseHeader_OAuthのレート制限の429にもnoStoreを付ける(t *testing.T) {
	t.Parallel()

	for _, path := range []string{"/oauth/token", "/oauth/revoke"} {
		for _, allowed := range []bool{true, false} {
			limiter := &stubAPIRateLimiter{result: &ratelimit.CheckResult{
				Allowed: allowed, Remaining: 0, ResetAt: time.Now().Add(time.Minute),
			}}
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			limited := middleware.NewAPIRateLimit(limiter, apierror.NewWriter("https://example.com"), middleware.OAuthIPRateLimitPolicy(nil)).Middleware(next)
			rr := httptest.NewRecorder()
			middleware.APIResponseHeader(limited).ServeHTTP(rr, httptest.NewRequest(http.MethodPost, path, nil))

			wantStatus := http.StatusOK
			if !allowed {
				wantStatus = http.StatusTooManyRequests
			}
			if rr.Code != wantStatus {
				t.Errorf("%s allowed=%v: status = %d、期待値 = %d", path, allowed, rr.Code, wantStatus)
			}
			if got := rr.Header().Get("Cache-Control"); got != "no-store" {
				t.Errorf("%s allowed=%v: Cache-Control = %q、期待値 = no-store", path, allowed, got)
			}
		}
	}
}

// ハンドラーは既定のCache-Controlを上書きできる
func TestAPIResponseHeader_HandlerCanOverride(t *testing.T) {
	t.Parallel()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=300")
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.yaml", nil)
	rr := httptest.NewRecorder()
	middleware.APIResponseHeader(next).ServeHTTP(rr, req)

	if got := rr.Header().Get("Cache-Control"); got != "public, max-age=300" {
		t.Errorf("Cache-Control = %q、期待値 = %q", got, "public, max-age=300")
	}
}
