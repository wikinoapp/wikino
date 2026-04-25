package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/i18n"
)

func TestNotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		locale       string
		wantStatus   int
		wantContents []string
	}{
		{
			name:       "日本語で404ページが表示される",
			locale:     "ja",
			wantStatus: http.StatusNotFound,
			wantContents: []string{
				"お探しのページは見つかりませんでした",
				"トップページに戻る",
				`href="/"`,
			},
		},
		{
			name:       "英語で404ページが表示される",
			locale:     "en",
			wantStatus: http.StatusNotFound,
			wantContents: []string{
				"The page you are looking for could not be found",
				"Back to top page",
				`href="/"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
			ctx := i18n.SetLocale(req.Context(), tt.locale)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.NotFound(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("ステータスコード: got %d, want %d", rr.Code, tt.wantStatus)
			}

			contentType := rr.Header().Get("Content-Type")
			if !strings.Contains(contentType, "text/html") {
				t.Errorf("Content-Type: got %q, want text/html", contentType)
			}

			body := rr.Body.String()
			for _, want := range tt.wantContents {
				if !strings.Contains(body, want) {
					t.Errorf("レスポンスに %q が含まれていません", want)
				}
			}
		})
	}
}
