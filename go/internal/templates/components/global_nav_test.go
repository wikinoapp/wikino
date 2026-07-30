package components_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// renderGlobalNav renders c with a ja locale and returns the HTML.
//
// [Ja] renderGlobalNav は c を ja ロケールで描画し、HTML を返す。
func renderGlobalNav(t *testing.T, c templ.Component) string {
	t.Helper()

	ctx := i18n.SetLocale(context.Background(), "ja")

	var buf bytes.Buffer
	if err := c.Render(ctx, &buf); err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}
	return buf.String()
}

// anchorSegment returns the substring of html covering the <a> element whose
// markup contains the given href, so a test can assert on that link in
// isolation (its aria-current, inner icon, and so on).
//
// [Ja] anchorSegment は指定した href を含む <a> 要素の範囲を切り出して返す。
// その 1 リンク (aria-current・内側のアイコンなど) を単独で検証するために使う。
func anchorSegment(html, href string) string {
	marker := `href="` + href + `"`
	i := strings.Index(html, marker)
	if i < 0 {
		return ""
	}
	start := strings.LastIndex(html[:i], "<a ")
	end := strings.Index(html[i:], "</a>")
	if start < 0 || end < 0 {
		return ""
	}
	return html[start : i+end]
}

// svgSegment returns the first complete SVG element in html.
//
// [Ja] svgSegment は html 内の最初の完全な SVG 要素を返す。
func svgSegment(html string) string {
	start := strings.Index(html, "<svg")
	if start < 0 {
		return ""
	}
	end := strings.Index(html[start:], "</svg>")
	if end < 0 {
		return ""
	}
	return html[start : start+end+len("</svg>")]
}

func TestGlobalNavMenu_SignedInLinks(t *testing.T) {
	t.Parallel()

	data := components.GlobalNavData{
		SignedIn:        true,
		UserAtname:      "alice",
		CurrentPageName: templates.PageNameHome,
	}
	html := renderGlobalNav(t, components.GlobalNavMenu(data, ""))

	// Signed-in users get home / search / profile, and no sign-in link.
	// [Ja] ログイン時はホーム / 検索 / プロフィールが出て、サインインリンクは出ない。
	for _, want := range []string{`href="/home"`, `href="/search"`, `href="/@alice"`} {
		if !strings.Contains(html, want) {
			t.Errorf("GlobalNavMenu にリンク %q が含まれていない", want)
		}
	}
	if strings.Contains(html, "/sign_in") {
		t.Error("ログイン時にサインインリンクが表示されるべきではない")
	}

	// Each of the three items renders exactly one icon.
	// [Ja] 3 項目それぞれがアイコンを 1 つずつ描画する。
	if got := strings.Count(html, "<svg"); got != 3 {
		t.Errorf("svg の数 = %d, want 3", got)
	}
}

func TestGlobalNavMenu_SignedOutLinks(t *testing.T) {
	t.Parallel()

	// A signed-out visitor is browsing a public space, the situation the signed-out items exist for.
	//
	// [Ja] 未ログインの訪問者が公開スペースを閲覧している状況を置く。未ログイン時の項目はこのために
	// 存在する。
	data := components.GlobalNavData{}
	html := renderGlobalNav(t, components.GlobalNavMenu(data, ""))

	// Signed-out users get home (root) and sign-in only.
	// [Ja] 未ログイン時はホーム (ルート) とサインインのみが出る。
	for _, want := range []string{`href="/"`, `href="/sign_in"`} {
		if !strings.Contains(html, want) {
			t.Errorf("GlobalNavMenu にリンク %q が含まれていない", want)
		}
	}

	// Neither signed-out item is ever active: the top page the home link points at renders no
	// navigation at all, so no screen showing these items is one of them.
	//
	// [Ja] 未ログイン時の項目はいずれもアクティブにならない。ホームリンクが指すトップページはナビ自体を
	// 描画しないため、これらの項目が出る画面はどれも項目の指す画面ではない。
	if strings.Contains(html, `aria-current="page"`) {
		t.Error("未ログイン時にアクティブ項目があるべきではない")
	}

	// Search and profile require authentication, so they are omitted.
	// [Ja] 検索・プロフィールはログイン必須のため出さない。
	if strings.Contains(html, `href="/search"`) {
		t.Error("未ログイン時に検索リンクが表示されるべきではない")
	}
	if strings.Contains(html, `href="/@`) {
		t.Error("未ログイン時にプロフィールリンクが表示されるべきではない")
	}
	if strings.Contains(html, `href="/home"`) {
		t.Error("未ログイン時のホームはルート (/) であるべき")
	}

	if got := strings.Count(html, "<svg"); got != 2 {
		t.Errorf("svg の数 = %d, want 2", got)
	}
}

