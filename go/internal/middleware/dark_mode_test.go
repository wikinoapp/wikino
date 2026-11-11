package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/darkmode"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// stubDarkModeCheckerは、受け取った引数を記録して決めた結果を返すフラグの判定
type stubDarkModeChecker struct {
	enabled bool
	err     error

	calls        int
	deviceToken  string
	sessionToken string
	name         model.FeatureFlagName
}

func (s *stubDarkModeChecker) IsEnabledForDevice(_ context.Context, deviceToken string, sessionToken string, name model.FeatureFlagName) (bool, error) {
	s.calls++
	s.deviceToken = deviceToken
	s.sessionToken = sessionToken
	s.name = name
	return s.enabled, s.err
}

// serveDarkModeは、ミドルウェアを通したリクエストの中で判定をcallsの回数だけ呼び、最後の結果を返す
func serveDarkMode(t *testing.T, checker *stubDarkModeChecker, cookies []*http.Cookie, calls int) bool {
	t.Helper()

	var got bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for range calls {
			got = darkmode.IsEnabled(r.Context())
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	middleware.DarkMode(checker)(next).ServeHTTP(httptest.NewRecorder(), req)

	return got
}

func TestDarkMode(t *testing.T) {
	t.Parallel()

	deviceCookie := &http.Cookie{Name: middleware.DeviceTokenCookieName, Value: "device-1"}
	sessionCookie := &http.Cookie{Name: session.CookieName, Value: "session-1"}

	tests := []struct {
		name             string
		cookies          []*http.Cookie
		enabled          bool
		err              error
		want             bool
		wantCalls        int
		wantDeviceToken  string
		wantSessionToken string
	}{
		{
			name:      "Cookieが無いときは判定せずに無効とする",
			cookies:   nil,
			enabled:   true,
			want:      false,
			wantCalls: 0,
		},
		{
			name:            "device_tokenでフラグが有効なら有効とする",
			cookies:         []*http.Cookie{deviceCookie},
			enabled:         true,
			want:            true,
			wantCalls:       1,
			wantDeviceToken: "device-1",
		},
		{
			name:             "ログインセッションでフラグが有効なら有効とする",
			cookies:          []*http.Cookie{sessionCookie},
			enabled:          true,
			want:             true,
			wantCalls:        1,
			wantSessionToken: "session-1",
		},
		{
			name:             "両方のCookieを判定に渡す",
			cookies:          []*http.Cookie{deviceCookie, sessionCookie},
			enabled:          false,
			want:             false,
			wantCalls:        1,
			wantDeviceToken:  "device-1",
			wantSessionToken: "session-1",
		},
		{
			name:            "判定でエラーが起きたときは無効とする",
			cookies:         []*http.Cookie{deviceCookie},
			enabled:         true,
			err:             errors.New("接続エラー"),
			want:            false,
			wantCalls:       1,
			wantDeviceToken: "device-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			checker := &stubDarkModeChecker{enabled: tt.enabled, err: tt.err}
			got := serveDarkMode(t, checker, tt.cookies, 1)

			if got != tt.want {
				t.Errorf("IsEnabled() = %v、期待値 = %v", got, tt.want)
			}
			if checker.calls != tt.wantCalls {
				t.Errorf("判定の回数 = %d、期待値 = %d", checker.calls, tt.wantCalls)
			}
			if tt.wantCalls == 0 {
				return
			}
			if checker.deviceToken != tt.wantDeviceToken {
				t.Errorf("deviceToken = %q、期待値 = %q", checker.deviceToken, tt.wantDeviceToken)
			}
			if checker.sessionToken != tt.wantSessionToken {
				t.Errorf("sessionToken = %q、期待値 = %q", checker.sessionToken, tt.wantSessionToken)
			}
			if checker.name != model.FeatureFlagDarkMode {
				t.Errorf("フラグ名 = %q、期待値 = %q", checker.name, model.FeatureFlagDarkMode)
			}
		})
	}
}

// 判定はクエリを伴うため、1つのリクエストで何度読まれても1回だけ行う
func TestDarkMode_JudgesOncePerRequest(t *testing.T) {
	t.Parallel()

	checker := &stubDarkModeChecker{enabled: true}
	cookies := []*http.Cookie{{Name: middleware.DeviceTokenCookieName, Value: "device-1"}}

	if got := serveDarkMode(t, checker, cookies, 3); !got {
		t.Errorf("IsEnabled() = %v、期待値 = true", got)
	}
	if checker.calls != 1 {
		t.Errorf("判定の回数 = %d、期待値 = 1", checker.calls)
	}
}

// htmxのフラグメントやAPIのように判定を読まないリクエストでは、クエリを発行しない
func TestDarkMode_DoesNotJudgeUntilRead(t *testing.T) {
	t.Parallel()

	checker := &stubDarkModeChecker{enabled: true}
	cookies := []*http.Cookie{{Name: middleware.DeviceTokenCookieName, Value: "device-1"}}

	serveDarkMode(t, checker, cookies, 0)

	if checker.calls != 0 {
		t.Errorf("判定の回数 = %d、期待値 = 0", checker.calls)
	}
}

// APIリファレンスは専用の配色を持つため、PageMetaを作ってもダークモードのフラグは判定しない
func TestDarkMode_SkipsAPIReference(t *testing.T) {
	t.Parallel()

	for _, method := range []string{http.MethodGet, http.MethodHead} {
		t.Run(method+"リクエスト", func(t *testing.T) {
			t.Parallel()

			checker := &stubDarkModeChecker{enabled: true}
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				meta := viewmodel.DefaultPageMeta(r.Context(), &config.Config{Domain: "example.com"})
				if meta.DarkModeEnabled {
					t.Error("APIリファレンスのダークモードは無効を期待")
				}
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(method, model.APIReferencePath, nil)
			req.AddCookie(&http.Cookie{Name: middleware.DeviceTokenCookieName, Value: "device-1"})
			req.AddCookie(&http.Cookie{Name: session.CookieName, Value: "session-1"})
			rr := httptest.NewRecorder()
			middleware.DarkMode(checker)(next).ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
			}
			if checker.calls != 0 {
				t.Errorf("判定の回数 = %d、期待値 = 0", checker.calls)
			}
		})
	}
}
