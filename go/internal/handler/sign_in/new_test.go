package sign_in_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestNew(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	handler := setupHandler(t, tx, true)

	req := httptest.NewRequest(http.MethodGet, "/sign_in", nil)
	req.Header.Set("Accept-Language", "ja")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	if !strings.Contains(body, `action="/sign_in"`) {
		t.Error("login form action not found in response")
	}

	if !strings.Contains(body, "test-csrf-token") {
		t.Error("CSRF token not found in response")
	}

	if !strings.Contains(body, `name="email"`) {
		t.Error("email input field not found in response")
	}

	if !strings.Contains(body, `name="password"`) {
		t.Error("password input field not found in response")
	}

	if !strings.Contains(body, "test-site-key") {
		t.Error("Turnstile site key not found in response")
	}

	if !strings.Contains(body, `name="back"`) {
		t.Error("back hidden field not found in response")
	}
}

func TestNew_WithBackParameter(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	handler := setupHandler(t, tx, true)

	tests := []struct {
		name       string
		backURL    string
		wantInBody string
	}{
		{
			name:       "backパラメータあり",
			backURL:    "/oauth/authorize?client_id=xxx",
			wantInBody: `name="back" value="/oauth/authorize?client_id=xxx"`,
		},
		{
			name:       "backパラメータなし",
			backURL:    "",
			wantInBody: `name="back" value=""`,
		},
		{
			name:       "日本語パスのbackパラメータ",
			backURL:    "/users/テスト",
			wantInBody: `name="back" value="/users/テスト"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			targetURL := "/sign_in"
			if tt.backURL != "" {
				targetURL = "/sign_in?back=" + tt.backURL
			}
			req := httptest.NewRequest(http.MethodGet, targetURL, nil)
			req.Header.Set("Accept-Language", "ja")

			ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.New(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
			}

			body := rr.Body.String()
			if !strings.Contains(body, tt.wantInBody) {
				t.Errorf("backパラメータがhiddenフィールドに含まれていません\nwant: %s", tt.wantInBody)
			}
		})
	}
}
