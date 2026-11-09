package viewmodel_test

import (
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

func TestNewSpaceAtomFeed(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Domain: "wikino.example.com"}
	jst := time.FixedZone("JST", 9*60*60)
	space := &model.Space{
		ID:         "space-id",
		Identifier: "my-space",
		Name:       "マイスペース",
		JoinedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	title1 := "古いページ"
	title2 := "新しいページ"
	published1 := time.Date(2025, 1, 1, 9, 0, 0, 0, jst)
	published2 := time.Date(2025, 1, 2, 9, 0, 0, 0, jst)
	pages := []*model.Page{
		{
			ID:          "page-2",
			Number:      2,
			Title:       &title2,
			PublishedAt: &published2,
			ModifiedAt:  time.Date(2025, 1, 2, 10, 0, 0, 0, jst),
		},
		{
			ID:          "page-1",
			Number:      1,
			Title:       &title1,
			PublishedAt: &published1,
			// 公開の古いページのほうが最近更新されている
			ModifiedAt: time.Date(2025, 1, 3, 10, 0, 0, 0, jst),
		},
	}

	feed := viewmodel.NewSpaceAtomFeed(cfg, viewmodel.AtomFeedSource{
		Space:               space,
		Pages:               pages,
		ContentHTMLByPageID: map[model.PageID]string{"page-2": "<p>本文2</p>", "page-1": "<p>本文1</p>"},
		AlternatePath:       "/s/my-space",
		SelfPath:            "/s/my-space/atom",
	})

	t.Run("フィードの要素", func(t *testing.T) {
		t.Parallel()

		if want := "tag:wikino.example.com,2025:/s/my-space/atom"; feed.ID != want {
			t.Errorf("ID = %q、期待値 = %q", feed.ID, want)
		}
		if want := "マイスペース | Wikino"; feed.Title != want {
			t.Errorf("Title = %q、期待値 = %q", feed.Title, want)
		}
		wantLinks := []viewmodel.AtomLink{
			{Rel: "alternate", Type: "text/html", Href: "https://wikino.example.com/s/my-space"},
			{Rel: "self", Type: "application/atom+xml", Href: "https://wikino.example.com/s/my-space/atom"},
		}
		if len(feed.Links) != len(wantLinks) {
			t.Fatalf("len(Links) = %d、期待値 = %d", len(feed.Links), len(wantLinks))
		}
		for i, want := range wantLinks {
			if feed.Links[i] != want {
				t.Errorf("Links[%d] = %+v、期待値 = %+v", i, feed.Links[i], want)
			}
		}
		// エントリーの更新日時の最大値 (並び順の先頭ではない)
		if want := "2025-01-03T01:00:00Z"; feed.Updated != want {
			t.Errorf("Updated = %q、期待値 = %q", feed.Updated, want)
		}
		if feed.Author.Name != "マイスペース" {
			t.Errorf("Author.Name = %q、期待値 = %q", feed.Author.Name, "マイスペース")
		}
	})

	t.Run("エントリーの要素", func(t *testing.T) {
		t.Parallel()

		if len(feed.Entries) != 2 {
			t.Fatalf("len(Entries) = %d、期待値 = 2", len(feed.Entries))
		}
		entry := feed.Entries[0]
		if want := "tag:Wikino,2025:Page/page-2"; entry.ID != want {
			t.Errorf("ID = %q、期待値 = %q", entry.ID, want)
		}
		if entry.Title != "新しいページ" {
			t.Errorf("Title = %q、期待値 = %q", entry.Title, "新しいページ")
		}
		wantLink := viewmodel.AtomLink{Rel: "alternate", Type: "text/html", Href: "https://wikino.example.com/s/my-space/pages/2"}
		if entry.Link != wantLink {
			t.Errorf("Link = %+v、期待値 = %+v", entry.Link, wantLink)
		}
		if want := "2025-01-02T00:00:00Z"; entry.Published != want {
			t.Errorf("Published = %q、期待値 = %q", entry.Published, want)
		}
		if want := "2025-01-02T01:00:00Z"; entry.Updated != want {
			t.Errorf("Updated = %q、期待値 = %q", entry.Updated, want)
		}
		wantContent := viewmodel.AtomText{Type: "html", Body: "<p>本文2</p>"}
		if entry.Content != wantContent {
			t.Errorf("Content = %+v、期待値 = %+v", entry.Content, wantContent)
		}
	})

	t.Run("XMLに書き出すと本文のHTMLをエスケープする", func(t *testing.T) {
		t.Parallel()

		out, err := xml.Marshal(feed)
		if err != nil {
			t.Fatalf("xml.Marshal()のエラー = %v", err)
		}
		got := string(out)
		wantContains := []string{
			`<feed xmlns="http://www.w3.org/2005/Atom">`,
			`<content type="html">&lt;p&gt;本文2&lt;/p&gt;</content>`,
			`<link rel="self" type="application/atom+xml" href="https://wikino.example.com/s/my-space/atom"></link>`,
		}
		for _, want := range wantContains {
			if !strings.Contains(got, want) {
				t.Errorf("XMLに %q が含まれていない: %s", want, got)
			}
		}
	})
}

func TestNewSpaceAtomFeed_NoEntries(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Domain: "wikino.example.com"}
	space := &model.Space{
		ID:         "space-id",
		Identifier: "my-space",
		Name:       "マイスペース",
		JoinedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	feed := viewmodel.NewSpaceAtomFeed(cfg, viewmodel.AtomFeedSource{
		Space:         space,
		AlternatePath: "/s/my-space",
		SelfPath:      "/s/my-space/atom",
	})

	if len(feed.Entries) != 0 {
		t.Errorf("len(Entries) = %d、期待値 = 0", len(feed.Entries))
	}
	// エントリーが無くてもupdatedは必ず出す。値はスペースの作成日時で、取得のたびに変わらない
	if want := "2024-01-01T00:00:00Z"; feed.Updated != want {
		t.Errorf("Updated = %q、期待値 = %q", feed.Updated, want)
	}
}

func TestNewTopicAtomFeed(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Domain: "wikino.example.com"}
	space := &model.Space{
		ID:         "space-id",
		Identifier: "my-space",
		Name:       "マイスペース",
		JoinedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	topic := &model.Topic{ID: "topic-id", Number: 3, Name: "設計メモ"}
	title := "ページ"
	published := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	pages := []*model.Page{
		{ID: "page-1", Number: 1, Title: &title, PublishedAt: &published, ModifiedAt: published},
	}

	feed := viewmodel.NewTopicAtomFeed(cfg, topic, viewmodel.AtomFeedSource{
		Space:               space,
		Pages:               pages,
		ContentHTMLByPageID: map[model.PageID]string{"page-1": "<p>本文</p>"},
		AlternatePath:       "/s/my-space/topics/3",
		SelfPath:            "/s/my-space/topics/3/atom",
	})

	if want := "tag:wikino.example.com,2025:/s/my-space/topics/3/atom"; feed.ID != want {
		t.Errorf("ID = %q、期待値 = %q", feed.ID, want)
	}
	if want := "設計メモ | マイスペース | Wikino"; feed.Title != want {
		t.Errorf("Title = %q、期待値 = %q", feed.Title, want)
	}
	if want := "https://wikino.example.com/s/my-space/topics/3"; len(feed.Links) == 0 || feed.Links[0].Href != want {
		t.Errorf("Links = %+v、alternateの先頭が %q を期待", feed.Links, want)
	}
	if feed.Author.Name != "マイスペース" {
		t.Errorf("Author.Name = %q、期待値 = %q", feed.Author.Name, "マイスペース")
	}
	// 同じページなら、スペースのフィードと同じエントリーのidになる
	if len(feed.Entries) != 1 || feed.Entries[0].ID != "tag:Wikino,2025:Page/page-1" {
		t.Errorf("Entries = %+v、idが tag:Wikino,2025:Page/page-1 の1件を期待", feed.Entries)
	}
}
