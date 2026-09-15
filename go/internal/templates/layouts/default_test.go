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

// renderDefaultはdefaultレイアウトをjaロケール・指定データで描画し、HTMLを返す。
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
			t.Errorf("Defaultレイアウトの出力に%qが含まれていない", want)
		}
	}

	// PC向けのナビはヘッダーの行の一部のため、メインコンテンツの上に固定されるものは無い。
	// TailwindはGoの文字列リテラルも走査するため、削除済みレールの固定配置クラスは断片から組み
	// 立てる。完全なクラスをここに書くと、この確認だけのために不要なutilityが生成される。
	removedRailPosition := `class="fixed ` + strings.Join([]string{"left", "2"}, "-")
	if strings.Contains(html, removedRailPosition) {
		t.Error("デスクトップのナビゲーションがメインコンテンツの上に固定されている")
	}

	// 旧off-canvasサイドバーとその開閉ボタンは廃止され、グローバルナビが置き換える。
	if strings.Contains(html, `id="sidebar"`) {
		t.Error("旧サイドバーの要素が描画されている")
	}
	if strings.Contains(html, "basecoat:sidebar") {
		t.Error("旧サイドバーの開閉イベントの発行が描画されている")
	}

	// 上部バーと下部バーは同じメニューを埋め込むため、ホームリンクはちょうど2回描画される。
	if got := strings.Count(html, `href="/home"`); got != 2 {
		t.Errorf(`href="/home"の数 = %d、期待値 = 2 (上部バーと下部バー)`, got)
	}

	// スキップリンクは最初のフォーカス可能要素でなければならないため、DOM上でナビのリンクより
	// 前に現れる必要がある。
	if skip, nav := strings.Index(html, `href="#main"`), strings.Index(html, `href="/home"`); skip == -1 || nav == -1 || skip > nav {
		t.Errorf("スキップリンク (index %d) がナビゲーションのリンク (index %d) より前にない", skip, nav)
	}
}

func TestDefault_SignedOutNav(t *testing.T) {
	t.Parallel()

	// 未ログイン時の項目が出るのは公開スペースの匿名閲覧。
	data := layouts.DefaultLayoutData{
		Meta:      viewmodel.PageMeta{Title: "テストタイトル"},
		GlobalNav: components.GlobalNavData{},
	}
	html := renderDefault(t, data, templ.Raw(""))

	// 未ログインの訪問者は上部バー・下部バーともにホーム (ルート) + サインインが出る。
	if got := strings.Count(html, `href="/sign_in"`); got != 2 {
		t.Errorf(`href="/sign_in"の数 = %d、期待値 = 2 (上部バーと下部バー)`, got)
	}
	if strings.Contains(html, `href="/search"`) {
		t.Error("ログアウト中なのに検索リンクが表示されている")
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
		t.Error("レイアウトがパンくずリストのヘッダーを描画していない")
	}
	if !strings.Contains(html, "テストスペース") {
		t.Error("パンくずリストの項目のラベルが見つからない")
	}

	// ヘッダーは <main> の外になければならない。スキップリンクの飛び先は #mainのため、
	// <main> の内側にあるヘッダーは飛ばされず着地先の内側に入ってしまう。
	header, main := strings.Index(html, `aria-label="パンくずリスト"`), strings.Index(html, `<main id="main"`)
	if header == -1 || main == -1 || header > main {
		t.Errorf("パンくずリストのヘッダー (index %d) が<main> (index %d) より前にない", header, main)
	}
}

func TestDefault_RendersBreadcrumbListStructuredData(t *testing.T) {
	t.Parallel()

	data := layouts.DefaultLayoutData{
		Meta: viewmodel.PageMeta{Title: "テストタイトル"},
		BreadcrumbHeader: components.BreadcrumbHeaderData{
			StructuredDataBaseURL: "https://example.com",
			Items: []components.BreadcrumbItem{
				{IconName: "house-regular", AriaLabel: "ホーム", Path: templates.Path("/home")},
				{Label: "テストスペース", Path: templates.Path("/s/test")},
				{Label: "現在のページ", IsCurrent: true},
			},
		},
	}
	html := renderDefault(t, data, templ.Raw(""))

	for _, want := range []string{
		"<script type=\"application/ld+json\">",
		"\"@type\":\"BreadcrumbList\"",
		"\"position\":1,\"name\":\"ホーム\",\"item\":\"https://example.com/home\"",
		"\"position\":2,\"name\":\"テストスペース\",\"item\":\"https://example.com/s/test\"",
		"\"position\":3,\"name\":\"現在のページ\"",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("構造化データに%qが含まれていない", want)
		}
	}
	if strings.Contains(html, "\"name\":\"現在のページ\",\"item\":") {
		t.Error("現在のパンくずの構造化データの項目が自身へリンクしている")
	}
}

// 構造化データはオプトイン。このレイアウトを使う画面の大半はStructuredDataBaseURLを空の
// ままにしており、パンくずのラベルとURLを機械可読なデータとして出してはならない。
func TestDefault_OmitsBreadcrumbListStructuredDataWithoutBaseURL(t *testing.T) {
	t.Parallel()

	data := layouts.DefaultLayoutData{
		Meta: viewmodel.PageMeta{Title: "テストタイトル"},
		BreadcrumbHeader: components.BreadcrumbHeaderData{
			Items: []components.BreadcrumbItem{
				{IconName: "house-regular", AriaLabel: "ホーム", Path: templates.Path("/home")},
				{Label: "非公開スペース", Path: templates.Path("/s/private")},
				{Label: "現在のページ", IsCurrent: true},
			},
		},
	}
	html := renderDefault(t, data, templ.Raw(""))

	for _, notWant := range []string{
		"application/ld+json",
		"BreadcrumbList",
	} {
		if strings.Contains(html, notWant) {
			t.Errorf("構造化データに%qが含まれている", notWant)
		}
	}

	// 見た目のパンくずは描画され、落ちるのは機械可読な複製だけである。
	if !strings.Contains(html, "非公開スペース") {
		t.Error("パンくずリストの項目のラベルが見つからない")
	}
}