func TestGlobalNavMenu_ActiveHighlight(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		pageName   templates.PageName
		activeHref string
	}{
		{name: "home がアクティブ", pageName: templates.PageNameHome, activeHref: "/home"},
		{name: "search がアクティブ", pageName: templates.PageNameSearch, activeHref: "/search"},
		{name: "profile がアクティブ", pageName: templates.PageNameProfile, activeHref: "/@alice"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data := components.GlobalNavData{
				SignedIn:        true,
				UserAtname:      "alice",
				CurrentPageName: tt.pageName,
			}
			html := renderGlobalNav(t, components.GlobalNavMenu(data, ""))

			// Exactly one item is marked as the current page.
			// [Ja] 現在ページとしてマークされる項目は 1 つだけ。
			if got := strings.Count(html, `aria-current="page"`); got != 1 {
				t.Errorf("aria-current の数 = %d, want 1", got)
			}

			// The active marker must belong to the active item's link.
			// [Ja] アクティブなマーカーはアクティブ項目のリンク内に存在しなければならない。
			seg := anchorSegment(html, tt.activeHref)
			if seg == "" {
				t.Fatalf("%q のアンカーが見つからない", tt.activeHref)
			}
			if !strings.Contains(seg, `aria-current="page"`) {
				t.Errorf("%q のリンクがアクティブとして描画されていない", tt.activeHref)
			}
		})
	}
}

func TestGlobalNavMenu_NoActiveWhenUnmatched(t *testing.T) {
	t.Parallel()

	// On a page none of the items map to (e.g. a topic page), no item is active.
	// [Ja] どの項目にも対応しないページ (トピックページなど) では、いずれの項目もアクティブにしない。
	data := components.GlobalNavData{
		SignedIn:        true,
		UserAtname:      "alice",
		CurrentPageName: templates.PageNameTopicShow,
	}
	html := renderGlobalNav(t, components.GlobalNavMenu(data, ""))

	if strings.Contains(html, `aria-current="page"`) {
		t.Error("対応しないページではアクティブ項目が無いべき")
	}
}

func TestGlobalNavMenu_ActiveIconSwap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		pageName    templates.PageName
		href        string
		activeIcon  viewmodel.IconName
		defaultIcon viewmodel.IconName
	}{
		{name: "home", pageName: templates.PageNameHome, href: "/home", activeIcon: "house-fill", defaultIcon: "house-regular"},
		{name: "search", pageName: templates.PageNameSearch, href: "/search", activeIcon: "magnifying-glass-fill", defaultIcon: "magnifying-glass-regular"},
		{name: "profile", pageName: templates.PageNameProfile, href: "/@alice", activeIcon: "user-circle-fill", defaultIcon: "user-circle-regular"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			activeData := components.GlobalNavData{SignedIn: true, UserAtname: "alice", CurrentPageName: tt.pageName}
			inactiveData := components.GlobalNavData{SignedIn: true, UserAtname: "alice", CurrentPageName: templates.PageNameTopicShow}

			activeSVG := svgSegment(anchorSegment(renderGlobalNav(t, components.GlobalNavMenu(activeData, "")), tt.href))
			inactiveSVG := svgSegment(anchorSegment(renderGlobalNav(t, components.GlobalNavMenu(inactiveData, "")), tt.href))
			wantActiveSVG := svgSegment(renderGlobalNav(t, templates.DecorativeIcon(tt.activeIcon, "size-6")))
			wantInactiveSVG := svgSegment(renderGlobalNav(t, templates.DecorativeIcon(tt.defaultIcon, "size-6")))

			if activeSVG != wantActiveSVG {
				t.Error("アクティブ時の SVG が期待値と異なる")
			}
			if inactiveSVG != wantInactiveSVG {
				t.Error("非アクティブ時の SVG が期待値と異なる")
			}
		})
	}
}

