package viewmodel_test

import (
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

func TestProfile_TitleとSubtitle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		user         *model.User
		wantTitle    string
		wantSubtitle string
	}{
		{
			name:         "名前があれば名前を見出しにし、アットネームを添える",
			user:         &model.User{Atname: "alice", Name: "Alice"},
			wantTitle:    "Alice",
			wantSubtitle: "@alice",
		},
		{
			name:         "名前が空ならアットネームを見出しにし、添えない",
			user:         &model.User{Atname: "alice"},
			wantTitle:    "@alice",
			wantSubtitle: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			profile := viewmodel.NewProfile(tt.user)
			if got := profile.Title(); got != tt.wantTitle {
				t.Errorf("Title() = %q、期待値 = %q", got, tt.wantTitle)
			}
			if got := profile.Subtitle(); got != tt.wantSubtitle {
				t.Errorf("Subtitle() = %q、期待値 = %q", got, tt.wantSubtitle)
			}
		})
	}
}
