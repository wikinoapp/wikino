package layouts_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// renderDefault renders the default layout with a ja locale and the given data, returning the HTML.
//
// [Ja] renderDefault は default レイアウトを ja ロケール・指定データで描画し、HTML を返す。
func renderDefault(t *testing.T, data layouts.DefaultLayoutData, content templ.Component) string {
	t.Helper()

	ctx := i18n.SetLocale(context.Background(), "ja")

	var buf bytes.Buffer
	if err := layouts.Default(data, content).Render(ctx, &buf); err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}
	return buf.String()
}

func TestDefault_RendersGlobalNavAndSkipLink(t *testing.T) {
	t.Parallel()

	data := layouts.DefaultLayoutData{
		Meta: viewmodel.PageMeta{Title: "テストタイトル"},
		GlobalNav: components.GlobalNavData{
			CurrentPageName: templates.PageNameHome,
			SignedIn:        true,
			UserAtname:      "alice",
		},
	}
	html := renderDefault(t, data, templ.Raw(`<p>content-marker</p>`))

	checks := []string{
		"<!doctype html>",
		`<html lang="ja"`,
		"content-marker",
		`href="#main"`,
		"メインコンテンツへスキップ",
		`<main id="main" tabindex="-1">`,
		`aria-label="グローバルナビゲーション"`,
		`aria-label="グローバルナビゲーション (モバイル)"`,
	}
	for _, want := range checks {
		if !strings.Contains(html, want) {
			t.Errorf("Default layout output missing %q", want)
		}
	}

	// The main content reserves no left space for the rail, so it stays centered regardless of the
	// floating pill (the rail overlays the content).
	// [Ja] メインコンテンツはレール用の左余白を確保しないため、浮遊ピルに関係なく中央のまま
	// (レールは本文にオーバーレイする)。
	if strings.Contains(html, "nav-rail-width") {
		t.Error("main content should not reserve left space for the rail")
	}

	// The old off-canvas sidebar and its toggle are gone; the global nav replaces them.
	// [Ja] 旧 off-canvas サイドバーとその開閉ボタンは廃止され、グローバルナビが置き換える。
	if strings.Contains(html, `id="sidebar"`) {
		t.Error("legacy sidebar element should not be rendered")
	}
	if strings.Contains(html, "basecoat:sidebar") {
		t.Error("legacy sidebar toggle dispatch should not be rendered")
	}

	// The rail and the bottom bar embed the same menu, so the home link renders exactly twice.
	// [Ja] レールと下部バーは同じメニューを埋め込むため、ホームリンクはちょうど 2 回描画される。
	if got := strings.Count(html, `href="/home"`); got != 2 {
		t.Errorf(`href="/home" count = %d, want 2 (rail + bottom bar)`, got)
	}

	// The skip link must be the first focusable element, so it must precede the nav links in the DOM.
	// [Ja] スキップリンクは最初のフォーカス可能要素でなければならないため、DOM 上でナビのリンクより
	// 前に現れる必要がある。
	if skip, nav := strings.Index(html, `href="#main"`), strings.Index(html, `href="/home"`); skip == -1 || nav == -1 || skip > nav {
		t.Errorf("skip link (index %d) must precede nav links (index %d)", skip, nav)
	}
}

func TestDefault_SignedOutNav(t *testing.T) {
	t.Parallel()

	data := layouts.DefaultLayoutData{
		Meta: viewmodel.PageMeta{Title: "テストタイトル"},
		GlobalNav: components.GlobalNavData{
			CurrentPageName: templates.PageNameWelcome,
		},
	}
	html := renderDefault(t, data, templ.Raw(""))

	// Signed-out visitors get home (root) + sign in in both the rail and the bottom bar.
	// [Ja] 未ログインの訪問者はレール・下部バーともにホーム (ルート) + サインインが出る。
	if got := strings.Count(html, `href="/sign_in"`); got != 2 {
		t.Errorf(`href="/sign_in" count = %d, want 2 (rail + bottom bar)`, got)
	}
	if strings.Contains(html, `href="/search"`) {
		t.Error("search link must not appear when signed out")
	}
}

func TestDefault_HideFooter(t *testing.T) {
	t.Parallel()

	base := components.GlobalNavData{CurrentPageName: templates.PageNameHome, SignedIn: true, UserAtname: "alice"}

	shown := renderDefault(t, layouts.DefaultLayoutData{Meta: viewmodel.PageMeta{Title: "t"}, GlobalNav: base}, templ.Raw(""))
	hidden := renderDefault(t, layouts.DefaultLayoutData{Meta: viewmodel.PageMeta{Title: "t"}, HideFooter: true, GlobalNav: base}, templ.Raw(""))

	// The footer renders by default and is dropped when HideFooter is set (e.g. the page editor).
	// [Ja] フッターは既定で描画され、HideFooter 指定時 (編集画面など) は描画されない。
	if !strings.Contains(shown, "<footer") {
		t.Error("footer should render by default")
	}
	if strings.Contains(hidden, "<footer") {
		t.Error("footer should be hidden when HideFooter is set")
	}
}
