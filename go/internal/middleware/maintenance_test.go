package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
)

func TestMaintenanceMiddleware_Disabled(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{MaintenanceMode: false}
	mw := NewMaintenanceMiddleware(cfg)

	handler := mw.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("メンテナンスモード無効時にステータスが期待と異なる: got %d want %d", rr.Code, http.StatusOK)
	}
}

func TestMaintenanceMiddleware_Enabled_GeneralIP(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		MaintenanceMode: true,
		AdminIPs:        []string{"192.168.1.100"},
	}
	mw := NewMaintenanceMiddleware(cfg)

	handler := mw.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("一般IPのステータスが期待と異なる: got %d want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

func TestMaintenanceMiddleware_Enabled_AdminIP(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		MaintenanceMode: true,
		AdminIPs:        []string{"192.168.1.100"},
	}
	mw := NewMaintenanceMiddleware(cfg)

	handler := mw.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("管理者IPのステータスが期待と異なる: got %d want %d", rr.Code, http.StatusOK)
	}
}

func TestMaintenanceMiddleware_Enabled_HealthCheck(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		MaintenanceMode: true,
		AdminIPs:        []string{},
	}
	mw := NewMaintenanceMiddleware(cfg)

	handler := mw.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ヘルスチェックのステータスが期待と異なる: got %d want %d", rr.Code, http.StatusOK)
	}
}

func TestMaintenanceMiddleware_MultipleAdminIPs(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		MaintenanceMode: true,
		AdminIPs:        []string{"192.168.1.100", "10.0.0.50", "172.16.0.1"},
	}
	mw := NewMaintenanceMiddleware(cfg)

	handler := mw.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	testCases := []struct {
		name       string
		remoteAddr string
		wantStatus int
	}{
		{
			name:       "最初の管理者IP",
			remoteAddr: "192.168.1.100:12345",
			wantStatus: http.StatusOK,
		},
		{
			name:       "2番目の管理者IP",
			remoteAddr: "10.0.0.50:12345",
			wantStatus: http.StatusOK,
		},
		{
			name:       "3番目の管理者IP",
			remoteAddr: "172.16.0.1:12345",
			wantStatus: http.StatusOK,
		},
		{
			name:       "管理者IP以外",
			remoteAddr: "10.0.0.99:12345",
			wantStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tc.remoteAddr
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("ステータスが期待と異なる: got %d want %d", rr.Code, tc.wantStatus)
			}
		})
	}
}

func TestMaintenanceMiddleware_XForwardedFor(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		MaintenanceMode: true,
		AdminIPs:        []string{"203.0.113.50"},
	}
	mw := NewMaintenanceMiddleware(cfg)

	handler := mw.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.50, 198.51.100.1")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("X-Forwarded-For経由の管理者IPのステータスが期待と異なる: got %d want %d", rr.Code, http.StatusOK)
	}
}

func TestMaintenanceMiddleware_CFConnectingIP(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		MaintenanceMode: true,
		AdminIPs:        []string{"203.0.113.50"},
	}
	mw := NewMaintenanceMiddleware(cfg)

	handler := mw.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("CF-Connecting-IP", "203.0.113.50")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("CF-Connecting-IP経由の管理者IPのステータスが期待と異なる: got %d want %d", rr.Code, http.StatusOK)
	}
}

func TestMaintenanceMiddleware_XRealIP(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		MaintenanceMode: true,
		AdminIPs:        []string{"203.0.113.50"},
	}
	mw := NewMaintenanceMiddleware(cfg)

	handler := mw.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Real-IP", "203.0.113.50")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("X-Real-IP経由の管理者IPのステータスが期待と異なる: got %d want %d", rr.Code, http.StatusOK)
	}
}

func TestMaintenanceMiddleware_NoAdminIPs(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		MaintenanceMode: true,
		AdminIPs:        []string{},
	}
	mw := NewMaintenanceMiddleware(cfg)

	handler := mw.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("管理者IP未設定時のステータスが期待と異なる: got %d want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

func TestMaintenanceMiddleware_ResponseHeaders(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		MaintenanceMode: true,
		AdminIPs:        []string{},
	}
	mw := NewMaintenanceMiddleware(cfg)

	handler := mw.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	contentType := rr.Header().Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("Content-Typeが期待と異なる: got %q want %q", contentType, "text/html; charset=utf-8")
	}

	retryAfter := rr.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Error("Retry-Afterヘッダーが設定されていない")
	}
}

func TestMaintenanceMiddleware_PageContent(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		MaintenanceMode: true,
		AdminIPs:        []string{},
	}
	mw := NewMaintenanceMiddleware(cfg)

	handler := mw.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	body := rr.Body.String()

	expectedContents := []string{
		"Wikino",
		"メンテナンス中",
		"<!doctype html>",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(body, expected) {
			t.Errorf("レスポンスボディに %q が含まれていない", expected)
		}
	}
}
