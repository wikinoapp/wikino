package components_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// renderHeadは共通headをjaロケールで描画し、HTMLを返す。
func renderHead(t *testing.T, meta viewmodel.PageMeta) string {
	t.Helper()

	ctx := i18n.SetLocale(context.Background(), "ja")

	var buf bytes.Buffer
	if err := components.Head(meta).Render(ctx, &buf); err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}
	return buf.String()
}

// 正規アドレスが分かっている画面は、それを持つ2箇所の双方で宣言する。canonicalリンクと
// og:urlがずれることがないようにするためである。
func TestHead_DeclaresCanonicalURLWhenSet(t *testing.T) {
	t.Parallel()

	html := renderHead(t, viewmodel.PageMeta{OGURL: "https://localhost/s/example/pages/1"})

	for _, want := range []string{
		`<link rel="canonical" href="https://localhost/s/example/pages/1">`,
		`<meta property="og:url" content="https://localhost/s/example/pages/1">`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("headに%qが含まれていない", want)
		}
	}
}

// 空の値は「正規アドレスを宣言しない」ことと同じではない。空のhrefも空のcontentも
// リクエストされたURLに解決されるため、画面のクエリ違いがそれぞれ自分自身を正規アドレスとして
// 宣言してしまう。正規アドレスを持たない画面はどちらの要素も出さない。
func TestHead_OmitsCanonicalURLWhenUnset(t *testing.T) {
	t.Parallel()

	html := renderHead(t, viewmodel.PageMeta{})

	for _, notWant := range []string{
		`<link rel="canonical"`,
		`property="og:url"`,
	} {
		if strings.Contains(html, notWant) {
			t.Errorf("headに%qが含まれている", notWant)
		}
	}
}

// twitter:cardは画面ごとに変わる。ページ固有の画像を出す画面だけがXで大きなカードになるよう、
// PageMetaの値をそのまま出す。
func TestHead_DeclaresTwitterCard(t *testing.T) {
	t.Parallel()

	for _, card := range []string{viewmodel.TwitterCardSummary, viewmodel.TwitterCardSummaryLargeImage} {
		html := renderHead(t, viewmodel.PageMeta{TwitterCard: card})

		want := `<meta name="twitter:card" content="` + card + `">`
		if !strings.Contains(html, want) {
			t.Errorf("headに%qが含まれていない", want)
		}
	}
}

// 寸法はog:imageが常に同じ寸法の画像 (カード画像) を指すときだけ宣言する。
// リサイズ後の寸法が画像ごとに変わるアイキャッチ画像では誤った寸法を宣言しないよう出さない。
func TestHead_OGImageDimensions(t *testing.T) {
	t.Parallel()

	t.Run("寸法があれば宣言する", func(t *testing.T) {
		t.Parallel()

		html := renderHead(t, viewmodel.PageMeta{OGImage: "https://localhost/card.png", OGImageWidth: 1200, OGImageHeight: 630})

		for _, want := range []string{
			`<meta property="og:image:width" content="1200">`,
			`<meta property="og:image:height" content="630">`,
		} {
			if !strings.Contains(html, want) {
				t.Errorf("headに%qが含まれていない", want)
			}
		}
	})

	t.Run("寸法が無ければ宣言しない", func(t *testing.T) {
		t.Parallel()

		html := renderHead(t, viewmodel.PageMeta{OGImage: "https://localhost/cover.jpg"})

		for _, notWant := range []string{`property="og:image:width"`, `property="og:image:height"`} {
			if strings.Contains(html, notWant) {
				t.Errorf("headに%qが含まれている", notWant)
			}
		}
	})
}
