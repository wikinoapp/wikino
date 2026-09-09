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

// Link cards render the same destination as two icon-only links: one revealed on hover for
// non-touch devices and one always visible for touch devices. These are the leading classes of
// each link's wrapper, used to scope assertions to one of the two.
//
// [Ja] リンクカードは同じ遷移先をアイコンのみのリンク 2 つとして描画する (非タッチ端末の
// ホバー表示と、タッチ端末の常時表示)。以下は各リンクのラッパーの先頭クラスで、アサートを
// 一方のリンクへスコープするために使う。
const (
	nonTouchLinkWrapper = "touch:hidden non-touch:flex"
	touchLinkWrapper    = "touch:flex non-touch:hidden"
)

func TestCardLinkPage_EditLinkAccessibleName(t *testing.T) {
	t.Parallel()

	page := viewmodel.CardLinkPage{
		Title:   "ページタイトル",
		Number:  7,
		CanEdit: true,
	}

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)

	var buf bytes.Buffer
	if err := components.CardLinkPage(page, viewmodel.SpaceIdentifier("my-space")).Render(ctx, &buf); err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}

	html := buf.String()

	assertCardLinkIconLinkLabels(t, html, "編集する")
	assertCardLinkFocusVisible(t, html)
}

// assertCardLinkIconLinkLabels checks that both icon-only links of a link card carry wantLabel as
// their own accessible name and that their icons are decorative.
//
// Only the hover-revealed link keeps a title tooltip. It renders exclusively under
// (hover: hover) and (pointer: fine), where the tooltip does fire and is the only visual hint for
// an icon that appears on hover alone. The touch link renders where a tooltip never fires.
//
// [Ja] assertCardLinkIconLinkLabels は、リンクカードのアイコンのみリンク 2 つが wantLabel を
// それぞれのアクセシブルネームとして持ち、アイコンが装飾として扱われていることを確認する。
//
// title のツールチップを持つのはホバー表示のリンクだけである。こちらは
// (hover: hover) and (pointer: fine) でしか描画されず、ツールチップが発火するうえ、ホバー時に
// だけ現れるアイコンにとって唯一の視覚的な補助になる。タッチ端末向けリンクはツールチップが
// 発火しない環境でしか描画されない。
func assertCardLinkIconLinkLabels(t *testing.T, html, wantLabel string) {
	t.Helper()

	nonTouch := cardLinkIconLink(t, html, nonTouchLinkWrapper, "非タッチ端末向けリンク")
	touch := cardLinkIconLink(t, html, touchLinkWrapper, "タッチ端末向けリンク")

	links := []struct {
		name    string
		segment string
	}{
		{name: "非タッチ端末向けリンク", segment: nonTouch},
		{name: "タッチ端末向けリンク", segment: touch},
	}
	for _, link := range links {
		if !strings.Contains(link.segment, `aria-label="`+wantLabel+`"`) {
			t.Errorf("%s に aria-label %q が無い", link.name, wantLabel)
		}
		if !strings.Contains(link.segment, `<svg aria-hidden="true" focusable="false"`) {
			t.Errorf("%s のアイコンが装飾として扱われていない", link.name)
		}
	}

	if !strings.Contains(nonTouch, `title="`+wantLabel+`"`) {
		t.Errorf("非タッチ端末向けリンクに title %q が無い", wantLabel)
	}
	if strings.Contains(touch, "title=") {
		t.Error("タッチ端末向けリンクに title が付いている")
	}
}

// assertCardLinkFocusVisible checks that the hover-revealed link becomes visible on keyboard
// focus. It starts at opacity 0 and its ancestor never enters :hover during keyboard navigation,
// so without these classes a keyboard user lands on a link that stays invisible.
//
// [Ja] assertCardLinkFocusVisible は、ホバー表示のリンクがキーボードフォーカス時にも可視化
// されることを確認する。このリンクは opacity 0 から始まり、キーボード操作では祖先が :hover に
// ならないため、これらのクラスが無いとキーボード利用者は不可視のリンクへ移動することになる。
func assertCardLinkFocusVisible(t *testing.T, html string) {
	t.Helper()

	nonTouch := cardLinkIconLink(t, html, nonTouchLinkWrapper, "非タッチ端末向けリンク")

	focusClasses := []string{
		"focus-visible:opacity-100",
		"focus-visible:bg-primary",
		"focus-visible:ring-[3px]",
		"focus-visible:ring-ring/50",
	}
	for _, className := range focusClasses {
		if !strings.Contains(nonTouch, className) {
			t.Errorf("非タッチ端末向けリンクにフォーカス表示クラス %q が無い", className)
		}
	}
}

// cardLinkIconLink returns the icon link rendered inside the wrapper carrying wrapperClass.
//
// [Ja] cardLinkIconLink は wrapperClass を持つラッパー内に描画されるアイコンリンクを返す。
func cardLinkIconLink(t *testing.T, html, wrapperClass, name string) string {
	t.Helper()

	segment := elementSegment(html, wrapperClass, "</a>")
	if segment == "" {
		t.Fatalf("%sが見つからない", name)
	}

	return segment
}
