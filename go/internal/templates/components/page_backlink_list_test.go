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

	// The heading (h2) was moved to the caller, so the component alone never renders it.
	//
	// [Ja] 見出し (h2) は呼び出し側に移したため、コンポーネント単体では描画されない。
	if strings.TrimSpace(html) != "" {
		t.Errorf("バックリンクが空のとき、HTMLは空であるべき: got %q", html)
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
	})

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	var buf bytes.Buffer
	err := components.PageBacklinkList(data).Render(ctx, &buf)
	if err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}

	html := buf.String()

	// ページネーションコンテナが表示されること
	if !strings.Contains(html, "page-backlink-list-pagination") {
		t.Error("ページネーションコンテナが表示されていない")
	}

	// 「もっと見る」ボタンが表示されること（HasNext=true）
	if !strings.Contains(html, "/s/my-space/pages/1/backlinks?page=2") {
		t.Error("「もっと見る」ボタンのURLが正しくない")
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

	// 「もっと見る」ボタンが表示されないこと（HasNext=false）
	if strings.Contains(html, "/backlinks?page=") {
		t.Error("ページネーションが不要なとき「もっと見る」ボタンが表示されてはいけない")
	}
}
