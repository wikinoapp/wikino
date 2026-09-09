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
)

// renderBreadcrumbHeader renders c with a ja locale and returns the HTML.
//
// [Ja] renderBreadcrumbHeader は c を ja ロケールで描画し、HTML を返す。
func renderBreadcrumbHeader(t *testing.T, c templ.Component) string {
	t.Helper()

	ctx := i18n.SetLocale(context.Background(), "ja")

	var buf bytes.Buffer
	if err := c.Render(ctx, &buf); err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}
	return buf.String()
}

// signedInNav is the nav state the header renders its navigation bar from.
//
// [Ja] signedInNav はヘッダーがナビバーを描画するためのナビ状態。
var signedInNav = components.GlobalNavData{
	SignedIn:        true,
	UserAtname:      "alice",
	CurrentPageName: templates.PageNameHome,
}

func TestBreadcrumbHeader_RendersBreadcrumbWithoutSidebarToggle(t *testing.T) {
	t.Parallel()

	data := components.BreadcrumbHeaderData{
		MaxWidthClass: "max-w-3xl",
		Items: []components.BreadcrumbItem{
			{Label: "テストスペース", Path: templates.Path("/s/test")},
		},
	}
	html := renderBreadcrumbHeader(t, components.BreadcrumbHeader(components.BreadcrumbHeaderRenderData{Header: data, GlobalNav: signedInNav}))

	// The breadcrumb landmark and its item render.
	// [Ja] パンくずのランドマークとその項目が描画される。
	if !strings.Contains(html, `aria-label="パンくずリスト"`) {
		t.Error("breadcrumb nav aria-label not found")
	}
	if !strings.Contains(html, "テストスペース") {
		t.Error("breadcrumb item label not found")
	}

	// The removed sidebar toggle leaves no button or dispatch behind.
	// [Ja] 廃止したサイドバー開閉ボタンは、ボタンも dispatch も残さない。
	if strings.Contains(html, "basecoat:sidebar") {
		t.Error("sidebar toggle dispatch should not be rendered")
	}
	if strings.Contains(html, "サイドバーの開閉") {
		t.Error("sidebar toggle button should not be rendered")
	}
}

func TestBreadcrumbHeader_MarksCurrentItemWithoutLink(t *testing.T) {
	t.Parallel()

	data := components.BreadcrumbHeaderData{
		Items: []components.BreadcrumbItem{
			{Label: "テストスペース", Path: templates.Path("/s/test")},
			{Label: "現在のページ", Path: templates.Path("/s/test/pages/1"), IsCurrent: true},
		},
	}
	html := renderBreadcrumbHeader(t, components.BreadcrumbHeader(components.BreadcrumbHeaderRenderData{Header: data, GlobalNav: signedInNav}))

	if !strings.Contains(html, "aria-current=\"page\"") {
		t.Error("current breadcrumb item must carry aria-current")
	}
	if strings.Contains(html, "href=\"/s/test/pages/1\"") {
		t.Error("current breadcrumb item must not render as a link")
	}
	if !strings.Contains(html, "現在のページ") {
		t.Error("current breadcrumb label not found")
	}
}

