package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestNewAccessLog_端末名をログに残さずハンドラーに渡す(t *testing.T) {
	t.Parallel()

	const deviceName = "private-person-laptop"
	authorizeURL := "/oauth/authorize?" + url.Values{"device_name": {deviceName}, "client_id": {"test-client"}}.Encode()
	tests := []struct {
		name           string
		method         string
		target         string
		form           url.Values
		wantDeviceName string
		wantBack       string
	}{
		{name: "認可要求のクエリ", method: http.MethodGet, target: authorizeURL, wantDeviceName: deviceName},
		{name: "ログインの戻り先のクエリ", method: http.MethodGet, target: "/sign_in?" + url.Values{"back": {authorizeURL}}.Encode(), wantBack: authorizeURL},
		{name: "認可フォームの本文", method: http.MethodPost, target: "/oauth/authorize", form: url.Values{"device_name": {deviceName}}, wantDeviceName: deviceName},
		{name: "ログインフォームの戻り先", method: http.MethodPost, target: "/sign_in", form: url.Values{"back": {authorizeURL}}, wantBack: authorizeURL},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var logs bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&logs, nil))
			req := httptest.NewRequest(tt.method, tt.target, strings.NewReader(tt.form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			originalURI, originalURL := req.RequestURI, req.URL.String()
			handler := NewAccessLog(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := r.ParseForm(); err != nil {
					t.Fatalf("フォームの解析に失敗: %v", err)
				}
				if got := r.Form.Get("device_name"); got != tt.wantDeviceName {
					t.Errorf("device_name = %q、期待値 = %q", got, tt.wantDeviceName)
				}
				if got := r.Form.Get("back"); got != tt.wantBack {
					t.Errorf("back = %q、期待値 = %q", got, tt.wantBack)
				}
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte("ok"))
			}))
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if strings.Contains(logs.String(), deviceName) {
				t.Errorf("アクセスログに端末名が残っている: %s", logs.String())
			}
			if req.RequestURI != originalURI || req.URL.String() != originalURL {
				t.Error("ログのために実際のリクエストURLを変更している")
			}
			var entry map[string]any
			if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
				t.Fatalf("アクセスログを読めない: %v", err)
			}
			for key, want := range map[string]any{"method": tt.method, "path": req.URL.EscapedPath(), "status": float64(http.StatusCreated), "bytes": float64(2)} {
				if got := entry[key]; got != want {
					t.Errorf("ログの%s = %v、期待値 = %v", key, got, want)
				}
			}
			if rr.Code != http.StatusCreated || rr.Body.String() != "ok" {
				t.Errorf("応答 = %d %q、期待値 = 201 ok", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestNewAccessLog_本文を書かない成功応答を200と記録する(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	handler := NewAccessLog(slog.New(slog.NewJSONHandler(&logs, nil)))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/health", nil))
	var entry map[string]any
	if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
		t.Fatalf("アクセスログを読めない: %v", err)
	}
	if entry["status"] != float64(http.StatusOK) {
		t.Errorf("ログのstatus = %v、期待値 = 200", entry["status"])
	}
}
