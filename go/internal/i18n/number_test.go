package i18n_test

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
)

func TestFormatNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		locale string
		n      int
		want   string
	}{
		{name: "日本語で6桁の数値を桁区切りする", locale: i18n.LangJa, n: 500000, want: "500,000"},
		{name: "英語で6桁の数値を桁区切りする", locale: i18n.LangEn, n: 512345, want: "512,345"},
		{name: "3桁以下の数値は区切らない", locale: i18n.LangJa, n: 200, want: "200"},
		{name: "0はそのまま", locale: i18n.LangJa, n: 0, want: "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := i18n.SetLocale(context.Background(), tt.locale)
			if got := i18n.FormatNumber(ctx, tt.n); got != tt.want {
				t.Errorf("FormatNumber(%d) = %q、期待値 = %q", tt.n, got, tt.want)
			}
		})
	}
}
