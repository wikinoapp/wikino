package components_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/templates/components"
)

func TestPlainTextBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		text         string
		wantContains []string
		wantExcludes []string
	}{
		{
			name: "通常のテキストが p タグでラップされる",
			text: "Hello, World",
			wantContains: []string{
				`<p class="whitespace-pre-wrap">`,
				"Hello, World",
				"</p>",
			},
		},
		{
			name: "URL が <a> タグに変換される",
			text: "See https://example.com for more info",
			wantContains: []string{
				`<p class="whitespace-pre-wrap">`,
				`<a href="https://example.com" rel="noopener noreferrer" target="_blank">https://example.com</a>`,
				"</p>",
			},
		},
		{
			name: "HTML タグがエスケープされる",
			text: "<script>alert('xss')</script>",
			wantContains: []string{
				"&lt;script&gt;",
			},
			wantExcludes: []string{
				"<script>",
			},
		},
		{
			name: "空文字列でも p タグはレンダリングされる",
			text: "",
			wantContains: []string{
				`<p class="whitespace-pre-wrap">`,
				"</p>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			data := components.PlainTextBodyData{Text: tt.text}

			var buf bytes.Buffer
			err := components.PlainTextBody(data).Render(ctx, &buf)
			if err != nil {
				t.Fatalf("レンダリングに失敗: %v", err)
			}

			output := buf.String()

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("出力に %q が含まれていない\n出力: %s", want, output)
				}
			}

			for _, exclude := range tt.wantExcludes {
				if strings.Contains(output, exclude) {
					t.Errorf("出力に %q が含まれるべきではない\n出力: %s", exclude, output)
				}
			}
		})
	}
}
