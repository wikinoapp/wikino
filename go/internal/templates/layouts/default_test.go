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

	// The desktop nav is part of the header row, so nothing is fixed over the main content. Build the
	// removed rail's fixed-position class from fragments because Tailwind scans Go string literals;
	// spelling the complete class here would emit the dead utility solely for this check.
	//
	// [Ja] PC 向けのナビはヘッダーの行の一部のため、メインコンテンツの上に固定されるものは無い。
	// Tailwind は Go の文字列リテラルも走査するため、削除済みレールの固定配置クラスは断片から組み
	// 立てる。完全なクラスをここに書くと、この確認だけのために不要な utility が生成される。
	removedRailPosition := `class="fixed ` + strings.Join([]string{"left", "2"}, "-")
	if strings.Contains(html, removedRailPosition) {
		t.Error("desktop nav should not be fixed over the main content")
	}

	// The old off-canvas sidebar and its toggle are gone; the global nav replaces them.
	// [Ja] 旧 off-canvas サイドバーとその開閉ボタンは廃止され、グローバルナビが置き換える。
	if strings.Contains(html, `id="sidebar"`) {
		t.Error("legacy sidebar element should not be rendered")
	}
	if strings.Contains(html, "basecoat:sidebar") {
		t.Error("legacy sidebar toggle dispatch should not be rendered")
	}

	// The top bar and the bottom bar embed the same menu, so the home link renders exactly twice.
	// [Ja] 上部バーと下部バーは同じメニューを埋め込むため、ホームリンクはちょうど 2 回描画される。
	if got := strings.Count(html, `href="/home"`); got != 2 {
		t.Errorf(`href="/home" count = %d, want 2 (top bar + bottom bar)`, got)
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

	// Anonymous browsing of a public space is where the signed-out items show up.
	//
	// [Ja] 未ログイン時の項目が出るのは公開スペースの匿名閲覧。
	data := layouts.DefaultLayoutData{
		Meta:      viewmodel.PageMeta{Title: "テストタイトル"},
		GlobalNav: components.GlobalNavData{},
	}
	html := renderDefault(t, data, templ.Raw(""))

	// Signed-out visitors get home (root) + sign in in both the top bar and the bottom bar.
	// [Ja] 未ログインの訪問者は上部バー・下部バーともにホーム (ルート) + サインインが出る。
	if got := strings.Count(html, `href="/sign_in"`); got != 2 {
		t.Errorf(`href="/sign_in" count = %d, want 2 (top bar + bottom bar)`, got)
	}
	if strings.Contains(html, `href="/search"`) {
		t.Error("search link must not appear when signed out")
	}
}

func TestDefault_RendersBreadcrumbHeaderOutsideMain(t *testing.T) {
	t.Parallel()

	data := layouts.DefaultLayoutData{
		Meta: viewmodel.PageMeta{Title: "テストタイトル"},
		GlobalNav: components.GlobalNavData{
			CurrentPageName: templates.PageNameHome,
			SignedIn:        true,
			UserAtname:      "alice",
		},
		BreadcrumbHeader: components.BreadcrumbHeaderData{
			MaxWidthClass: "max-w-3xl",
			Items: []components.BreadcrumbItem{
				{Label: "テストスペース", Path: templates.Path("/s/test")},
			},
		},
	}
	html := renderDefault(t, data, templ.Raw(`<p>content-marker</p>`))

	if !strings.Contains(html, `aria-label="パンくずリスト"`) {
		t.Error("breadcrumb header should be rendered by the layout")
	}
	if !strings.Contains(html, "テストスペース") {
		t.Error("breadcrumb item label not found")
	}

	// The header must sit outside <main>: the skip link targets #main, so a header inside it would
	// be landed on rather than skipped.
	//
	// [Ja] ヘッダーは <main> の外になければならない。スキップリンクの飛び先は #main のため、
	// <main> の内側にあるヘッダーは飛ばされず着地先の内側に入ってしまう。
	header, main := strings.Index(html, `aria-label="パンくずリスト"`), strings.Index(html, `<main id="main"`)
	if header == -1 || main == -1 || header > main {
		t.Errorf("breadcrumb header (index %d) must precede <main> (index %d)", header, main)
	}
}