// A crumb long enough to be cut carries its full label in a title attribute, so a pointer can
// bring back what the ellipsis took. A crumb drawn whole does not: there the tooltip would repeat
// text already on screen, and it would fire on every crumb the pointer crosses.
//
// Both a linked and a current crumb are checked, since the two are rendered by separate branches.
//
// [Ja] 切り詰められる長さの項目は、省略記号が奪った分をポインタで呼び戻せるよう、title 属性に
// ラベル全体を持つ。全体が描かれる項目は持たない。そこでのツールチップは画面に出ている文字列を
// 繰り返すだけで、ポインタが通り過ぎる項目ごとに出てしまうため。
//
// リンク付きの項目と現在地の項目は別の分岐で描画されるため、両方を確認する。
func TestBreadcrumbHeader_AddsTitleOnlyToLabelsThatMayBeTruncated(t *testing.T) {
	t.Parallel()

	// 30 characters is the longest name the model accepts, and 4 is well inside what max-w-48 draws.
	//
	// [Ja] 30 文字はモデルが受け付ける最長の名前で、4 文字は max-w-48 が描き切る範囲に十分収まる。
	longLabel := strings.Repeat("長", 30)
	shortLabel := "短い名前"

	data := components.BreadcrumbHeaderData{
		Items: []components.BreadcrumbItem{
			{Label: longLabel, Path: templates.Path("/s/long")},
			{Label: shortLabel, Path: templates.Path("/s/long/topics/1")},
			{Label: longLabel, Path: templates.Path("/s/long/topics/2"), IsCurrent: true},
		},
	}
	html := renderBreadcrumbHeader(t, components.BreadcrumbHeader(components.BreadcrumbHeaderRenderData{Header: data, GlobalNav: signedInNav}))

	if strings.Count(html, `title="`+longLabel+`"`) != 2 {
		t.Errorf("切り詰められる項目 (リンク付きと現在地) の両方に title を期待したが %q だった", html)
	}
	if strings.Contains(html, `title="`+shortLabel+`"`) {
		t.Error("全体が描かれる項目に title が付いている")
	}
}

// The separator between two crumbs is a visual affordance with no accessible name, so it stays out
// of the accessibility tree: otherwise a three-crumb trail is announced as a list of five items,
// two of them nameless.
//
// [Ja] 2 つの項目の間の区切りはアクセシブルネームを持たない視覚的な装飾のため、アクセシビリティ
// ツリーの外に置く。そうしないと 3 項目の経路が 5 項目のリストとして読み上げられ、うち 2 項目が
// 名前を持たない項目になる。
func TestBreadcrumbHeader_HidesSeparatorsFromAssistiveTechnology(t *testing.T) {
	t.Parallel()

	data := components.BreadcrumbHeaderData{
		Items: []components.BreadcrumbItem{
			{Label: "テストスペース", Path: templates.Path("/s/test")},
			{Label: "テストトピック", Path: templates.Path("/s/test/topics/1")},
			{Label: "現在のページ", IsCurrent: true},
		},
	}
	html := renderBreadcrumbHeader(t, components.BreadcrumbHeader(components.BreadcrumbHeaderRenderData{Header: data, GlobalNav: signedInNav}))

	if got, want := strings.Count(html, `<li class="leading-none" aria-hidden="true">`), 2; got != want {
		t.Errorf("hidden separator count = %d, want %d", got, want)
	}
	if strings.Contains(html, `<li class="leading-none">`) {
		t.Error("区切りがアクセシビリティツリーに残っている")
	}
}

// Breadcrumb icons repeat the accessible name supplied by the parent link or visible label, so
// every item icon is decorative. This covers the icon-only link, labeled link and current-item
// branches together.
//
// [Ja] パンくずのアイコンは親リンクまたは可視ラベルが伝えるアクセシブルネームと重複するため、すべて
// 装飾として扱う。アイコンのみのリンク・ラベル付きリンク・現在項目の 3 分岐をまとめて固定する。
func TestBreadcrumbHeader_HidesItemIconsFromAssistiveTechnology(t *testing.T) {
	t.Parallel()

	data := components.BreadcrumbHeaderData{
		Items: []components.BreadcrumbItem{
			{Path: templates.HomePath(), IconName: "house-regular", AriaLabel: "ホーム"},
			{Label: "テストスペース", Path: templates.Path("/s/test"), IconName: "folders-regular"},
			{Label: "現在のページ", IconName: "note-regular", IsCurrent: true},
		},
	}
	html := renderBreadcrumbHeader(t, components.BreadcrumbHeader(components.BreadcrumbHeaderRenderData{
		Header:        data,
		HideGlobalNav: true,
	}))

	const decorativeSVG = `<svg aria-hidden="true" focusable="false"`
	if got, want := strings.Count(html, "<svg"), 5; got != want {
		t.Fatalf("breadcrumb SVG count = %d, want %d", got, want)
	}
	if got, want := strings.Count(html, decorativeSVG), 5; got != want {
		t.Errorf("decorative breadcrumb SVG count = %d, want %d", got, want)
	}
	if strings.Contains(html, `<svg class="size-4 fill-gray-600"`) {
		t.Error("パンくず項目のアイコンがアクセシビリティツリーに残っている")
	}
}

