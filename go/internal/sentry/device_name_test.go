package sentry

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
)

func TestInit_端末名をエラーとトランザクションに残さない(t *testing.T) {
	// Initのコールバック設定を確かめるため、グローバルHubを使うこのテストは並列にしない。
	previousClient := sentry.CurrentHub().Client()
	t.Cleanup(func() { sentry.CurrentHub().BindClient(previousClient) })
	if err := Init(Config{DSN: "https://public@example.com/1", TracesSampleRate: 1}); err != nil {
		t.Fatalf("Sentryの初期化に失敗: %v", err)
	}
	options := sentry.CurrentHub().Client().Options()

	const deviceName = "private-person-laptop"
	authorizeURL := "/oauth/authorize?" + url.Values{"device_name": {deviceName}}.Encode()
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
		{name: "デバイス認可のフォーム", method: http.MethodPost, target: "/oauth/device_authorization", form: url.Values{"device_name": {deviceName}}, wantDeviceName: deviceName},
		{name: "ログインフォームの戻り先", method: http.MethodPost, target: "/sign_in", form: url.Values{"back": {authorizeURL}}, wantBack: authorizeURL},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := &deviceNameCaptureTransport{}
			localOptions := options
			localOptions.Transport = transport
			client, err := sentry.NewClient(localOptions)
			if err != nil {
				t.Fatalf("Sentryクライアントの作成に失敗: %v", err)
			}
			hub := sentry.NewHub(client, sentry.NewScope())
			hub.Scope().SetTag("device_name", deviceName)
			req := httptest.NewRequest(tt.method, tt.target, strings.NewReader(tt.form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Referer", "https://example.com"+authorizeURL)
			req = req.WithContext(sentry.SetHubOnContext(req.Context(), hub))
			originalURI, originalURL := req.RequestURI, req.URL.String()
			handler := sentryhttp.New(sentryhttp.Options{}).Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := r.ParseForm(); err != nil {
					t.Fatalf("フォームの解析に失敗: %v", err)
				}
				if got := r.Form.Get("device_name"); got != tt.wantDeviceName {
					t.Errorf("device_name = %q、期待値 = %q", got, tt.wantDeviceName)
				}
				if got := r.Form.Get("back"); got != tt.wantBack {
					t.Errorf("back = %q、期待値 = %q", got, tt.wantBack)
				}
				sentry.GetHubFromContext(r.Context()).CaptureException(errors.New("テスト用のエラー"))
				w.WriteHeader(http.StatusNoContent)
			}))
			handler.ServeHTTP(httptest.NewRecorder(), req)

			transport.mu.Lock()
			defer transport.mu.Unlock()
			if len(transport.events) != 2 {
				t.Fatalf("イベント数 = %d、期待値 = 2 (エラーとトランザクション)", len(transport.events))
			}
			seen := map[string]bool{}
			for _, event := range transport.events {
				seen[event.Type] = true
				if event.Request == nil {
					t.Fatal("イベントにリクエスト情報が無い")
				}
				encoded, err := json.Marshal(event)
				if err != nil {
					t.Fatalf("イベントを読めない: %v", err)
				}
				if strings.Contains(string(encoded), deviceName) {
					t.Errorf("%qイベントに端末名が残っている: %s", event.Type, encoded)
				}
			}
			if !seen[""] || !seen["transaction"] {
				t.Errorf("イベント種別 = %v、期待値はエラーとトランザクション", seen)
			}
			if req.RequestURI != originalURI || req.URL.String() != originalURL || req.Header.Get("Referer") != "https://example.com"+authorizeURL {
				t.Error("送信情報のマスクで実際のリクエストを変更している")
			}
		})
	}
}

// deviceNameCaptureTransportはネットワーク送信の代わりにイベントを保持する。
type deviceNameCaptureTransport struct {
	mu     sync.Mutex
	events []*sentry.Event
}

func (t *deviceNameCaptureTransport) Configure(_ sentry.ClientOptions)        {}
func (t *deviceNameCaptureTransport) Flush(_ time.Duration) bool              { return true }
func (t *deviceNameCaptureTransport) FlushWithContext(_ context.Context) bool { return true }
func (t *deviceNameCaptureTransport) Close()                                  {}
func (t *deviceNameCaptureTransport) SendEventWithContext(_ context.Context, event *sentry.Event) {
	t.SendEvent(event)
}
func (t *deviceNameCaptureTransport) SendEvent(event *sentry.Event) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = append(t.events, event)
}
