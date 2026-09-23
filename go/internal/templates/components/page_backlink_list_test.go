package components_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

func TestPageBacklinkList_WithBacklinks(t *testing.T) {
	t.Parallel()

	title := "リンク元ページ"
	pages := []*model.Page{
		{
			Number: 10,
			Title:  &title,
		},
	}

	data := viewmodel.NewBacklinkList(viewmodel.NewBacklinkListInput{
		Pages:           pages,
		SpaceIdentifier: model.SpaceIdentifier("my-space"),
	})

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	var buf bytes.Buffer
	err := components.PageBacklinkList(data).Render(ctx, &buf)
	if err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}

	html := buf.String()

	// リンク先のタイトルが表示されること
	if !strings.Contains(html, "リンク元ページ") {
		t.Error("バックリンクのタイトルが表示されていない")
	}

	// リンクのhrefが正しいこと
	if !strings.Contains(html, "/s/my-space/pages/10") {
		t.Error("バックリンクのリンク先が正しくない")
	}
}

func TestPageBacklinkList_Empty(t *testing.T) {
	t.Parallel()

	data := viewmodel.NewBacklinkList(viewmodel.NewBacklinkListInput{
		Pages:           nil,
		SpaceIdentifier: model.SpaceIdentifier("my-space"),
	})

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	var buf bytes.Buffer
	err := components.PageBacklinkList(data).Render(ctx, &buf)
	if err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}

	html := buf.String()

	// 見出し (h2) は呼び出し側に移したため、コンポーネント単体では描画されない。
	if strings.TrimSpace(html) != "" {
		t.Errorf("バックリンクが空なのにHTMLが空ではない: %q", html)
	}
}

func TestPageBacklinkList_UntitledPage(t *testing.T) {
	t.Parallel()

	pages := []*model.Page{
		{
			Number: 5,
			Title:  nil,
		},
	}

	data := viewmodel.NewBacklinkList(viewmodel.NewBacklinkListInput{
		Pages:           pages,
		SpaceIdentifier: model.SpaceIdentifier("my-space"),
	})

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	var buf bytes.Buffer
	err := components.PageBacklinkList(data).Render(ctx, &buf)
	if err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}

	html := buf.String()

	// 「無題」が表示されること
	if !strings.Contains(html, "無題") {
		t.Error("タイトルが空のバックリンクに「無題」が表示されていない")
	}

	// リンクのhrefが正しいこと
	if !strings.Contains(html, "/s/my-space/pages/5") {
		t.Error("タイトルなしバックリンクのリンク先が正しくない")
	}
}

func TestPageBacklinkList_WithPagination(t *testing.T) {
	t.Parallel()

	title := "リンク元ページ"
	pages := []*model.Page{
		{
			Number: 10,
			Title:  &title,
		},
	}

	data := viewmodel.NewBacklinkList(viewmodel.NewBacklinkListInput{
		Pages:           pages,
		Pagination:      viewmodel.NewPagination(1, 20, 15),
		SpaceIdentifier: model.SpaceIdentifier("my-space"),
		PageNumber:      1,
		State: viewmodel.PageLinkState{
			Context:            viewmodel.PageLinkContextShow,
			LinkPage:           2,
			LinkedPageNumber:   9,
			LinkedBacklinkPage: 2,
		},
	})

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	var buf bytes.Buffer
	err := components.PageBacklinkList(data).Render(ctx, &buf)
	if err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}

	html := buf.String()

	if !strings.Contains(html, `id="page-backlink-list-content"`) {
		t.Error("ページのバックリンク一覧のページ全体のフォールバックのアンカーが無い")
	}

	// ページネーションコンテナが表示されること
	if !strings.Contains(html, "page-backlink-list-pagination") {
		t.Error("ページネーションコンテナが表示されていない")
	}

	// htmxはフラグメントURLを使い、通常の遷移はフルページURLを使う。
	if !strings.Contains(html, `hx-get="/s/my-space/pages/1/backlinks?context=show&amp;linked_backlinks_page=2&amp;linked_page_number=9&amp;linked_page_parent_page=2&amp;links_page=2&amp;page=2"`) {
		t.Error("htmxのさらに読み込むリンクのURLが正しくない")
	}
	// フォールバックはページ自身のバックリンクだけを進め、リンク一覧とネストした一覧は現在位置の
	// ままになる。
	if !strings.Contains(html, `href="/s/my-space/pages/1?backlinks_page=2&amp;linked_backlinks_page=2&amp;linked_page_number=9&amp;links_page=2#page-backlink-list-content"`) {
		t.Error("ページ全体のフォールバックURLが正しくない")
	}
	if !strings.Contains(html, `aria-label="バックリンクをもっと見る"`) {
		t.Error("さらに読み込むリンクに進める一覧の名前が無い")
	}
}