func TestGlobalNavMenu_AriaLabels(t *testing.T) {
	t.Parallel()

	type expectedLink struct {
		href  string
		label string
	}
	tests := []struct {
		name  string
		data  components.GlobalNavData
		links []expectedLink
	}{
		{
			name: "ログイン時",
			data: components.GlobalNavData{SignedIn: true, UserAtname: "alice", CurrentPageName: templates.PageNameHome},
			links: []expectedLink{
				{href: "/home", label: "ホーム"},
				{href: "/search", label: "検索"},
				{href: "/@alice", label: "プロフィール"},
			},
		},
		{
			name: "未ログイン時",
			data: components.GlobalNavData{},
			links: []expectedLink{
				{href: "/", label: "ホーム"},
				{href: "/sign_in", label: "ログイン"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			html := renderGlobalNav(t, components.GlobalNavMenu(tt.data, ""))
			for _, link := range tt.links {
				segment := anchorSegment(html, link.href)
				if !strings.Contains(segment, `aria-label="`+link.label+`"`) {
					t.Errorf("%q のリンクに aria-label %q が含まれていない", link.href, link.label)
				}
			}
		})
	}
}

func TestGlobalNavMenu_DecorativeIcons(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		data      components.GlobalNavData
		iconCount int
	}{
		{
			name:      "ログイン時",
			data:      components.GlobalNavData{SignedIn: true, UserAtname: "alice"},
			iconCount: 3,
		},
		{
			name:      "未ログイン時",
			data:      components.GlobalNavData{},
			iconCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			html := renderGlobalNav(t, components.GlobalNavMenu(tt.data, ""))
			if got := strings.Count(html, `aria-hidden="true"`); got != tt.iconCount {
				t.Errorf("aria-hidden を持つ SVG の数 = %d, want %d", got, tt.iconCount)
			}
			if got := strings.Count(html, `focusable="false"`); got != tt.iconCount {
				t.Errorf("focusable=false を持つ SVG の数 = %d, want %d", got, tt.iconCount)
			}
		})
	}
}

func TestGlobalNavMenu_SpaceFilterSearchPath(t *testing.T) {
	t.Parallel()

	// Inside a space, the search link is scoped to that space.
	// [Ja] スペース内では検索リンクがそのスペースに絞り込まれる。
	data := components.GlobalNavData{SignedIn: true, UserAtname: "alice", SpaceIdentifier: "my-space"}
	html := renderGlobalNav(t, components.GlobalNavMenu(data, ""))

	if !strings.Contains(html, "space:my-space") {
		t.Error("スペースフィルター付きの検索パスが含まれていない")
	}
}

func TestGlobalNavMenu_ItemsRoundedFull(t *testing.T) {
	t.Parallel()

	// Each item link is fully rounded so its hover highlight reads as a circle.
	// Render with an empty container className so only the link's own rounding shows.
	// [Ja] 各項目リンクは完全な円 (rounded-full) で、hover ハイライトが円形に見える。
	// リンク自身の角丸だけを見るため、コンテナの className を空にして描画する。
	data := components.GlobalNavData{SignedIn: true, UserAtname: "alice", CurrentPageName: templates.PageNameHome}
	html := renderGlobalNav(t, components.GlobalNavMenu(data, ""))

	seg := anchorSegment(html, "/home")
	if seg == "" {
		t.Fatal(`"/home" のアンカーが見つからない`)
	}
	if !strings.Contains(seg, "rounded-full") {
		t.Error("ナビ項目のリンクは rounded-full であるべき")
	}
	if strings.Contains(seg, "rounded-md") {
		t.Error("ナビ項目のリンクに rounded-md が残っている")
	}
}

func TestGlobalNavMenu_EnglishLocale(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), "en")
	data := components.GlobalNavData{SignedIn: true, UserAtname: "alice", CurrentPageName: templates.PageNameHome}

	var buf bytes.Buffer
	if err := components.GlobalNavMenu(data, "").Render(ctx, &buf); err != nil {
		t.Fatalf("failed to render: %v", err)
	}
	html := buf.String()

	for _, want := range []string{`aria-label="Home"`, `aria-label="Search"`, `aria-label="Profile"`} {
		if !strings.Contains(html, want) {
			t.Errorf("GlobalNavMenu output missing %q", want)
		}
	}
}

