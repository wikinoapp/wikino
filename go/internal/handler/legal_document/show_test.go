package legal_document_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/handler/legal_document"
)

func TestShow_固定URLを本文のページへ301で転送する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path         string
		wantLocation string
	}{
		{path: "/terms", wantLocation: "/s/wikino/pages/41"},
		{path: "/terms/api", wantLocation: "/s/wikino/pages/323"},
		{path: "/privacy", wantLocation: "/s/wikino/pages/42"},
		{path: "/en/terms", wantLocation: "/s/wikino/pages/325"},
		{path: "/en/terms/api", wantLocation: "/s/wikino/pages/326"},
		{path: "/en/privacy", wantLocation: "/s/wikino/pages/328"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()

			// 問い合わせ文字列は転送先に引き継がない
			req := httptest.NewRequest(http.MethodGet, tt.path+"?utm_source=test", nil)
			rr := httptest.NewRecorder()
			legal_document.NewHandler().Show(rr, req)

			if rr.Code != http.StatusMovedPermanently {
				t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusMovedPermanently)
			}
			if location := rr.Header().Get("Location"); location != tt.wantLocation {
				t.Errorf("Location = %q、期待値 = %q", location, tt.wantLocation)
			}
		})
	}
}