func TestPageBacklinkList_WithoutPagination(t *testing.T) {
	t.Parallel()

	title := "リンク元ページ"
	pages := []*model.Page{
		{
			Number: 10,
			Title:  &title,
		},
	}

	data := viewmodel.NewBacklinkList(viewmodel.NewBacklinkListInput{
		Pages:           pages,
		Pagination:      viewmodel.NewPagination(1, 1, 15),
		SpaceIdentifier: model.SpaceIdentifier("my-space"),
		PageNumber:      1,
	})

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	var buf bytes.Buffer
	err := components.PageBacklinkList(data).Render(ctx, &buf)
	if err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}

	html := buf.String()

	// 「もっと見る」ボタンが表示されないこと (HasNext=false)
	if strings.Contains(html, "/backlinks?page=") {
		t.Error("ページネーションが不要なとき「もっと見る」ボタンが表示されてはいけない")
	}
}

// TestPageBacklinkList_LoadMoreFallbackNamesTheSelectedCardsLinkPageは、ページ自身のバックリンク
// 一覧のフルページフォールバックで、ネスト状態とリンク一覧のページが整合することを固定する。本リンクは
// 画面が開いているカードのネスト状態を運ぶが、そのカードは描画元のリンク一覧ページにしか存在しないため、
// リンク一覧が現在到達しているページではなくそのページを指す。
func TestPageBacklinkList_LoadMoreFallbackNamesTheSelectedCardsLinkPage(t *testing.T) {
	t.Parallel()

	title := "リンク元ページ"
	pages := []*model.Page{
		{
			Number: 10,
			Title:  &title,
		},
	}

	data := viewmodel.NewBacklinkList(viewmodel.NewBacklinkListInput{
		Pages:           pages,
		Pagination:      viewmodel.NewPagination(1, 20, 15),
		SpaceIdentifier: model.SpaceIdentifier("my-space"),
		PageNumber:      1,
		State: viewmodel.PageLinkState{
			Context:              viewmodel.PageLinkContextEdit,
			LinkPage:             3,
			LinkedPageNumber:     9,
			LinkedBacklinkPage:   2,
			LinkedPageParentPage: 1,
		},
	})

	ctx := i18n.SetLocale(context.Background(), "ja")

	var buf bytes.Buffer
	if err := components.PageBacklinkList(data).Render(ctx, &buf); err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}

	html := buf.String()

	// 選択カードはリンク一覧の1ページ目にあるため、フォールバックはリンク一覧が到達している
	// 3ページ目ではなくlinks_pageを省く。
	wantHref := `href="/s/my-space/pages/1/edit?backlinks_page=2&amp;linked_backlinks_page=2&amp;linked_page_number=9#page-backlink-list-content"`
	if !strings.Contains(html, wantHref) {
		t.Errorf("ページ全体のフォールバックが選択中のカードのリンクのページを指していない: 期待値 = %s", wantHref)
	}

	// htmxフラグメントは両方の値を分けて運び、応答が同じフォールバックを組み立て直せるようにする。
	wantGet := `hx-get="/s/my-space/pages/1/backlinks?context=edit&amp;page=2"`
	if !strings.Contains(html, wantGet) {
		t.Errorf("エディタのフラグメントがURLではなく共有の状態を読んでいない: 期待値 = %s", wantGet)
	}
}