func TestDefault_RendersSharedHeaderWithoutBreadcrumbItems(t *testing.T) {
	t.Parallel()

	// The signed-in home has no breadcrumb items but still renders the header for the desktop
	// navigation bar. Its left slot stays empty rather than becoming a breadcrumb landmark with
	// nothing in it, and the header switches with the bar it carries so no banner landmark is left
	// with nothing in it below the breakpoint.
	//
	// [Ja] ログイン済みホームはパンくず項目を持たないが、PC 向けナビバーのためにヘッダーを描画する。
	// 左側は中身の無いパンくずランドマークにはせず空のままにし、ヘッダーは持っているバーと一緒に
	// 切り替わるため、ブレークポイント未満で中身の無い banner ランドマークが残らない。
	data := layouts.DefaultLayoutData{
		Meta: viewmodel.PageMeta{Title: "テストタイトル"},
		GlobalNav: components.GlobalNavData{
			CurrentPageName: templates.PageNameHome,
			SignedIn:        true,
			UserAtname:      "alice",
		},
		BreadcrumbHeader: components.BreadcrumbHeaderData{
			MaxWidthClass: "max-w-3xl",
		},
	}
	html := renderDefault(t, data, templ.Raw(""))

	if !strings.Contains(html, `<header class="hidden md:block">`) {
		t.Error("layout should render the shared header switching with the navigation bar")
	}
	if strings.Contains(html, `aria-label="パンくずリスト"`) {
		t.Error("layout should not render a breadcrumb landmark without breadcrumb items")
	}

	header, main := strings.Index(html, "<header"), strings.Index(html, `<main id="main"`)
	if header == -1 || main == -1 || header > main {
		t.Errorf("shared header (index %d) must precede <main> (index %d)", header, main)
	}
}

func TestDefault_ContentBottomPaddingMatchesBottomBar(t *testing.T) {
	t.Parallel()

	// The bottom padding keeps the last content clear of the fixed bottom bar, so it must be
	// dropped at exactly the width where the bar stops rendering (md). Match the full opening tag so
	// the assertion stays pinned to the content wrapper.
	//
	// [Ja] 下部余白は最下部のコンテンツが固定の下部バーに隠れないためのものなので、バーが描画され
	// なくなる幅 (md) とちょうど一致させて外す必要がある。コンテンツラッパーに固定するため開始タグ
	// 全体で照合する。
	const wantClass = "flex-1 flex flex-col min-h-screen pb-[calc(var(--app-bottom-nav-max-height)+0.5rem+env(safe-area-inset-bottom))] md:pb-0"

	data := layouts.DefaultLayoutData{
		Meta: viewmodel.PageMeta{Title: "テストタイトル"},
		GlobalNav: components.GlobalNavData{
			CurrentPageName: templates.PageNameHome,
			SignedIn:        true,
			UserAtname:      "alice",
		},
	}
	html := renderDefault(t, data, templ.Raw(""))

	if !strings.Contains(html, `<div class="`+wantClass+`">`) {
		t.Errorf("content wrapper に %q が含まれていない", wantClass)
	}
	if !strings.Contains(html, `<nav class="md:hidden"`) {
		t.Error("下部バーが余白の解除と同じ md で切り替わっていない")
	}
}

