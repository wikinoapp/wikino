package viewmodel_test

import (
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

func TestNewLinkList(t *testing.T) {
	t.Parallel()

	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name            string
		pages           []*model.Page
		spaceIdentifier string
		wantItemCount   int
		wantTitles      []string
		wantNumbers     []int32
	}{
		{
			name: "複数ページのリンク一覧を生成できる",
			pages: []*model.Page{
				{Number: 1, Title: strPtr("ページ1")},
				{Number: 2, Title: strPtr("ページ2")},
			},
			spaceIdentifier: "test-space",
			wantItemCount:   2,
			wantTitles:      []string{"ページ1", "ページ2"},
			wantNumbers:     []int32{1, 2},
		},
		{
			name: "タイトルがnilの場合は空文字になる",
			pages: []*model.Page{
				{Number: 3, Title: nil},
			},
			spaceIdentifier: "test-space",
			wantItemCount:   1,
			wantTitles:      []string{""},
			wantNumbers:     []int32{3},
		},
		{
			name: "タイトルありとnilが混在する場合",
			pages: []*model.Page{
				{Number: 1, Title: strPtr("タイトルあり")},
				{Number: 2, Title: nil},
				{Number: 3, Title: strPtr("別のタイトル")},
			},
			spaceIdentifier: "my-space",
			wantItemCount:   3,
			wantTitles:      []string{"タイトルあり", "", "別のタイトル"},
			wantNumbers:     []int32{1, 2, 3},
		},
		{
			name:            "空のページリストの場合はアイテムが0件になる",
			pages:           []*model.Page{},
			spaceIdentifier: "test-space",
			wantItemCount:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := viewmodel.NewLinkList(tt.pages, tt.spaceIdentifier)

			if len(got.Items) != tt.wantItemCount {
				t.Errorf("len(Items) = %d, want %d", len(got.Items), tt.wantItemCount)
			}

			if got.SpaceIdentifier != tt.spaceIdentifier {
				t.Errorf("SpaceIdentifier = %q, want %q", got.SpaceIdentifier, tt.spaceIdentifier)
			}

			for i, item := range got.Items {
				if i < len(tt.wantTitles) && item.Title != tt.wantTitles[i] {
					t.Errorf("Items[%d].Title = %q, want %q", i, item.Title, tt.wantTitles[i])
				}
				if i < len(tt.wantNumbers) && item.Number != tt.wantNumbers[i] {
					t.Errorf("Items[%d].Number = %d, want %d", i, item.Number, tt.wantNumbers[i])
				}
			}
		})
	}
}
