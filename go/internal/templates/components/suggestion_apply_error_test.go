package components_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

func TestSuggestionApplyError_PageTitle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		pageTitle string
		want      string
	}{
		{name: "タイトルのあるページはタイトルで示す", pageTitle: "ページのタイトル", want: "ページ「ページのタイトル」: "},
		{name: "タイトルの無いページは無題で示す", pageTitle: "", want: "ページ「無題」: "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := i18n.SetLocale(t.Context(), i18n.LangJa)
			data := &viewmodel.SuggestionApplyError{
				PageErrors: []viewmodel.SuggestionApplyPageError{
					{PageTitle: tt.pageTitle, Message: "本文が長すぎます"},
				},
			}

			var buf bytes.Buffer
			if err := components.SuggestionApplyError(data).Render(ctx, &buf); err != nil {
				t.Fatalf("レンダリングに失敗: %v", err)
			}

			if html := buf.String(); !strings.Contains(html, tt.want) {
				t.Errorf("出力に %qが含まれていない: %s", tt.want, html)
			}
		})
	}
}
