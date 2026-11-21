package errors_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/templates/pages/errors"
)

// 単独のエラーページは共通のheadとスタイルシートを読まないため、ページの中でOSの配色に合わせる
func TestStandalonePages_ColorScheme(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		component templ.Component
	}{
		{name: "404", component: errors.NotFoundPage()},
		{name: "リクエストが大きすぎる", component: errors.ContentTooLargePage("10MB")},
		{name: "メンテナンス", component: errors.MaintenancePage()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := i18n.SetLocale(context.Background(), "ja")
			var buf bytes.Buffer
			if err := tt.component.Render(ctx, &buf); err != nil {
				t.Fatalf("レンダリングに失敗: %v", err)
			}
			html := buf.String()

			for _, want := range []string{
				`<meta name="color-scheme" content="light dark">`,
				"color-scheme: light dark;",
				"@media (prefers-color-scheme: dark)",
				"background-color: #121a1b;",
				"color: #faf5f1;",
			} {
				if !strings.Contains(html, want) {
					t.Errorf("%qが含まれていない", want)
				}
			}
			// JavaScriptを読み込まない完全なHTMLのままにする
			if strings.Contains(html, "<script") {
				t.Error("scriptが含まれている")
			}
		})
	}
}