func TestDefault_HideNavigation(t *testing.T) {
	t.Parallel()

	globalNav := components.GlobalNavData{
		CurrentPageName: templates.PageNameHome,
		SignedIn:        true,
		UserAtname:      "alice",
	}
	breadcrumbHeader := components.BreadcrumbHeaderData{MaxWidthClass: "max-w-5xl"}

	shown := renderDefault(t, layouts.DefaultLayoutData{
		Meta:             viewmodel.PageMeta{Title: "テストタイトル"},
		GlobalNav:        globalNav,
		BreadcrumbHeader: breadcrumbHeader,
	}, templ.Raw(""))
	hidden := renderDefault(t, layouts.DefaultLayoutData{
		Meta:             viewmodel.PageMeta{Title: "テストタイトル"},
		HideNavigation:   true,
		GlobalNav:        globalNav,
		BreadcrumbHeader: breadcrumbHeader,
	}, templ.Raw(""))

	// With no breadcrumb items the header carries only the navigation bar, so HideNavigation drops the
	// header (leaving no banner landmark with nothing in it) together with the bottom bar and its
	// fixed wrapper.
	//
	// [Ja] パンくず項目が無い画面ではヘッダーの中身はナビバーだけなので、HideNavigation はヘッダーごと
	// 落とし (中身の無い banner ランドマークを残さない)、下部バーとその固定ラッパーも落とす。
	shownOnly := []string{
		"<header",
		`aria-label="グローバルナビゲーション"`,
		`aria-label="グローバルナビゲーション (モバイル)"`,
		`class="fixed bottom-2`,
	}
	for _, want := range shownOnly {
		if !strings.Contains(shown, want) {
			t.Errorf("既定では %q が描画されるべき", want)
		}
		if strings.Contains(hidden, want) {
			t.Errorf("HideNavigation 指定時に %q が描画されている", want)
		}
	}

	// The bottom padding only exists to keep content clear of the bottom bar, so it goes with the bar.
	//
	// [Ja] 下部余白は下部バーにコンテンツが隠れないためだけのものなので、バーと一緒に外れる。
	if !strings.Contains(shown, "md:pb-0") {
		t.Error("既定では下部バー分の余白が確保されるべき")
	}
	if strings.Contains(hidden, "md:pb-0") {
		t.Error("HideNavigation 指定時は下部バー分の余白を確保しないべき")
	}

	// The skip link follows the header. This case has neither breadcrumbs nor global navigation, so
	// HideNavigation removes the header and, with it, the skip link. The <main> landmark stays either
	// way: it is the page's main region, not a navigation part.
	//
	// [Ja] スキップリンクはヘッダーに追従する。このケースはパンくずもグローバルナビも無いため、
	// HideNavigation によってヘッダーとスキップリンクが落ちる。<main> ランドマークはどちらでも
	// 残る。ナビの部品ではなくページの主要領域だからである。
	if !strings.Contains(shown, `href="#main"`) {
		t.Error("既定ではスキップリンクが描画されるべき")
	}
	if strings.Contains(hidden, `href="#main"`) {
		t.Error("HideNavigation 指定時に飛ばす対象の無いスキップリンクが描画されている")
	}
	if !strings.Contains(hidden, `<main id="main" tabindex="-1">`) {
		t.Error("HideNavigation 指定時も main ランドマークは残るべき")
	}
}

func TestDefault_HideNavigationKeepsHeaderWithBreadcrumbItems(t *testing.T) {
	t.Parallel()

	// A screen with breadcrumb items keeps its header even out of the global navigation: the header
	// still has content, and only the navigation bar inside it is dropped. The skip link stays with the
	// header, because the breadcrumb left in front of the main content is still a block to bypass.
	//
	// [Ja] パンくず項目を持つ画面は、グローバルナビの対象外でもヘッダーを保つ。ヘッダーには中身が残って
	// おり、落ちるのは中のナビバーだけ。スキップリンクはヘッダーと一緒に残る。本文の前に残るパンくずは
	// 依然として飛ばすべきブロックだからである。
	data := layouts.DefaultLayoutData{
		Meta:           viewmodel.PageMeta{Title: "テストタイトル"},
		HideNavigation: true,
		GlobalNav:      components.GlobalNavData{CurrentPageName: templates.PageNameHome, SignedIn: true, UserAtname: "alice"},
		BreadcrumbHeader: components.BreadcrumbHeaderData{
			MaxWidthClass: "max-w-3xl",
			Items: []components.BreadcrumbItem{
				{Label: "テストスペース", Path: templates.Path("/s/test")},
			},
		},
	}
	html := renderDefault(t, data, templ.Raw(""))

	if !strings.Contains(html, `aria-label="パンくずリスト"`) {
		t.Error("パンくず項目があるヘッダーは描画されるべき")
	}
	if strings.Contains(html, `aria-label="グローバルナビゲーション"`) {
		t.Error("HideNavigation 指定時にナビバーが描画されている")
	}
	if !strings.Contains(html, `href="#main"`) {
		t.Error("ヘッダーが残る画面ではスキップリンクも描画されるべき")
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