func TestGlobalNavTopBar(t *testing.T) {
	t.Parallel()

	data := components.GlobalNavData{SignedIn: true, UserAtname: "alice", CurrentPageName: templates.PageNameHome}
	html := renderGlobalNav(t, components.GlobalNavTopBar(data))

	// The top bar sits in the header's normal flow, is shown only at and above md (the width
	// GlobalNavBottomBar stops rendering at), and keeps its icons at full size next to a long
	// breadcrumb. It carries no pill of its own, so nothing overlays the content. Match the full
	// class attribute so the assertion stays pinned to the visibility pair with the bottom bar.
	//
	// [Ja] 上部バーはヘッダーの通常フローに入り、md (GlobalNavBottomBar が描画されなくなる幅) 以上で
	// のみ表示し、長いパンくずの隣でもアイコンを潰さない。自身のピルは持たないため、本文にオーバー
	// レイするものは無い。下部バーとの表示切り替えの対に固定するため class 属性全体で照合する。
	for _, want := range []string{`class="shrink-0 hidden md:flex"`, `href="/home"`} {
		if !strings.Contains(html, want) {
			t.Errorf("GlobalNavTopBar に %q が含まれていない", want)
		}
	}
	// Build the removed rail transform from fragments because Tailwind scans Go string literals;
	// spelling the complete class here would emit the dead utility solely for this negative check.
	//
	// [Ja] Tailwind は Go の文字列リテラルも走査するため、削除済みレールの transform は断片から
	// 組み立てる。完全なクラスをここに書くと、この否定確認だけのために不要な utility が生成される。
	removedRailTransform := strings.Join([]string{"-translate-y", "1/2"}, "-")
	for _, notWant := range []string{"fixed", removedRailTransform, "bg-card", "border"} {
		if strings.Contains(html, notWant) {
			t.Errorf("GlobalNavTopBar に浮遊ピルのクラス %q が残っている", notWant)
		}
	}

	// The wrapper is a <nav> landmark carrying its own aria-label.
	// [Ja] ラッパーは固有の aria-label を持つ <nav> ランドマーク。
	if !strings.Contains(html, "<nav") {
		t.Error("GlobalNavTopBar に <nav> ランドマークが無い")
	}
	if !strings.Contains(html, `aria-label="グローバルナビゲーション"`) {
		t.Error("GlobalNavTopBar に aria-label が無い")
	}
}

func TestGlobalNavBottomBar(t *testing.T) {
	t.Parallel()

	data := components.GlobalNavData{SignedIn: true, UserAtname: "alice", CurrentPageName: templates.PageNameHome}
	html := renderGlobalNav(t, components.GlobalNavBottomBar(data))

	// The bottom bar is shown only below md (the width GlobalNavTopBar starts rendering at) and
	// rendered as a bordered floating pill. Match the <nav>'s full class attribute so the assertion
	// stays pinned to the visibility pair with the top bar.
	//
	// [Ja] 下部バーは md (GlobalNavTopBar が描画され始める幅) 未満でのみ表示し、枠線付きのピルとして
	// 描画する。上部バーとの表示切り替えの対に固定するため <nav> の class 属性全体で照合する。
	for _, want := range []string{`<nav class="md:hidden"`, "rounded-full", "border", `href="/home"`} {
		if !strings.Contains(html, want) {
			t.Errorf("GlobalNavBottomBar に %q が含まれていない", want)
		}
	}

	// The wrapper is a <nav> landmark whose aria-label differs from the top bar's.
	// [Ja] ラッパーは <nav> ランドマークで、その aria-label は上部バーとは異なる。
	if !strings.Contains(html, "<nav") {
		t.Error("GlobalNavBottomBar に <nav> ランドマークが無い")
	}
	if !strings.Contains(html, `aria-label="グローバルナビゲーション (モバイル)"`) {
		t.Error("GlobalNavBottomBar に aria-label が無い")
	}
}
