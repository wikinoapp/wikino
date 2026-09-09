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

// renderHead renders the shared head with a ja locale and returns the HTML.
//
// [Ja] renderHead は共通 head を ja ロケールで描画し、HTML を返す。
func renderHead(t *testing.T, meta viewmodel.PageMeta) string {
	t.Helper()

	ctx := i18n.SetLocale(context.Background(), "ja")

	var buf bytes.Buffer
	if err := components.Head(meta).Render(ctx, &buf); err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}
	return buf.String()
}

// A screen that knows its canonical address declares it in both places that carry it, so the
// canonical link and og:url never drift apart.
//
// [Ja] 正規アドレスが分かっている画面は、それを持つ 2 箇所の双方で宣言する。canonical リンクと
// og:url がずれることがないようにするためである。
func TestHead_DeclaresCanonicalURLWhenSet(t *testing.T) {
	t.Parallel()

	html := renderHead(t, viewmodel.PageMeta{OGURL: "https://localhost/s/example/pages/1"})

	for _, want := range []string{
		`<link rel="canonical" href="https://localhost/s/example/pages/1">`,
		`<meta property="og:url" content="https://localhost/s/example/pages/1">`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("head does not contain %q", want)
		}
	}
}

// An empty value is not the same as declaring no canonical address: an empty href and an empty
// content both resolve to the requested URL, so every query variant of a screen would declare
// itself as its own canonical address. A screen that has no canonical address emits neither
// element.
//
// [Ja] 空の値は「正規アドレスを宣言しない」ことと同じではない。空の href も空の content も
// リクエストされた URL に解決されるため、画面のクエリ違いがそれぞれ自分自身を正規アドレスとして
// 宣言してしまう。正規アドレスを持たない画面はどちらの要素も出さない。
func TestHead_OmitsCanonicalURLWhenUnset(t *testing.T) {
	t.Parallel()

	html := renderHead(t, viewmodel.PageMeta{})

	for _, notWant := range []string{
		`<link rel="canonical"`,
		`property="og:url"`,
	} {
		if strings.Contains(html, notWant) {
			t.Errorf("head unexpectedly contains %q", notWant)
		}
	}
}
