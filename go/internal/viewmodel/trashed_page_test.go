package viewmodel

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestNewTrashedPages(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	trashedAt := time.Date(2025, 1, 2, 3, 4, 0, 0, time.UTC)
	title := "ゴミ箱のページ"
	topic := &model.Topic{ID: "topic-1", Name: "トピック", Number: 1}

	pages := []*model.Page{
		{ID: "page-1", TopicID: topic.ID, Number: 3, Title: &title, TrashedAt: &trashedAt},
		{ID: "page-2", TopicID: topic.ID, Number: 4, TrashedAt: &trashedAt},
		// トピックが渡されていないページは、閲覧者がゴミ箱を見られないページのため行にしない
		{ID: "page-3", TopicID: "other-topic", Number: 5, Title: &title, TrashedAt: &trashedAt},
	}

	got := NewTrashedPages(pages, map[model.TopicID]*model.Topic{topic.ID: topic})

	if len(got) != 2 {
		t.Fatalf("行数 = %d、期待値 = 2", len(got))
	}
	if got[0].ID != "page-1" || got[0].Number != 3 || got[0].Topic.Name != "トピック" || !got[0].TrashedAt.Equal(trashedAt) {
		t.Errorf("1行目 = %+v", got[0])
	}
	if got[0].DisplayTitle(ctx) != title {
		t.Errorf("1行目のタイトル = %q、期待値 = %q", got[0].DisplayTitle(ctx), title)
	}
	if got[1].DisplayTitle(ctx) != "無題" {
		t.Errorf("タイトルの無いページの表示 = %q、期待値 = %q", got[1].DisplayTitle(ctx), "無題")
	}
}
