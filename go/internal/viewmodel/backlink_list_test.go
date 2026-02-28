package viewmodel_test

import (
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

func TestNewBacklinkList(t *testing.T) {
	t.Parallel()

	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name          string
		pages         []*model.Page
		wantItemCount int
		wantTitles    []string
	}{
		{
			name: "バックリンクページからBacklinkListを生成できる",
			pages: []*model.Page{
				{Number: 10, Title: strPtr("バックリンク1")},
				{Number: 20, Title: strPtr("バックリンク2")},
			},
			wantItemCount: 2,
			wantTitles:    []string{"バックリンク1", "バックリンク2"},
		},
		{
			name:          "空のページリストの場合はアイテムが0件になる",
			pages:         []*model.Page{},
			wantItemCount: 0,
		},
		{
			name: "タイトルがnilの場合は空文字になる",
			pages: []*model.Page{
				{Number: 5, Title: nil},
			},
			wantItemCount: 1,
			wantTitles:    []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := viewmodel.NewBacklinkList(viewmodel.NewBacklinkListInput{
				Pages: tt.pages,
			})

			if len(got.Items) != tt.wantItemCount {
				t.Errorf("len(Items) = %d, want %d", len(got.Items), tt.wantItemCount)
			}

			for i, item := range got.Items {
				if i < len(tt.wantTitles) && item.Page.Title != tt.wantTitles[i] {
					t.Errorf("Items[%d].Page.Title = %q, want %q", i, item.Page.Title, tt.wantTitles[i])
				}
			}
		})
	}
}

func TestNewBacklinkList_WithPagination(t *testing.T) {
	t.Parallel()

	strPtr := func(s string) *string { return &s }

	pages := []*model.Page{
		{Number: 1, Title: strPtr("ページ1")},
	}

	pagination := viewmodel.Pagination{
		Current:     1,
		Total:       3,
		HasNext:     true,
		HasPrevious: false,
	}

	got := viewmodel.NewBacklinkList(viewmodel.NewBacklinkListInput{
		Pages:      pages,
		Pagination: pagination,
	})

	if got.Pagination.Current != 1 {
		t.Errorf("Pagination.Current = %d, want 1", got.Pagination.Current)
	}
	if got.Pagination.Total != 3 {
		t.Errorf("Pagination.Total = %d, want 3", got.Pagination.Total)
	}
	if !got.Pagination.HasNext {
		t.Error("Pagination.HasNext = false, want true")
	}
	if got.Pagination.HasPrevious {
		t.Error("Pagination.HasPrevious = true, want false")
	}
}
