package components_test

import (
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
)

func TestRequiredBadge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		locale string
		want   string
	}{
		{name: "日本語", locale: "ja", want: "必須"},
		{name: "英語", locale: "en", want: "Required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := i18n.SetLocale(context.Background(), tt.locale)

			var buf strings.Builder
			if err := components.RequiredBadge().Render(ctx, &buf); err != nil {
				t.Fatalf("レンダリングエラー: %v", err)
			}
			html := buf.String()

			// 凡例が無くても意味が伝わり、スクリーンリーダーにも言葉として読まれるよう、必須を文言で示す
			if !strings.Contains(html, tt.want) {
				t.Errorf("レスポンスに%qが含まれていない: %q", tt.want, html)
			}
			// 状態を示す色に見える別のvariantへ流れないよう、outlineのバッジに固定する
			if !strings.Contains(html, `class="badge" data-variant="outline"`) {
				t.Errorf("必須の印がoutlineのバッジで表示されていない: %q", html)
			}
			// エラーの色はバリデーションエラーのもので、平常の状態である必須が借りてはいけない
			if strings.Contains(html, "text-error") || strings.Contains(html, "destructive") {
				t.Errorf("必須の印がエラー用の色を使っている: %q", html)
			}
		})
	}
}
