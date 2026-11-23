package apple_touch_icon_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/handler/apple_touch_icon"
)

func TestShow(t *testing.T) {
	t.Parallel()

	want, err := os.ReadFile("../../../static/images/apple-touch-icon.png")
	if err != nil {
		t.Fatalf("Apple Touch Iconの読み込みに失敗: %v", err)
	}
	h := apple_touch_icon.NewHandler(os.DirFS("../../../static"))

	for _, path := range []string{"/apple-touch-icon.png", "/apple-touch-icon-precomposed.png"} {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			t.Run(method+" "+path, func(t *testing.T) {
				t.Parallel()

				req := httptest.NewRequest(method, path, nil)
				rr := httptest.NewRecorder()
				h.Show(rr, req)

				if rr.Code != http.StatusOK {
					t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
				}
				if got := rr.Header().Get("Content-Type"); got != "image/png" {
					t.Errorf("Content-Type = %q、期待値 = %q", got, "image/png")
				}
				if method == http.MethodHead {
					if rr.Body.Len() != 0 {
						t.Errorf("HEADの本文の長さ = %d、期待値 = 0", rr.Body.Len())
					}
					return
				}
				if !bytes.Equal(rr.Body.Bytes(), want) {
					t.Error("本文が `static/images/apple-touch-icon.png` と一致しない")
				}
			})
		}
	}
}
