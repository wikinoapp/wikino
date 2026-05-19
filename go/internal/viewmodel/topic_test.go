package viewmodel_test

import (
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

func TestNewTopic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		topic        *model.Topic
		wantName     string
		wantNumber   int32
		wantIconName viewmodel.IconName
	}{
		{
			name: "公開トピック",
			topic: &model.Topic{
				ID:         "topic-1",
				Space:      &model.Space{ID: "space-1"},
				Number:     1,
				Name:       "一般",
				Visibility: model.TopicVisibilityPublic,
			},
			wantName:     "一般",
			wantNumber:   1,
			wantIconName: "globe-regular",
		},
		{
			name: "非公開トピック",
			topic: &model.Topic{
				ID:         "topic-2",
				Space:      &model.Space{ID: "space-1"},
				Number:     2,
				Name:       "秘密",
				Visibility: model.TopicVisibilityPrivate,
			},
			wantName:     "秘密",
			wantNumber:   2,
			wantIconName: "lock-regular",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := viewmodel.NewTopic(tt.topic)

			if got.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", got.Name, tt.wantName)
			}

			if got.Number != tt.wantNumber {
				t.Errorf("Number = %d, want %d", got.Number, tt.wantNumber)
			}

			if got.IconName != tt.wantIconName {
				t.Errorf("IconName = %q, want %q", got.IconName, tt.wantIconName)
			}
		})
	}
}

func TestNewTopicForShow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		topic             *model.Topic
		canUpdate         bool
		canCreatePage     bool
		wantName          string
		wantNumber        int32
		wantDescription   string
		wantIconName      viewmodel.IconName
		wantCanUpdate     bool
		wantCanCreatePage bool
	}{
		{
			name: "公開トピック、管理者権限あり",
			topic: &model.Topic{
				ID:          "topic-1",
				Space:       &model.Space{ID: "space-1"},
				Number:      1,
				Name:        "一般",
				Description: "一般的な話題",
				Visibility:  model.TopicVisibilityPublic,
			},
			canUpdate:         true,
			canCreatePage:     true,
			wantName:          "一般",
			wantNumber:        1,
			wantDescription:   "一般的な話題",
			wantIconName:      "globe-regular",
			wantCanUpdate:     true,
			wantCanCreatePage: true,
		},
		{
			name: "非公開トピック、閲覧のみ",
			topic: &model.Topic{
				ID:         "topic-2",
				Space:      &model.Space{ID: "space-1"},
				Number:     2,
				Name:       "秘密",
				Visibility: model.TopicVisibilityPrivate,
			},
			canUpdate:         false,
			canCreatePage:     false,
			wantName:          "秘密",
			wantNumber:        2,
			wantDescription:   "",
			wantIconName:      "lock-regular",
			wantCanUpdate:     false,
			wantCanCreatePage: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := viewmodel.NewTopicForShow(tt.topic, tt.canUpdate, tt.canCreatePage)

			if got.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", got.Name, tt.wantName)
			}

			if got.Number != tt.wantNumber {
				t.Errorf("Number = %d, want %d", got.Number, tt.wantNumber)
			}

			if got.Description != tt.wantDescription {
				t.Errorf("Description = %q, want %q", got.Description, tt.wantDescription)
			}

			if got.IconName != tt.wantIconName {
				t.Errorf("IconName = %q, want %q", got.IconName, tt.wantIconName)
			}

			if got.CanUpdate != tt.wantCanUpdate {
				t.Errorf("CanUpdate = %v, want %v", got.CanUpdate, tt.wantCanUpdate)
			}

			if got.CanCreatePage != tt.wantCanCreatePage {
				t.Errorf("CanCreatePage = %v, want %v", got.CanCreatePage, tt.wantCanCreatePage)
			}
		})
	}
}

func TestNewJoinedTopicCard_FieldMapping(t *testing.T) {
	t.Parallel()

	t.Run("公開トピックは globe-regular アイコンを設定する", func(t *testing.T) {
		topic := &model.Topic{
			Number:     7,
			Name:       "トピックA",
			Visibility: model.TopicVisibilityPublic,
			Space: &model.Space{
				Identifier: "space-a",
				Name:       "スペースA",
			},
		}
		card := viewmodel.NewJoinedTopicCard(topic)

		if card.Name != "トピックA" {
			t.Errorf("Name = %q, want %q", card.Name, "トピックA")
		}
		if card.Number != 7 {
			t.Errorf("Number = %d, want %d", card.Number, 7)
		}
		if string(card.SpaceIdentifier) != "space-a" {
			t.Errorf("SpaceIdentifier = %q, want %q", string(card.SpaceIdentifier), "space-a")
		}
		if card.SpaceName != "スペースA" {
			t.Errorf("SpaceName = %q, want %q", card.SpaceName, "スペースA")
		}
		if card.TopicIconName != "globe-regular" {
			t.Errorf("TopicIconName = %q, want %q", card.TopicIconName, "globe-regular")
		}
	})

	t.Run("非公開トピックは lock-regular アイコンを設定する", func(t *testing.T) {
		topic := &model.Topic{
			Number:     3,
			Name:       "プライベートトピック",
			Visibility: model.TopicVisibilityPrivate,
			Space: &model.Space{
				Identifier: "space-b",
				Name:       "スペースB",
			},
		}
		card := viewmodel.NewJoinedTopicCard(topic)

		if card.TopicIconName != "lock-regular" {
			t.Errorf("TopicIconName = %q, want %q", card.TopicIconName, "lock-regular")
		}
	})
}

func TestNewJoinedTopicCards(t *testing.T) {
	t.Parallel()

	topics := []*model.Topic{
		{
			Number:     1,
			Name:       "T1",
			Visibility: model.TopicVisibilityPublic,
			Space:      &model.Space{Identifier: "s1", Name: "S1"},
		},
		{
			Number:     2,
			Name:       "T2",
			Visibility: model.TopicVisibilityPrivate,
			Space:      &model.Space{Identifier: "s2", Name: "S2"},
		},
	}

	cards := viewmodel.NewJoinedTopicCards(topics)
	if len(cards) != 2 {
		t.Fatalf("len(cards) = %d, want 2", len(cards))
	}
	if cards[0].Name != "T1" || cards[1].Name != "T2" {
		t.Errorf("topic names mismatch: got %q, %q", cards[0].Name, cards[1].Name)
	}
	if cards[0].TopicIconName != "globe-regular" {
		t.Errorf("cards[0].TopicIconName = %q, want %q", cards[0].TopicIconName, "globe-regular")
	}
	if cards[1].TopicIconName != "lock-regular" {
		t.Errorf("cards[1].TopicIconName = %q, want %q", cards[1].TopicIconName, "lock-regular")
	}
}
