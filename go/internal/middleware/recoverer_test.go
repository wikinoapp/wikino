package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/apierror"
	"github.com/wikinoapp/wikino/go/internal/middleware"
)

func TestRecoverer(t *testing.T) {
	t.Parallel()

	recoverer := middleware.NewRecoverer(apierror.NewWriter("https://example.com"))
	panicking := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("テスト用のパニック")
	})

	t.Run("APIのパスには500をProblem Detailsで返す", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.yaml", nil)
		rr := httptest.NewRecorder()
		recoverer.Middleware(panicking).ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("ステータス = %d、期待値 = %d", rr.Code, http.StatusInternalServerError)
		}
		if got := rr.Header().Get("Content-Type"); got != apierror.ContentType {
			t.Errorf("Content-Type = %q、期待値 = %q", got, apierror.ContentType)
		}
		if !strings.Contains(rr.Body.String(), `"status":500`) {
			t.Errorf("本文がProblem Detailsではない: %s", rr.Body.String())
		}
	})

	t.Run("APIのパス以外はchiのRecovererと同じく本文の無い500を返す", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/s/example", nil)
		rr := httptest.NewRecorder()
		recoverer.Middleware(panicking).ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("ステータス = %d、期待値 = %d", rr.Code, http.StatusInternalServerError)
		}
		if rr.Body.Len() != 0 {
			t.Errorf("本文 = %q、期待値は空", rr.Body.String())
		}
	})

	t.Run("APIのパスでもhttp.ErrAbortHandlerは投げ直す", func(t *testing.T) {
		t.Parallel()

		aborting := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic(http.ErrAbortHandler)
		})
		req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.yaml", nil)
		rr := httptest.NewRecorder()

		defer func() {
			if rvr := recover(); rvr != http.ErrAbortHandler {
				t.Errorf("recover() = %v、期待値 = http.ErrAbortHandler", rvr)
			}
		}()
		recoverer.Middleware(aborting).ServeHTTP(rr, req)
	})

	t.Run("パニックしなければそのまま返す", func(t *testing.T) {
		t.Parallel()

		ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
		req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.yaml", nil)
		rr := httptest.NewRecorder()
		recoverer.Middleware(ok).ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Errorf("ステータス = %d、期待値 = %d", rr.Code, http.StatusNoContent)
		}
	})
}
