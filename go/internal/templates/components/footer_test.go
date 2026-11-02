package components_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
)

func TestFooter_法的文書へのリンクは表示言語に応じた本文のページを指す(t *testing.T) {
	t.Parallel()

	tests := []struct {
		locale string
		want   []string
	}{
		{locale: "ja", want: []string{`href="/s/wikino/pages/41"`, `href="/s/wikino/pages/42"`}},
		{locale: "en", want: []string{`href="/s/wikino/pages/325"`, `href="/s/wikino/pages/328"`}},
	}
	for _, tt := range tests {
		t.Run(tt.locale, func(t *testing.T) {
			t.Parallel()

			ctx := i18n.SetLocale(context.Background(), tt.locale)
			var buf bytes.Buffer
			if err := components.Footer().Render(ctx, &buf); err != nil {
				t.Fatalf("Render()のエラー = %v", err)
			}
			html := buf.String()
			for _, want := range tt.want {
				if !strings.Contains(html, want) {
					t.Errorf("フッターに%qが無い: %s", want, html)
				}
			}
			// 固定URLは転送を挟むため、画面のリンクには使わない
			for _, unwanted := range []string{`href="/terms"`, `href="/privacy"`} {
				if strings.Contains(html, unwanted) {
					t.Errorf("フッターに固定URLの%qがある", unwanted)
				}
			}
		})
	}
}
