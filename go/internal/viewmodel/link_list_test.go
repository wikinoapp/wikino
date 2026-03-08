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
		spaceIdentifier model.SpaceIdentifier
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

			got := viewmodel.NewLinkList(viewmodel.NewLinkListInput{
				Pages:           tt.pages,
				SpaceIdentifier: tt.spaceIdentifier,
			})

			if len(got.Items) != tt.wantItemCount {
				t.Errorf("len(Items) = %d, want %d", len(got.Items), tt.wantItemCount)
			}

			if got.SpaceIdentifier != tt.spaceIdentifier {
				t.Errorf("SpaceIdentifier = %q, want %q", got.SpaceIdentifier, tt.spaceIdentifier)
			}

			for i, item := range got.Items {
				if i < len(tt.wantTitles) && item.CardLinkPage.Title != tt.wantTitles[i] {
					t.Errorf("Items[%d].Page.Title = %q, want %q", i, item.CardLinkPage.Title, tt.wantTitles[i])
				}
				if i < len(tt.wantNumbers) && item.CardLinkPage.Number != tt.wantNumbers[i] {
					t.Errorf("Items[%d].Page.Number = %d, want %d", i, item.CardLinkPage.Number, tt.wantNumbers[i])
				}
			}
		})
	}
}

func TestNewLinkList_WithBacklinkMap(t *testing.T) {
	t.Parallel()

	strPtr := func(s string) *string { return &s }

	page1ID := model.PageID("page-1")
	page2ID := model.PageID("page-2")

	pages := []*model.Page{
		{ID: page1ID, Number: 1, Title: strPtr("ページ1")},
		{ID: page2ID, Number: 2, Title: strPtr("ページ2")},
	}

	backlinkMap := map[model.PageID]viewmodel.BacklinkList{
		page1ID: {
			Items: []viewmodel.BacklinkListItem{
				{CardLinkPage: viewmodel.CardLinkPage{Title: "バックリンク1", Number: 10}},
			},
		},
	}

	got := viewmodel.NewLinkList(viewmodel.NewLinkListInput{
		Pages:           pages,
		BacklinkMap:     backlinkMap,
		SpaceIdentifier: "test-space",
	})

	if len(got.Items) != 2 {
		t.Fatalf("len(Items) = %d, want 2", len(got.Items))
	}

	// page1にはバックリンクが設定される
	if len(got.Items[0].BacklinkList.Items) != 1 {
		t.Errorf("Items[0].BacklinkList.Items の件数 = %d, want 1", len(got.Items[0].BacklinkList.Items))
	}

	// page2にはバックリンクがない
	if len(got.Items[1].BacklinkList.Items) != 0 {
		t.Errorf("Items[1].BacklinkList.Items の件数 = %d, want 0", len(got.Items[1].BacklinkList.Items))
	}
}

func TestNewLinkList_WithPagination(t *testing.T) {
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

	got := viewmodel.NewLinkList(viewmodel.NewLinkListInput{
		Pages:           pages,
		Pagination:      pagination,
		SpaceIdentifier: "test-space",
		PageNumber:      42,
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
	if got.PageNumber != 42 {
		t.Errorf("PageNumber = %d, want 42", got.PageNumber)
	}
}
