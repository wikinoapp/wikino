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
				t.Errorf("Name = %q、期待値 = %q", got.Name, tt.wantName)
			}

			if got.Number != tt.wantNumber {
				t.Errorf("Number = %d、期待値 = %d", got.Number, tt.wantNumber)
			}

			if got.IconName != tt.wantIconName {
				t.Errorf("IconName = %q、期待値 = %q", got.IconName, tt.wantIconName)
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
				t.Errorf("Name = %q、期待値 = %q", got.Name, tt.wantName)
			}

			if got.Number != tt.wantNumber {
				t.Errorf("Number = %d、期待値 = %d", got.Number, tt.wantNumber)
			}

			if got.Description != tt.wantDescription {
				t.Errorf("Description = %q、期待値 = %q", got.Description, tt.wantDescription)
			}

			if got.IconName != tt.wantIconName {
				t.Errorf("IconName = %q、期待値 = %q", got.IconName, tt.wantIconName)
			}

			if got.CanUpdate != tt.wantCanUpdate {
				t.Errorf("CanUpdate = %v、期待値 = %v", got.CanUpdate, tt.wantCanUpdate)
			}

			if got.CanCreatePage != tt.wantCanCreatePage {
				t.Errorf("CanCreatePage = %v、期待値 = %v", got.CanCreatePage, tt.wantCanCreatePage)
			}
		})
	}
}

func TestNewCardLinkTopicsForSpace(t *testing.T) {
	t.Parallel()

	topic1 := &model.Topic{ID: "topic-1", Number: 1, Name: "T1", Visibility: model.TopicVisibilityPublic}
	topic2 := &model.Topic{ID: "topic-2", Number: 2, Name: "T2", Visibility: model.TopicVisibilityPrivate}
	topics := []*model.Topic{topic1, topic2}

	// topic1のみ書き込み可。topic2はマップに無くfalseにフォールバックする。
	canCreatePageByTopic := map[model.TopicID]bool{"topic-1": true}
	spaceIdentifier := viewmodel.NewSpaceIdentifier("my-space")

	got := viewmodel.NewCardLinkTopicsForSpace(topics, canCreatePageByTopic, spaceIdentifier)

	if len(got) != 2 {
		t.Fatalf("len(got) = %d、期待値 = 2", len(got))
	}

	// 各カードは渡されたスペース識別子を使い (トピックはSpaceを読み込んでいない)、スペース名は
	// 非表示にする。一方、公開範囲アイコンと作成権限はトピックごと / マップから決まる。
	if got[0].Name != "T1" || got[1].Name != "T2" {
		t.Errorf("トピック名が一致しない: %q、%q", got[0].Name, got[1].Name)
	}
	for i, card := range got {
		if card.SpaceIdentifier != spaceIdentifier {
			t.Errorf("got[%d].SpaceIdentifier = %q、期待値 = %q", i, card.SpaceIdentifier, spaceIdentifier)
		}
		if card.SpaceName != "" {
			t.Errorf("got[%d].SpaceName = %q、期待値 = 空", i, card.SpaceName)
		}
	}
	if got[0].TopicIconName != "globe-regular" {
		t.Errorf("got[0].TopicIconName = %q、期待値 = %q", got[0].TopicIconName, "globe-regular")
	}
	if got[1].TopicIconName != "lock-regular" {
		t.Errorf("got[1].TopicIconName = %q、期待値 = %q", got[1].TopicIconName, "lock-regular")
	}
	if !got[0].CanCreatePage {
		t.Error("got[0].CanCreatePage = false、期待値 = true")
	}
	if got[1].CanCreatePage {
		t.Error("got[1].CanCreatePage = true、期待値 = false (mapに無い)")
	}
}

func TestNewCardLinkTopic_FieldMapping(t *testing.T) {
	t.Parallel()

	t.Run("公開トピックはglobe-regularアイコンを設定し、作成権限を引き継ぐ", func(t *testing.T) {
		topic := &model.Topic{
			Number:     7,
			Name:       "トピックA",
			Visibility: model.TopicVisibilityPublic,
			Space: &model.Space{
				Identifier: "space-a",
				Name:       "スペースA",
			},
		}
		card := viewmodel.NewCardLinkTopic(topic, true)

		if card.Name != "トピックA" {
			t.Errorf("Name = %q、期待値 = %q", card.Name, "トピックA")
		}
		if card.Number != 7 {
			t.Errorf("Number = %d、期待値 = %d", card.Number, 7)
		}
		if string(card.SpaceIdentifier) != "space-a" {
			t.Errorf("SpaceIdentifier = %q、期待値 = %q", string(card.SpaceIdentifier), "space-a")
		}
		if card.SpaceName != "スペースA" {
			t.Errorf("SpaceName = %q、期待値 = %q", card.SpaceName, "スペースA")
		}
		if card.TopicIconName != "globe-regular" {
			t.Errorf("TopicIconName = %q、期待値 = %q", card.TopicIconName, "globe-regular")
		}
		if !card.CanCreatePage {
			t.Error("CanCreatePage = false、期待値 = true")
		}
	})

	t.Run("非公開トピックはlock-regularアイコンを設定し、作成権限なしを引き継ぐ", func(t *testing.T) {
		topic := &model.Topic{
			Number:     3,
			Name:       "プライベートトピック",
			Visibility: model.TopicVisibilityPrivate,
			Space: &model.Space{
				Identifier: "space-b",
				Name:       "スペースB",
			},
		}
		card := viewmodel.NewCardLinkTopic(topic, false)

		if card.TopicIconName != "lock-regular" {
			t.Errorf("TopicIconName = %q、期待値 = %q", card.TopicIconName, "lock-regular")
		}
		if card.CanCreatePage {
			t.Error("CanCreatePage = true、期待値 = false")
		}
	})
}

func TestNewCardLinkTopics(t *testing.T) {
	t.Parallel()

	topics := []*model.Topic{
		{
			ID:         "topic-1",
			Number:     1,
			Name:       "T1",
			Visibility: model.TopicVisibilityPublic,
			Space:      &model.Space{Identifier: "s1", Name: "S1"},
		},
		{
			ID:         "topic-2",
			Number:     2,
			Name:       "T2",
			Visibility: model.TopicVisibilityPrivate,
			Space:      &model.Space{Identifier: "s2", Name: "S2"},
		},
	}

	// topic-1のみ書き込み可。topic-2はマップに無くfalseにフォールバックする。
	canCreatePageByTopic := map[model.TopicID]bool{"topic-1": true}

	cards := viewmodel.NewCardLinkTopics(topics, canCreatePageByTopic)
	if len(cards) != 2 {
		t.Fatalf("len(cards) = %d、期待値 = 2", len(cards))
	}
	if cards[0].Name != "T1" || cards[1].Name != "T2" {
		t.Errorf("トピック名が一致しない: %q、%q", cards[0].Name, cards[1].Name)
	}
	if cards[0].TopicIconName != "globe-regular" {
		t.Errorf("cards[0].TopicIconName = %q、期待値 = %q", cards[0].TopicIconName, "globe-regular")
	}
	if cards[1].TopicIconName != "lock-regular" {
		t.Errorf("cards[1].TopicIconName = %q、期待値 = %q", cards[1].TopicIconName, "lock-regular")
	}
	if !cards[0].CanCreatePage {
		t.Error("cards[0].CanCreatePage = false、期待値 = true")
	}
	if cards[1].CanCreatePage {
		t.Error("cards[1].CanCreatePage = true、期待値 = false (mapに無い)")
	}
}