func TestBreadcrumbHeader_DropsBreadcrumbWithoutNavigableItems(t *testing.T) {
	t.Parallel()

	// A trail holding only the current item gives no way back up the hierarchy, so it must not become
	// a navigation landmark with no links in it. A public screen whose parent is authenticated-only
	// reaches this state for signed-out viewers, once the home crumb is omitted.
	//
	// [Ja] 現在項目だけの経路は上位へ戻る手段を持たないため、リンクの無いナビゲーションランドマークに
	// なってはならない。親が認証必須の公開画面は、未ログインの閲覧者にホーム項目を出さなくなった結果
	// この状態になる。
	data := components.BreadcrumbHeaderData{
		MaxWidthClass: "max-w-3xl",
		Items: []components.BreadcrumbItem{
			{Label: "現在のスペース", IsCurrent: true},
		},
	}
	html := renderBreadcrumbHeader(t, components.BreadcrumbHeader(components.BreadcrumbHeaderRenderData{Header: data, GlobalNav: signedInNav}))

	if strings.Contains(html, `aria-label="パンくずリスト"`) {
		t.Error("たどれる項目が無い経路でパンくずランドマークを描画するべきではない")
	}
	if strings.Contains(html, "現在のスペース") {
		t.Error("落としたパンくずの項目が残っている")
	}

	// The bar is the header's only content there, so the header switches with the bar just as it does
	// on a screen with no items at all.
	//
	// The header owns its top spacing, so the spacing sits on the same element as the visibility
	// class and goes away with the header below the breakpoint.
	//
	// [Ja] その状態ではバーがヘッダーの唯一の中身になるため、項目がまったく無い画面と同じくヘッダーは
	// バーと一緒に切り替わる。
	//
	// 上余白はヘッダー自身が持つため、表示切り替えクラスと同じ要素に付き、ブレークポイント未満では
	// ヘッダーと一緒に消える。
	if !strings.Contains(html, `<header class="pt-4 hidden md:block">`) {
		t.Error("ヘッダーがナビバーと同じ幅で切り替わっていない")
	}
	if !strings.Contains(html, `aria-label="グローバルナビゲーション"`) {
		t.Error("ナビバーが描画されていない")
	}
}

func TestHomeBreadcrumbItems(t *testing.T) {
	t.Parallel()

	// /home is behind authentication, so a screen reachable without signing in must not offer it as
	// the trail's root. Centralizing the condition keeps every public screen from having to remember
	// it.
	//
	// [Ja] /home は認証必須のため、未ログインでも到達できる画面が経路の起点として提示してはいけない。
	// 条件を 1 箇所にまとめることで、公開画面ごとに覚えておく必要が無くなる。
	ctx := i18n.SetLocale(context.Background(), "ja")

	if got := components.HomeBreadcrumbItems(ctx, false); len(got) != 0 {
		t.Errorf("未ログインではホーム項目を返すべきではない: %+v", got)
	}

	got := components.HomeBreadcrumbItems(ctx, true)
	if len(got) != 1 {
		t.Fatalf("len(HomeBreadcrumbItems()) = %d, want 1", len(got))
	}
	if got[0].Path != templates.HomePath() {
		t.Errorf("Path = %q, want %q", got[0].Path, templates.HomePath())
	}
	// The crumb is icon-only, so its accessible name comes from the localized aria-label.
	//
	// [Ja] この項目はアイコンのみのため、アクセシブルネームはローカライズされた aria-label が供給する。
	if got[0].AriaLabel != "ホーム" {
		t.Errorf("AriaLabel = %q, want %q", got[0].AriaLabel, "ホーム")
	}
	if got[0].IconName == "" {
		t.Error("ホーム項目にアイコンが設定されていない")
	}
}

