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

// renderGlobalNavはcをjaロケールで描画し、HTMLを返す。
func renderGlobalNav(t *testing.T, c templ.Component) string {
	t.Helper()

	ctx := i18n.SetLocale(context.Background(), "ja")

	var buf bytes.Buffer
	if err := c.Render(ctx, &buf); err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}
	return buf.String()
}

// anchorSegmentは指定したhrefを含む <a> 要素の範囲を切り出して返す。
// その1リンク (aria-current・内側のアイコンなど) を単独で検証するために使う。
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

// svgSegmentはhtml内の最初の完全なSVG要素を返す。
func svgSegment(html string) string {
	return elementSegment(html, "<svg", "</svg>")
}

func TestGlobalNavMenu_SignedInLinks(t *testing.T) {
	t.Parallel()

	data := components.GlobalNavData{
		SignedIn:        true,
		UserAtname:      "alice",
		CurrentPageName: templates.PageNameHome,
	}
	html := renderGlobalNav(t, components.GlobalNavMenu(data, ""))

	// ログイン時はホーム / 検索 / プロフィールが出て、サインインリンクは出ない。
	for _, want := range []string{`href="/home"`, `href="/search"`, `href="/@alice"`} {
		if !strings.Contains(html, want) {
			t.Errorf("GlobalNavMenuにリンク%qが含まれていない", want)
		}
	}
	if strings.Contains(html, "/sign_in") {
		t.Error("ログイン時にサインインリンクが表示されるべきではない")
	}

	// 3項目それぞれがアイコンを1つずつ描画する。
	if got := strings.Count(html, "<svg"); got != 3 {
		t.Errorf("svgの数 = %d、期待値 = 3", got)
	}
}

func TestGlobalNavMenu_SignedOutLinks(t *testing.T) {
	t.Parallel()

	// 未ログインの訪問者が公開スペースを閲覧している状況を置く。未ログイン時の項目はこのために
	// 存在する。
	data := components.GlobalNavData{}
	html := renderGlobalNav(t, components.GlobalNavMenu(data, ""))

	// 未ログイン時はホーム (ルート) とサインインのみが出る。
	for _, want := range []string{`href="/"`, `href="/sign_in"`} {
		if !strings.Contains(html, want) {
			t.Errorf("GlobalNavMenuにリンク%qが含まれていない", want)
		}
	}

	// 未ログイン時の項目はいずれもアクティブにならない。ホームリンクが指すトップページはナビ自体を
	// 描画しないため、これらの項目が出る画面はどれも項目の指す画面ではない。
	if strings.Contains(html, `aria-current="page"`) {
		t.Error("未ログイン時にアクティブ項目があるべきではない")
	}

	// 検索・プロフィールはログイン必須のため出さない。
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
		t.Errorf("svgの数 = %d、期待値 = 2", got)
	}
}

func TestGlobalNavMenu_ActiveHighlight(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		pageName   templates.PageName
		activeHref string
	}{
		{name: "homeがアクティブ", pageName: templates.PageNameHome, activeHref: "/home"},
		{name: "searchがアクティブ", pageName: templates.PageNameSearch, activeHref: "/search"},
		{name: "profileがアクティブ", pageName: templates.PageNameProfile, activeHref: "/@alice"},
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

			// 現在ページとしてマークされる項目は1つだけ。
			if got := strings.Count(html, `aria-current="page"`); got != 1 {
				t.Errorf("aria-currentの数 = %d、期待値 = 1", got)
			}

			// アクティブなマーカーはアクティブ項目のリンク内に存在しなければならない。
			seg := anchorSegment(html, tt.activeHref)
			if seg == "" {
				t.Fatalf("%qのアンカーが見つからない", tt.activeHref)
			}
			if !strings.Contains(seg, `aria-current="page"`) {
				t.Errorf("%qのリンクがアクティブとして描画されていない", tt.activeHref)
			}
		})
	}
}

func TestGlobalNavMenu_NoActiveWhenUnmatched(t *testing.T) {
	t.Parallel()

	// どの項目にも対応しないページ (トピックページなど) では、いずれの項目もアクティブにしない。
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
				t.Error("アクティブ時のSVGが期待値と異なる")
			}
			if inactiveSVG != wantInactiveSVG {
				t.Error("非アクティブ時のSVGが期待値と異なる")
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
					t.Errorf("%qのリンクにaria-label %qが含まれていない", link.href, link.label)
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
				t.Errorf("aria-hiddenを持つSVGの数 = %d、期待値 = %d", got, tt.iconCount)
			}
			if got := strings.Count(html, `focusable="false"`); got != tt.iconCount {
				t.Errorf("focusable=falseを持つSVGの数 = %d、期待値 = %d", got, tt.iconCount)
			}
		})
	}
}

