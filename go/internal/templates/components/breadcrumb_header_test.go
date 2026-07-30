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

	if !strings.Contains(html, `<header class="hidden md:block">`) {
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
	// screen with no breadcrumb items that is also out of the global navigation leaves both empty.
	//
	// [Ja] ヘッダーは 2 つの枠のどちらかに中身がある限り banner ランドマークに値する。両方が空になるのは、
	// パンくず項目が無く、かつグローバルナビの対象外である画面だけ。
	items := []components.BreadcrumbItem{{Label: "テストスペース", Path: templates.Path("/s/test")}}

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

	if !strings.Contains(html, "<header>") {
		t.Error("パンくずがあるヘッダーに表示切り替えクラスが付いている")
	}
}