func TestBreadcrumbHeader_PlacesBreadcrumbLeftAndNavRight(t *testing.T) {
	t.Parallel()

	data := components.BreadcrumbHeaderData{
		MaxWidthClass: "max-w-3xl",
		Items: []components.BreadcrumbItem{
			{Label: "テストスペース", Path: templates.Path("/s/test")},
		},
	}
	html := renderBreadcrumbHeader(t, components.BreadcrumbHeader(components.BreadcrumbHeaderRenderData{Header: data, GlobalNav: signedInNav}))

	// One container as wide as the screen's content holds both sides, so the breadcrumb's left edge
	// and the navigation bar's right edge line up with the body. The breadcrumb may wrap (min-w-0)
	// rather than squeeze the bar (shrink-0).
	//
	// [Ja] 本文と同じ幅のコンテナ 1 つが両側を収めるため、パンくずの左端とナビバーの右端が本文と
	// 揃う。パンくずはバーを潰さず (shrink-0) 折り返せる (min-w-0)。
	const container = `<div class="max-w-3xl mx-auto flex w-full items-center justify-between gap-2 px-4">`
	if !strings.Contains(html, container) {
		t.Errorf("ヘッダーのコンテナ %q が含まれていない", container)
	}
	if !strings.Contains(html, `<div class="min-w-0">`) {
		t.Error("パンくず側に min-w-0 が付いていない")
	}
	if !strings.Contains(html, "shrink-0") {
		t.Error("ナビバー側に shrink-0 が付いていない")
	}

	// The breadcrumb comes first and the navigation bar last, and the flex-1 spacers that used to
	// center the breadcrumb are gone.
	//
	// [Ja] パンくずが先・ナビバーが後に並び、パンくずを中央寄せしていた flex-1 のスペーサーは無い。
	breadcrumb, nav := strings.Index(html, `aria-label="パンくずリスト"`), strings.Index(html, `aria-label="グローバルナビゲーション"`)
	if breadcrumb == -1 || nav == -1 || breadcrumb > nav {
		t.Errorf("breadcrumb (index %d) must precede the navigation bar (index %d)", breadcrumb, nav)
	}
	if strings.Contains(html, "flex-1") {
		t.Error("中央寄せ用の flex-1 スペーサーが残っている")
	}
}

func TestBreadcrumbHeader_RendersNavWithoutBreadcrumbItems(t *testing.T) {
	t.Parallel()

	// Render data without breadcrumb items that remains in the global navigation still gets the bar,
	// and the empty left slot must not become a breadcrumb landmark with nothing in it. The bar is the
	// header's only content there, so the header switches with the bar and no banner landmark is left
	// with nothing in it below the breakpoint.
	//
	// [Ja] パンくず項目を持たず、グローバルナビの対象である描画データにもナビバーは出す。空の左側は
	// 中身の無いパンくずランドマークにしない。その状態ではバーがヘッダーの唯一の中身になるため、
	// ヘッダーはバーと一緒に切り替わり、ブレークポイント未満で中身の無い banner ランドマークが
	// 残らない。
	data := components.BreadcrumbHeaderData{MaxWidthClass: "max-w-5xl"}
	html := renderBreadcrumbHeader(t, components.BreadcrumbHeader(components.BreadcrumbHeaderRenderData{Header: data, GlobalNav: signedInNav}))

	// The header owns its top spacing, so the spacing sits on the same element as the visibility
	// class and goes away with the header below the breakpoint.
	//
	// [Ja] 上余白はヘッダー自身が持つため、表示切り替えクラスと同じ要素に付き、ブレークポイント未満では
	// ヘッダーと一緒に消える。
	if !strings.Contains(html, `<header class="pt-4 hidden md:block">`) {
		t.Error("ヘッダーがナビバーと同じ幅で切り替わっていない")
	}
	if !strings.Contains(html, `aria-label="グローバルナビゲーション"`) {
		t.Error("ナビバーが描画されていない")
	}
	if strings.Contains(html, `aria-label="パンくずリスト"`) {
		t.Error("項目が無いときにパンくずランドマークを描画するべきではない")
	}
}