func TestGlobalNavMenu_SpaceFilterSearchPath(t *testing.T) {
	t.Parallel()

	// スペース内では検索リンクがそのスペースに絞り込まれる。
	data := components.GlobalNavData{SignedIn: true, UserAtname: "alice", SpaceIdentifier: "my-space"}
	html := renderGlobalNav(t, components.GlobalNavMenu(data, ""))

	if !strings.Contains(html, "space:my-space") {
		t.Error("スペースフィルター付きの検索パスが含まれていない")
	}
}

func TestGlobalNavMenu_ItemsRoundedFull(t *testing.T) {
	t.Parallel()

	// 各項目リンクは完全な円 (rounded-full) で、hoverハイライトが円形に見える。
	// リンク自身の角丸だけを見るため、コンテナのclassNameを空にして描画する。
	data := components.GlobalNavData{SignedIn: true, UserAtname: "alice", CurrentPageName: templates.PageNameHome}
	html := renderGlobalNav(t, components.GlobalNavMenu(data, ""))

	seg := anchorSegment(html, "/home")
	if seg == "" {
		t.Fatal(`"/home" のアンカーが見つからない`)
	}
	if !strings.Contains(seg, "rounded-full") {
		t.Error("ナビ項目のリンクはrounded-fullであるべき")
	}
	if strings.Contains(seg, "rounded-md") {
		t.Error("ナビ項目のリンクにrounded-mdが残っている")
	}
}

func TestGlobalNavMenu_EnglishLocale(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), "en")
	data := components.GlobalNavData{SignedIn: true, UserAtname: "alice", CurrentPageName: templates.PageNameHome}

	var buf bytes.Buffer
	if err := components.GlobalNavMenu(data, "").Render(ctx, &buf); err != nil {
		t.Fatalf("描画に失敗: %v", err)
	}
	html := buf.String()

	for _, want := range []string{`aria-label="Home"`, `aria-label="Search"`, `aria-label="Profile"`} {
		if !strings.Contains(html, want) {
			t.Errorf("GlobalNavMenuの出力に%qが含まれていない", want)
		}
	}
}

func TestGlobalNavTopBar(t *testing.T) {
	t.Parallel()

	data := components.GlobalNavData{SignedIn: true, UserAtname: "alice", CurrentPageName: templates.PageNameHome}
	html := renderGlobalNav(t, components.GlobalNavTopBar(data))

	// 上部バーはヘッダーの通常フローに入り、md (GlobalNavBottomBarが描画されなくなる幅) 以上で
	// のみ表示し、長いパンくずの隣でもアイコンを潰さない。自身のピルは持たないため、本文にオーバー
	// レイするものは無い。下部バーとの表示切り替えの対に固定するためclass属性全体で照合する。
	for _, want := range []string{`class="shrink-0 hidden md:flex"`, `href="/home"`} {
		if !strings.Contains(html, want) {
			t.Errorf("GlobalNavTopBarに%qが含まれていない", want)
		}
	}
	// TailwindはGoの文字列リテラルも走査するため、削除済みレールのtransformは断片から
	// 組み立てる。完全なクラスをここに書くと、この否定確認だけのために不要なutilityが生成される。
	removedRailTransform := strings.Join([]string{"-translate-y", "1/2"}, "-")
	for _, notWant := range []string{"fixed", removedRailTransform, "bg-card", "border"} {
		if strings.Contains(html, notWant) {
			t.Errorf("GlobalNavTopBarに浮遊ピルのクラス%qが残っている", notWant)
		}
	}

	// ラッパーは固有のaria-labelを持つ <nav> ランドマーク。
	if !strings.Contains(html, "<nav") {
		t.Error("GlobalNavTopBarに <nav> ランドマークが無い")
	}
	if !strings.Contains(html, `aria-label="グローバルナビゲーション"`) {
		t.Error("GlobalNavTopBarにaria-labelが無い")
	}
}

func TestGlobalNavBottomBar(t *testing.T) {
	t.Parallel()

	data := components.GlobalNavData{SignedIn: true, UserAtname: "alice", CurrentPageName: templates.PageNameHome}
	html := renderGlobalNav(t, components.GlobalNavBottomBar(data))

	// 下部バーはmd (GlobalNavTopBarが描画され始める幅) 未満でのみ表示し、枠線付きのピルとして
	// 描画する。上部バーとの表示切り替えの対に固定するため <nav> のclass属性全体で照合する。
	for _, want := range []string{`<nav class="md:hidden"`, "rounded-full", "border", `href="/home"`} {
		if !strings.Contains(html, want) {
			t.Errorf("GlobalNavBottomBarに%qが含まれていない", want)
		}
	}

	// ラッパーは <nav> ランドマークで、そのaria-labelは上部バーとは異なる。
	if !strings.Contains(html, "<nav") {
		t.Error("GlobalNavBottomBarに <nav> ランドマークが無い")
	}
	if !strings.Contains(html, `aria-label="グローバルナビゲーション (モバイル)"`) {
		t.Error("GlobalNavBottomBarにaria-labelが無い")
	}
}
