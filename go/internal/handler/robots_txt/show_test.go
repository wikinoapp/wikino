package robots_txt_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/handler/robots_txt"
)

func TestShow(t *testing.T) {
	t.Parallel()

	h := robots_txt.NewHandler()

	tests := []struct {
		method   string
		wantBody string
	}{
		{method: http.MethodGet, wantBody: "User-agent: *\nDisallow:\n"},
		{method: http.MethodHead, wantBody: ""},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(tt.method, "/robots.txt", nil)
			rr := httptest.NewRecorder()
			h.Show(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
			}
			if got := rr.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
				t.Errorf("Content-Type = %q、期待値 = %q", got, "text/plain; charset=utf-8")
			}
			if got := rr.Body.String(); got != tt.wantBody {
				t.Errorf("本文 = %q、期待値 = %q", got, tt.wantBody)
			}
		})
	}
}