// 構造化データは見た目のパンくずを写したものなので、パンくずを描画しない経路では構造化データも
// 出さない。親が認証必須しか無い公開画面は、未ログインの閲覧者に対してこの状態になる。
func TestDefault_OmitsBreadcrumbListStructuredDataWithoutNavigableItems(t *testing.T) {
	t.Parallel()

	data := layouts.DefaultLayoutData{
		Meta: viewmodel.PageMeta{Title: "テストタイトル"},
		BreadcrumbHeader: components.BreadcrumbHeaderData{
			StructuredDataBaseURL: "https://example.com",
			Items: []components.BreadcrumbItem{
				{Label: "現在のスペース", IsCurrent: true},
			},
		},
	}
	html := renderDefault(t, data, templ.Raw(""))

	for _, notWant := range []string{
		"application/ld+json",
		"BreadcrumbList",
		"現在のスペース",
	} {
		if strings.Contains(html, notWant) {
			t.Errorf("出力に%qが含まれている", notWant)
		}
	}
}

func TestDefault_RendersSharedHeaderWithoutBreadcrumbItems(t *testing.T) {
	t.Parallel()

	// ログイン済みホームはパンくず項目を持たないが、PC向けナビバーのためにヘッダーを描画する。
	// 左側は中身の無いパンくずランドマークにはせず空のままにし、ヘッダーは持っているバーと一緒に
	// 切り替わるため、ブレークポイント未満で中身の無いbannerランドマークが残らない。
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

	if !strings.Contains(html, `<header class="pt-4 hidden md:block">`) {
		t.Error("レイアウトがナビゲーションバーと切り替わる共通ヘッダーを描画していない")
	}
	if strings.Contains(html, `aria-label="パンくずリスト"`) {
		t.Error("パンくずリストの項目が無いのにレイアウトがパンくずリストのランドマークを描画している")
	}

	// 上余白はヘッダー自身が持つため、レイアウトはラッパーで包んではならない。ラッパーはヘッダーが
	// 自身を隠す画面で、本文の上に空白だけとなって残ってしまう。
	if strings.Contains(html, `<div class="pt-4">`) {
		t.Error("レイアウトが共通ヘッダーを独自の余白で囲んでいる")
	}

	header, main := strings.Index(html, "<header"), strings.Index(html, `<main id="main"`)
	if header == -1 || main == -1 || header > main {
		t.Errorf("共通のヘッダー (位置%d) が <main> (位置%d) より前にない", header, main)
	}
}

func TestDefault_ContentBottomPaddingMatchesBottomBar(t *testing.T) {
	t.Parallel()

	// 下部余白は最下部のコンテンツが固定の下部バーに隠れないためのものなので、バーが描画され
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
		t.Errorf("content wrapperに%qが含まれていない", wantClass)
	}
	if !strings.Contains(html, `<nav class="md:hidden"`) {
		t.Error("下部バーが余白の解除と同じmdで切り替わっていない")
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

	// パンくず項目が無い画面ではヘッダーの中身はナビバーだけなので、HideNavigationはヘッダーごと
	// 落とし (中身の無いbannerランドマークを残さない)、下部バーとその固定ラッパーも落とす。
	shownOnly := []string{
		"<header",
		`aria-label="グローバルナビゲーション"`,
		`aria-label="グローバルナビゲーション (モバイル)"`,
		`class="fixed bottom-2`,
	}
	for _, want := range shownOnly {
		if !strings.Contains(shown, want) {
			t.Errorf("既定では%qが描画されるべき", want)
		}
		if strings.Contains(hidden, want) {
			t.Errorf("HideNavigation指定時に%qが描画されている", want)
		}
	}

	// 下部余白は下部バーにコンテンツが隠れないためだけのものなので、バーと一緒に外れる。
	if !strings.Contains(shown, "md:pb-0") {
		t.Error("既定では下部バー分の余白が確保されるべき")
	}
	if strings.Contains(hidden, "md:pb-0") {
		t.Error("HideNavigation指定時は下部バー分の余白を確保しないべき")
	}

	// スキップリンクはヘッダーに追従する。このケースはパンくずもグローバルナビも無いため、
	// HideNavigationによってヘッダーとスキップリンクが落ちる。<main> ランドマークはどちらでも
	// 残る。ナビの部品ではなくページの主要領域だからである。
	if !strings.Contains(shown, `href="#main"`) {
		t.Error("既定ではスキップリンクが描画されるべき")
	}
	if strings.Contains(hidden, `href="#main"`) {
		t.Error("HideNavigation指定時に飛ばす対象の無いスキップリンクが描画されている")
	}
	if !strings.Contains(hidden, `<main id="main" tabindex="-1">`) {
		t.Error("HideNavigation指定時もmainランドマークは残るべき")
	}
}

func TestDefault_HideNavigationKeepsHeaderWithBreadcrumbItems(t *testing.T) {
	t.Parallel()

	// パンくず項目を持つ画面は、グローバルナビの対象外でもヘッダーを保つ。ヘッダーには中身が残って
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
		t.Error("HideNavigation指定時にナビバーが描画されている")
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

	// フッターは既定で描画され、HideFooter指定時 (編集画面など) は描画されない。
	if !strings.Contains(shown, "<footer") {
		t.Error("デフォルトでフッターが描画されていない")
	}
	if strings.Contains(hidden, "<footer") {
		t.Error("HideFooterを指定したのにフッターが隠れていない")
	}
}