func TestBreadcrumbHeader_DropsNavWhenHidden(t *testing.T) {
	t.Parallel()

	// A screen taken out of the global navigation keeps its breadcrumb: the header still has content,
	// so it renders with the left slot filled and the navigation bar dropped.
	//
	// [Ja] グローバルナビの対象外にした画面でもパンくずは残る。ヘッダーには中身があるため、左側を埋め・
	// ナビバーを落とした状態で描画される。
	data := components.BreadcrumbHeaderData{
		MaxWidthClass: "max-w-3xl",
		Items: []components.BreadcrumbItem{
			{Label: "テストスペース", Path: templates.Path("/s/test")},
		},
	}
	html := renderBreadcrumbHeader(t, components.BreadcrumbHeader(components.BreadcrumbHeaderRenderData{
		Header:        data,
		GlobalNav:     signedInNav,
		HideGlobalNav: true,
	}))

	if !strings.Contains(html, `aria-label="パンくずリスト"`) {
		t.Error("ナビバーを落としてもパンくずは描画されるべき")
	}
	if strings.Contains(html, `aria-label="グローバルナビゲーション"`) {
		t.Error("HideGlobalNav 指定時にナビバーが描画されている")
	}
}

func TestBreadcrumbHeader_RendersNothingWithoutContent(t *testing.T) {
	t.Parallel()

	// Both of the header's slots are empty on a screen that has no breadcrumb items and is out of the
	// global navigation, so the component emits nothing instead of a banner landmark with nothing in
	// it. The guard lives in the component, so no caller has to remember it.
	//
	// [Ja] パンくず項目を持たず、グローバルナビの対象外である画面ではヘッダーの両方の枠が空になるため、
	// コンポーネントは中身の無い banner ランドマークではなく何も出力しない。ガードはコンポーネント側に
	// あるため、呼び出し側が覚えておく必要は無い。
	html := renderBreadcrumbHeader(t, components.BreadcrumbHeader(components.BreadcrumbHeaderRenderData{
		GlobalNav:     signedInNav,
		HideGlobalNav: true,
	}))

	if html != "" {
		t.Errorf("描画されるべきでないヘッダーが出力されている: %q", html)
	}
}

