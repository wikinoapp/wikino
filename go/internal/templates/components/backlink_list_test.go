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

// TestBacklinkList_LoadMoreURLs verifies that advancing one listed page's nested backlinks keeps
// every other listing where it is, for both htmx and full-page navigation.
//
// [Ja] TestBacklinkList_LoadMoreURLs は、あるリンク先ページのネストしたバックリンクを進めても、
// htmx とフルページ遷移の双方で他の一覧が現在位置のまま保たれることを確認する。
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
		t.Fatalf("render BacklinkList: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, "hx-get=\"/s/my-space/pages/1/links/9/backlink_list?backlinks_page=3&amp;context=show&amp;links_page=2&amp;page=2&amp;parent_page=2\"") {
		t.Error("htmx nested-backlink URL is incorrect")
	}
	// The fallback advances only this card's nested list; links_page and backlinks_page stay put.
	//
	// [Ja] フォールバックはこのカードのネストした一覧だけを進め、links_page と backlinks_page は
	// 現在位置のままになる。
	if !strings.Contains(html, "href=\"/s/my-space/pages/1?backlinks_page=3&amp;linked_backlinks_page=2&amp;linked_page_number=9&amp;links_page=2#page-link-list-item-9\"") {
		t.Error("full-page nested-backlink fallback URL is incorrect")
	}
	if !strings.Contains(html, `aria-label="リンク先ページのバックリンクをもっと見る"`) {
		t.Error("the load-more link should name the listed page it belongs to")
	}
}

// TestBacklinkList_LoadMoreFallbackNamesTheCardsOwnLinkPage verifies that the full-page fallback
// renders the link-list page holding this card, even when the link list has since been advanced
// past it. In the editor the screen-wide state reaches the link a request at a time, so the two
// routinely differ; pairing this card with the later page would name a card the full page does not
// render, which the editor answers with a 404.
//
// [Ja] TestBacklinkList_LoadMoreFallbackNamesTheCardsOwnLinkPage は、リンク一覧がその先へ進んだ後でも
// フルページフォールバックがこのカードを含むリンク一覧ページを描画することを確認する。編集画面では
// 画面全体の状態がリクエストごとにリンクへ届くため両者は普通に食い違う。このカードを後続ページと
// 組み合わせると、フルページが描画しないカードを指すことになり、編集画面は 404 で答えてしまう。
func TestBacklinkList_LoadMoreFallbackNamesTheCardsOwnLinkPage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		parentLinkPage int32
		wantHref       string
	}{
		{
			name:           "card on the first link-list page",
			parentLinkPage: 1,
			wantHref:       `href="/s/my-space/pages/1/edit?linked_backlinks_page=2&amp;linked_page_number=9#page-link-list-item-9"`,
		},
		{
			name:           "card on an intermediate link-list page",
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
				t.Fatalf("render BacklinkList: %v", err)
			}

			if !strings.Contains(buf.String(), tt.wantHref) {
				t.Errorf("full-page fallback does not name the card's own link page: want %s", tt.wantHref)
			}
		})
	}
}

// TestBacklinkList_LoadMoreURLsWithoutTitle verifies that an untitled listed page still yields an
// accessible name, rather than one with a hole where the title would be.
//
// [Ja] TestBacklinkList_LoadMoreURLsWithoutTitle は、タイトルの無いリンク先ページでも、タイトルの
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
		t.Fatalf("render BacklinkList: %v", err)
	}

	if !strings.Contains(buf.String(), `aria-label="無題のバックリンクをもっと見る"`) {
		t.Error("an untitled listed page should fall back to the untitled label")
	}
}
