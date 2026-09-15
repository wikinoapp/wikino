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

// TestBacklinkList_LoadMoreURLsは、あるリンク先ページのネストしたバックリンクを進めても、
// htmxとフルページ遷移の双方で他の一覧が現在位置のまま保たれることを確認する。
func TestBacklinkList_LoadMoreURLs(t *testing.T) {
	t.Parallel()

	data := viewmodel.BacklinkList{
		Pagination:       viewmodel.NewPagination(1, 20, 13),
		SpaceIdentifier:  viewmodel.SpaceIdentifier("my-space"),
		PageNumber:       1,
		ParentLinkPage:   2,
		LinkedPageNumber: 9,
		LinkedPageTitle:  "リンク先ページ",
		State: viewmodel.PageLinkState{
			Context:          viewmodel.PageLinkContextShow,
			LinkPage:         2,
			PageBacklinkPage: 3,
		},
	}

	ctx := i18n.SetLocale(context.Background(), "ja")

	var buf bytes.Buffer
	if err := components.BacklinkList(data).Render(ctx, &buf); err != nil {
		t.Fatalf("BacklinkListの描画に失敗: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, "hx-get=\"/s/my-space/pages/1/links/9/backlink_list?backlinks_page=3&amp;context=show&amp;links_page=2&amp;page=2&amp;parent_page=2\"") {
		t.Error("htmxの入れ子のバックリンクのURLが正しくない")
	}
	// フォールバックはこのカードのネストした一覧だけを進め、links_pageとbacklinks_pageは
	// 現在位置のままになる。
	if !strings.Contains(html, "href=\"/s/my-space/pages/1?backlinks_page=3&amp;linked_backlinks_page=2&amp;linked_page_number=9&amp;links_page=2#page-link-list-item-9\"") {
		t.Error("入れ子のバックリンクのページ全体のフォールバックURLが正しくない")
	}
	if !strings.Contains(html, `aria-label="リンク先ページのバックリンクをもっと見る"`) {
		t.Error("さらに読み込むリンクに所属する一覧のページの名前が無い")
	}
}

// TestBacklinkList_LoadMoreFallbackNamesTheCardsOwnLinkPageは、リンク一覧がその先へ進んだ後でも
// フルページフォールバックがこのカードを含むリンク一覧ページを描画することを確認する。編集画面では
// 画面全体の状態がリクエストごとにリンクへ届くため両者は普通に食い違う。このカードを後続ページと
// 組み合わせると、フルページが描画しないカードを指すことになり、編集画面は404で答えてしまう。
func TestBacklinkList_LoadMoreFallbackNamesTheCardsOwnLinkPage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		parentLinkPage int32
		wantHref       string
	}{
		{
			name:           "リンク一覧の最初のページにあるカード",
			parentLinkPage: 1,
			wantHref:       `href="/s/my-space/pages/1/edit?linked_backlinks_page=2&amp;linked_page_number=9#page-link-list-item-9"`,
		},
		{
			name:           "リンク一覧の途中のページにあるカード",
			parentLinkPage: 2,
			wantHref:       `href="/s/my-space/pages/1/edit?linked_backlinks_page=2&amp;linked_page_number=9&amp;links_page=2#page-link-list-item-9"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data := viewmodel.BacklinkList{
				Pagination:       viewmodel.NewPagination(1, 20, 13),
				SpaceIdentifier:  viewmodel.SpaceIdentifier("my-space"),
				PageNumber:       1,
				ParentLinkPage:   tt.parentLinkPage,
				LinkedPageNumber: 9,
				LinkedPageTitle:  "リンク先ページ",
				State: viewmodel.PageLinkState{
					Context:              viewmodel.PageLinkContextEdit,
					LinkPage:             3,
					LinkedPageNumber:     9,
					LinkedBacklinkPage:   1,
					LinkedPageParentPage: tt.parentLinkPage,
				},
			}

			ctx := i18n.SetLocale(context.Background(), "ja")

			var buf bytes.Buffer
			if err := components.BacklinkList(data).Render(ctx, &buf); err != nil {
				t.Fatalf("BacklinkListの描画に失敗: %v", err)
			}

			if !strings.Contains(buf.String(), tt.wantHref) {
				t.Errorf("ページ全体のフォールバックがカード自身のリンクのページを指していない: 期待値 = %s", tt.wantHref)
			}
		})
	}
}

// TestBacklinkList_LoadMoreURLsWithoutTitleは、タイトルの無いリンク先ページでも、タイトルの
// 位置が空いたままにならずアクセシブルネームが成立することを確認する。
func TestBacklinkList_LoadMoreURLsWithoutTitle(t *testing.T) {
	t.Parallel()

	data := viewmodel.BacklinkList{
		Pagination:       viewmodel.NewPagination(1, 20, 13),
		SpaceIdentifier:  viewmodel.SpaceIdentifier("my-space"),
		PageNumber:       1,
		LinkedPageNumber: 9,
		State:            viewmodel.PageLinkState{Context: viewmodel.PageLinkContextShow},
	}

	ctx := i18n.SetLocale(context.Background(), "ja")

	var buf bytes.Buffer
	if err := components.BacklinkList(data).Render(ctx, &buf); err != nil {
		t.Fatalf("BacklinkListの描画に失敗: %v", err)
	}

	if !strings.Contains(buf.String(), `aria-label="無題のバックリンクをもっと見る"`) {
		t.Error("無題の一覧のページが無題のラベルになっていない")
	}
}
