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

func TestLinkList_LoadMoreURL(t *testing.T) {
	t.Parallel()

	data := viewmodel.LinkList{
		Items: []viewmodel.LinkListItem{
			{
				CardLinkPage: viewmodel.CardLinkPage{
					Title:  "リンク先ページ",
					Number: 2,
				},
			},
		},
		Pagination:      viewmodel.NewPagination(1, 20, 5),
		SpaceIdentifier: viewmodel.SpaceIdentifier("my-space"),
		PageNumber:      1,
		State: viewmodel.PageLinkState{
			Context:            viewmodel.PageLinkContextShow,
			LinkedPageNumber:   9,
			LinkedBacklinkPage: 2,
			PageBacklinkPage:   3,
		},
	}

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	var buf bytes.Buffer
	err := components.LinkList(data).Render(ctx, &buf)
	if err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}

	html := buf.String()

	if !strings.Contains(html, `id="page-link-list-content"`) {
		t.Error("full-page fallback anchor for the link list is missing")
	}
	if !strings.Contains(html, `id="page-link-list-item-2"`) {
		t.Error("full-page fallback anchor for the nested backlink list is missing")
	}

	// htmx uses the fragment URL, while normal navigation uses the full-page URL.
	//
	// [Ja] htmx はフラグメント URL を使い、通常の遷移はフルページ URL を使う。
	if !strings.Contains(html, `hx-get="/s/my-space/pages/1/link_list?backlinks_page=3&amp;context=show&amp;linked_backlinks_page=2&amp;linked_page_number=9&amp;linked_page_parent_page=1&amp;page=2"`) {
		t.Error("htmx load-more link does not contain /link_list")
	}
	// Advancing the parent link list resets its nested child, while the independent page-level
	// backlink list stays where it is.
	//
	// [Ja] 親のリンク一覧を進めると従属するネスト一覧はリセットし、独立したページ自身の
	// バックリンク一覧は現在位置のままにする。
	if !strings.Contains(html, `href="/s/my-space/pages/1?backlinks_page=3&amp;links_page=2#page-link-list-content"`) {
		t.Error("full-page fallback URL is incorrect")
	}
	if !strings.Contains(html, `aria-label="リンクをもっと見る"`) {
		t.Error("the load-more link should name the listing it advances")
	}

	// /draft_page が使われていないこと
	if strings.Contains(html, "/draft_page") {
		t.Error("「もっと見る」ボタンのURLに /draft_page が含まれてはいけない")
	}
}

func TestLinkList_Empty(t *testing.T) {
	t.Parallel()

	data := viewmodel.LinkList{
		Items:           nil,
		SpaceIdentifier: viewmodel.SpaceIdentifier("my-space"),
		PageNumber:      1,
	}

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	var buf bytes.Buffer
	err := components.LinkList(data).Render(ctx, &buf)
	if err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}

	html := buf.String()

	if strings.TrimSpace(html) != "" {
		t.Errorf("リンクが空のとき、HTMLは空であるべき: got %q", html)
	}
}

func TestLinkList_NoPagination(t *testing.T) {
	t.Parallel()

	data := viewmodel.LinkList{
		Items: []viewmodel.LinkListItem{
			{
				CardLinkPage: viewmodel.CardLinkPage{
					Title:  "リンク先ページ",
					Number: 2,
				},
			},
		},
		Pagination:      viewmodel.NewPagination(1, 1, 5),
		SpaceIdentifier: viewmodel.SpaceIdentifier("my-space"),
		PageNumber:      1,
	}

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	var buf bytes.Buffer
	err := components.LinkList(data).Render(ctx, &buf)
	if err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}

	html := buf.String()

	// 「もっと見る」ボタンが表示されないこと（HasNext=false）
	if strings.Contains(html, "/link_list?page=") {
		t.Error("ページネーションが不要なとき「もっと見る」ボタンが表示されてはいけない")
	}
}
