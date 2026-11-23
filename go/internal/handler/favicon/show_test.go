package favicon_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/handler/favicon"
)

func TestShow(t *testing.T) {
	t.Parallel()

	want, err := os.ReadFile("../../../static/images/favicon.ico")
	if err != nil {
		t.Fatalf("ファビコンの読み込みに失敗: %v", err)
	}
	h := favicon.NewHandler(os.DirFS("../../../static"))

	for _, method := range []string{http.MethodGet, http.MethodHead} {
		t.Run(method, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(method, "/favicon.ico", nil)
			rr := httptest.NewRecorder()
			h.Show(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
			}
			if got := rr.Header().Get("Content-Type"); got != "image/vnd.microsoft.icon" {
				t.Errorf("Content-Type = %q、期待値 = %q", got, "image/vnd.microsoft.icon")
			}
			if method == http.MethodHead {
				if rr.Body.Len() != 0 {
					t.Errorf("HEADの本文の長さ = %d、期待値 = 0", rr.Body.Len())
				}
				return
			}
			if !bytes.Equal(rr.Body.Bytes(), want) {
				t.Error("本文が `static/images/favicon.ico` と一致しない")
			}
		})
	}
}
