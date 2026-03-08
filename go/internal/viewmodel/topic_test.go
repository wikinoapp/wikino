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
