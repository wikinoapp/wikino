package viewmodel

import (
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestNewSearchResultPages(t *testing.T) {
	t.Parallel()

	title := "検索のページ"
	topic := &model.Topic{
		ID:     "topic-1",
		Name:   "トピック",
		Number: 1,
		Space:  &model.Space{ID: "space-1", Identifier: "my-space", Name: "スペース"},
	}
	pages := []*model.Page{
		{ID: "page-1", SpaceID: "space-1", TopicID: topic.ID, Number: 3, Title: &title},
		// トピックが解決できないページは、リンク先を組み立てられないため結果にしない
		{ID: "page-2", SpaceID: "space-2", TopicID: "unknown", Number: 4, Title: &title},
	}

	got := NewSearchResultPages(pages, map[model.TopicID]*model.Topic{topic.ID: topic})

	if len(got) != 1 {
		t.Fatalf("件数 = %d、期待値 = 1", len(got))
	}
	if got[0].SpaceIdentifier != "my-space" {
		t.Errorf("SpaceIdentifier = %q、期待値 = %q", got[0].SpaceIdentifier, "my-space")
	}
	card := got[0].Card
	if card.Title != title || card.Number != 3 || card.SpaceName != "スペース" || card.Topic == nil || card.Topic.Name != "トピック" {
		t.Errorf("Card = %+v", card)
	}
}
