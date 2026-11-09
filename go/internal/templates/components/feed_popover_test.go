package components_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
)

func TestFeedPopover(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		locale           string
		publicTopicsOnly bool
		wants            []string
		notWants         []string
	}{
		{
			name:             "日本語でスペースのフィード",
			locale:           i18n.LangJa,
			publicTopicsOnly: true,
			wants: []string{
				`aria-label="RSSフィード"`,
				`aria-label="RSSフィードのURLをコピー"`,
				"RSSリーダーにこのURLを登録すると、新しく公開されたページを受け取れます",
				"公開トピックのページだけを配信します",
				"RSSフィードについて",
			},
		},
		{
			name:             "英語でトピックのフィード",
			locale:           i18n.LangEn,
			publicTopicsOnly: false,
			wants: []string{
				`aria-label="RSS feed"`,
				`aria-label="Copy RSS feed URL"`,
				"About RSS feeds",
			},
			notWants: []string{"Only pages in public topics are included"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := i18n.SetLocale(t.Context(), tt.locale)

			var buf bytes.Buffer
			err := components.FeedPopover(components.FeedPopoverData{
				URL:              "https://example.com/s/demo/atom",
				PublicTopicsOnly: tt.publicTopicsOnly,
			}).Render(ctx, &buf)
			if err != nil {
				t.Fatalf("Render()のエラー = %v", err)
			}

			html := buf.String()
			wants := append([]string{
				// Basecoatのpopoverが開閉を付ける目印と、トリガーが指すポップオーバー
				`<div id="feed-popover" class="popover static">`,
				`aria-controls="feed-popover-popover"`,
				`id="feed-popover-popover" data-popover aria-hidden="true"`,
				`value="https://example.com/s/demo/atom"`,
				// ヘルプは日本語だけのため、表示言語によらず同じページへリンクする
				`href="https://wikino.app/s/wikino/pages/355"`,
				// gridの中でリンクが行の幅いっぱいに広がらず、文字の部分だけを押せる
				`class="link-foreground text-sm justify-self-start"`,
			}, tt.wants...)
			for _, want := range wants {
				if !strings.Contains(html, want) {
					t.Errorf("出力に%qが含まれていない\n%s", want, html)
				}
			}
			for _, notWant := range tt.notWants {
				if strings.Contains(html, notWant) {
					t.Errorf("出力に%qが含まれている", notWant)
				}
			}
		})
	}
}
