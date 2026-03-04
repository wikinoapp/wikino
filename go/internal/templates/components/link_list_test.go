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
		SpaceIdentifier: model.SpaceIdentifier("my-space"),
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

	// 「もっと見る」ボタンのURLが /link_list を使用していること
	if !strings.Contains(html, "/s/my-space/pages/1/link_list?page=2") {
		t.Error("「もっと見る」ボタンのURLに /link_list が含まれていない")
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
		SpaceIdentifier: model.SpaceIdentifier("my-space"),
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
		SpaceIdentifier: model.SpaceIdentifier("my-space"),
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