func TestBreadcrumbHeaderRenderData_ShouldRender(t *testing.T) {
	t.Parallel()

	// The header is worth a banner landmark as long as one of its two slots has content. Only a
	// screen with no navigable breadcrumb item that is also out of the global navigation leaves both
	// empty: a trail holding just the current item does not fill the left slot, because the
	// breadcrumb is dropped rather than rendered without links.
	//
	// [Ja] ヘッダーは 2 つの枠のどちらかに中身がある限り banner ランドマークに値する。両方が空になるのは、
	// たどれるパンくず項目が無く、かつグローバルナビの対象外である画面だけ。現在項目だけの経路は左側を
	// 埋めない。リンクの無いパンくずを描画せず落とすためである。
	items := []components.BreadcrumbItem{{Label: "テストスペース", Path: templates.Path("/s/test")}}
	currentOnlyItems := []components.BreadcrumbItem{{Label: "現在のスペース", IsCurrent: true}}

	tests := []struct {
		name string
		data components.BreadcrumbHeaderRenderData
		want bool
	}{
		{
			name: "パンくずとナビバーの両方がある",
			data: components.BreadcrumbHeaderRenderData{Header: components.BreadcrumbHeaderData{Items: items}, GlobalNav: signedInNav},
			want: true,
		},
		{
			name: "ナビバーだけがある",
			data: components.BreadcrumbHeaderRenderData{GlobalNav: signedInNav},
			want: true,
		},
		{
			name: "パンくずだけがある",
			data: components.BreadcrumbHeaderRenderData{Header: components.BreadcrumbHeaderData{Items: items}, HideGlobalNav: true},
			want: true,
		},
		{
			name: "どちらも無い",
			data: components.BreadcrumbHeaderRenderData{HideGlobalNav: true},
			want: false,
		},
		{
			name: "現在項目だけのパンくずしか無い",
			data: components.BreadcrumbHeaderRenderData{Header: components.BreadcrumbHeaderData{Items: currentOnlyItems}, HideGlobalNav: true},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.data.ShouldRender(); got != tt.want {
				t.Errorf("ShouldRender() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBreadcrumbHeaderData_ResolvedMaxWidthClass(t *testing.T) {
	t.Parallel()

	// Every screen is expected to pass its own content width, but an empty value must still resolve
	// to a width. Rendering the container with no max-width at all would stretch the header to the
	// viewport edge and pull the navigation bar's right edge out of line with the body.
	//
	// [Ja] 各画面が自身の本文幅を渡す前提だが、空の値でも幅に解決されなければならない。max-width の
	// 付かないコンテナを描画すると、ヘッダーがビューポート端まで伸びてナビバーの右端が本文と
	// 揃わなくなる。
	tests := []struct {
		name string
		data components.BreadcrumbHeaderData
		want string
	}{
		{name: "画面が渡した幅をそのまま使う", data: components.BreadcrumbHeaderData{MaxWidthClass: "max-w-6xl"}, want: "max-w-6xl"},
		{name: "空なら既定幅にフォールバックする", data: components.BreadcrumbHeaderData{}, want: "max-w-3xl"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.data.ResolvedMaxWidthClass(); got != tt.want {
				t.Errorf("ResolvedMaxWidthClass() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBreadcrumbHeader_FallsBackToDefaultMaxWidth(t *testing.T) {
	t.Parallel()

	// A screen that omits MaxWidthClass still gets a bounded container, so the header never renders
	// with a max-width-less container.
	//
	// [Ja] MaxWidthClass を渡さない画面でも幅の付いたコンテナになるため、max-width の無いコンテナで
	// ヘッダーが描画されることはない。
	data := components.BreadcrumbHeaderData{
		Items: []components.BreadcrumbItem{
			{Label: "テストスペース", Path: templates.Path("/s/test")},
		},
	}
	html := renderBreadcrumbHeader(t, components.BreadcrumbHeader(components.BreadcrumbHeaderRenderData{Header: data, GlobalNav: signedInNav}))

	const container = `<div class="max-w-3xl mx-auto flex w-full items-center justify-between gap-2 px-4">`
	if !strings.Contains(html, container) {
		t.Errorf("ヘッダーのコンテナ %q が含まれていない", container)
	}
	if strings.Contains(html, `<div class=" mx-auto`) {
		t.Error("max-width の無いコンテナが描画されている")
	}
}

func TestBreadcrumbHeader_KeepsHeaderVisibleWithBreadcrumbItems(t *testing.T) {
	t.Parallel()

	// With breadcrumb items the header has content at every width, so it must not take the
	// navigation bar's visibility class: the breadcrumb has to stay visible below the breakpoint,
	// where only the bar is hidden.
	//
	// [Ja] パンくず項目があるとヘッダーはどの幅でも中身を持つため、ナビバーの表示切り替えクラスを
	// 持ってはならない。ブレークポイント未満で隠れるのはバーだけで、パンくずは表示され続ける必要が
	// ある。
	data := components.BreadcrumbHeaderData{
		MaxWidthClass: "max-w-3xl",
		Items: []components.BreadcrumbItem{
			{Label: "テストスペース", Path: templates.Path("/s/test")},
		},
	}
	html := renderBreadcrumbHeader(t, components.BreadcrumbHeader(components.BreadcrumbHeaderRenderData{Header: data, GlobalNav: signedInNav}))

	if !strings.Contains(html, `<header class="pt-4">`) {
		t.Error("パンくずがあるヘッダーに表示切り替えクラスが付いている")
	}
}
