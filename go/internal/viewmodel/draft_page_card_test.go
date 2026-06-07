package viewmodel_test

import (
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

func TestDraftPageCard_DisplayTitle(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(t.Context(), i18n.LangJa)

	t.Run("下書きタイトルが設定されているときはそれを返す", func(t *testing.T) {
		draft := &model.DraftPage{
			Title: strPtr("下書きタイトル"),
			Page:  &model.Page{Number: 1, Title: strPtr("公開ページタイトル")},
			Topic: &model.Topic{
				Name:  "トピック",
				Space: &model.Space{Identifier: "space-a", Name: "スペースA"},
			},
		}
		card := viewmodel.NewDraftPageCard(draft)

		if got := card.DisplayTitle(ctx); got != "下書きタイトル" {
			t.Errorf("DisplayTitle() = %q, want %q", got, "下書きタイトル")
		}
	})

	t.Run("下書きタイトルが空のときは公開ページのタイトルにフォールバックする", func(t *testing.T) {
		empty := ""
		draft := &model.DraftPage{
			Title: &empty,
			Page:  &model.Page{Number: 1, Title: strPtr("公開ページタイトル")},
			Topic: &model.Topic{
				Name:  "トピック",
				Space: &model.Space{Identifier: "space-a", Name: "スペースA"},
			},
		}
		card := viewmodel.NewDraftPageCard(draft)

		if got := card.DisplayTitle(ctx); got != "公開ページタイトル" {
			t.Errorf("DisplayTitle() = %q, want %q", got, "公開ページタイトル")
		}
	})

	t.Run("下書きタイトルが nil のときは公開ページのタイトルにフォールバックする", func(t *testing.T) {
		draft := &model.DraftPage{
			Title: nil,
			Page:  &model.Page{Number: 1, Title: strPtr("公開ページタイトル")},
			Topic: &model.Topic{
				Name:  "トピック",
				Space: &model.Space{Identifier: "space-a", Name: "スペースA"},
			},
		}
		card := viewmodel.NewDraftPageCard(draft)

		if got := card.DisplayTitle(ctx); got != "公開ページタイトル" {
			t.Errorf("DisplayTitle() = %q, want %q", got, "公開ページタイトル")
		}
	})

	t.Run("下書き・公開ページのタイトルがいずれも未設定のときは 無題 を返す", func(t *testing.T) {
		draft := &model.DraftPage{
			Title: nil,
			Page:  &model.Page{Number: 1, Title: nil},
			Topic: &model.Topic{
				Name:  "トピック",
				Space: &model.Space{Identifier: "space-a", Name: "スペースA"},
			},
		}
		card := viewmodel.NewDraftPageCard(draft)

		if got := card.DisplayTitle(ctx); got != "無題" {
			t.Errorf("DisplayTitle() = %q, want %q", got, "無題")
		}
	})
}

func TestNewDraftPageCard_FieldMapping(t *testing.T) {
	t.Parallel()

	t.Run("公開トピックは globe-regular アイコンを設定する", func(t *testing.T) {
		draft := &model.DraftPage{
			Title: strPtr("タイトル"),
			Page:  &model.Page{Number: 7, Title: strPtr("公開ページタイトル")},
			Topic: &model.Topic{
				Name:       "トピックA",
				Visibility: model.TopicVisibilityPublic,
				Space:      &model.Space{Identifier: "space-a", Name: "スペースA"},
			},
		}
		card := viewmodel.NewDraftPageCard(draft)

		if card.SpaceName != "スペースA" {
			t.Errorf("SpaceName = %q, want %q", card.SpaceName, "スペースA")
		}
		if card.TopicName != "トピックA" {
			t.Errorf("TopicName = %q, want %q", card.TopicName, "トピックA")
		}
		if string(card.SpaceIdentifier) != "space-a" {
			t.Errorf("SpaceIdentifier = %q, want %q", string(card.SpaceIdentifier), "space-a")
		}
		if card.PageNumber != 7 {
			t.Errorf("PageNumber = %d, want %d", card.PageNumber, 7)
		}
		if card.TopicIconName != "globe-regular" {
			t.Errorf("TopicIconName = %q, want %q", card.TopicIconName, "globe-regular")
		}
	})

	t.Run("非公開トピックは lock-regular アイコンを設定する", func(t *testing.T) {
		draft := &model.DraftPage{
			Title: strPtr("タイトル"),
			Page:  &model.Page{Number: 7},
			Topic: &model.Topic{
				Name:       "プライベートトピック",
				Visibility: model.TopicVisibilityPrivate,
				Space:      &model.Space{Identifier: "space-a", Name: "スペースA"},
			},
		}
		card := viewmodel.NewDraftPageCard(draft)

		if card.TopicIconName != "lock-regular" {
			t.Errorf("TopicIconName = %q, want %q", card.TopicIconName, "lock-regular")
		}
	})
}

func TestNewDraftPageCards(t *testing.T) {
	t.Parallel()

	drafts := []*model.DraftPage{
		{
			Title: strPtr("下書き1"),
			Page:  &model.Page{Number: 1},
			Topic: &model.Topic{Name: "T1", Space: &model.Space{Identifier: "s1", Name: "S1"}},
		},
		{
			Title: strPtr("下書き2"),
			Page:  &model.Page{Number: 2},
			Topic: &model.Topic{Name: "T2", Space: &model.Space{Identifier: "s2", Name: "S2"}},
		},
	}

	cards := viewmodel.NewDraftPageCards(drafts)
	if len(cards) != 2 {
		t.Fatalf("len(cards) = %d, want 2", len(cards))
	}
	if cards[0].TopicName != "T1" || cards[1].TopicName != "T2" {
		t.Errorf("topic names mismatch: got %q, %q", cards[0].TopicName, cards[1].TopicName)
	}
}
