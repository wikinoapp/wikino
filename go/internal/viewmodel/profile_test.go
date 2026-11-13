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

func TestNewProfile_空白だけの項目を未設定にする(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		value string
	}{
		{name: "空文字列", value: ""},
		{name: "半角スペース", value: " "},
		{name: "タブ", value: "\t"},
		{name: "改行", value: "\n"},
		{name: "全角スペース", value: "　"},
		{name: "空白の混在", value: " \t\n　"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			profile := viewmodel.NewProfile(&model.User{
				Atname:      "alice",
				Name:        tc.value,
				Description: tc.value,
			})
			if profile.Name != "" || profile.Description != "" {
				t.Errorf("名前と自己紹介 = %q / %q、どちらも空文字列を期待", profile.Name, profile.Description)
			}
			if got := profile.Title(); got != "@alice" {
				t.Errorf("Title() = %q、期待値 = %q", got, "@alice")
			}
			if got := profile.Subtitle(); got != "" {
				t.Errorf("Subtitle() = %q、空文字列を期待", got)
			}
		})
	}
}

func TestNewProfile_文字を含む項目の前後の空白を保つ(t *testing.T) {
	t.Parallel()

	user := &model.User{
		Atname:      "alice",
		Name:        " Alice\t",
		Description: "\n自己紹介　",
	}
	profile := viewmodel.NewProfile(user)
	if profile.Name != user.Name {
		t.Errorf("Name = %q、期待値 = %q", profile.Name, user.Name)
	}
	if profile.Description != user.Description {
		t.Errorf("Description = %q、期待値 = %q", profile.Description, user.Description)
	}
}
