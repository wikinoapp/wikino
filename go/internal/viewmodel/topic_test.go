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
				SpaceID:    "space-1",
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
				SpaceID:    "space-1",
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
