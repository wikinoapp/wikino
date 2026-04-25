package viewmodel_test

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

func TestNewSuggestionsForList(t *testing.T) {
	t.Parallel()

	now := time.Now()
	memberID := model.SpaceMemberID("member-1")

	tests := []struct {
		name            string
		suggestions     []*model.Suggestion
		userMap         map[model.SpaceMemberID]*model.User
		wantCount       int
		wantTitles      []string
		wantCreatorName []string
	}{
		{
			name:        "空の編集提案リスト",
			suggestions: []*model.Suggestion{},
			userMap:     map[model.SpaceMemberID]*model.User{},
			wantCount:   0,
		},
		{
			name: "編集提案から一覧ViewModelを生成できる",
			suggestions: []*model.Suggestion{
				{
					Number:               1,
					Title:                "提案1",
					Status:               model.SuggestionStatusOpen,
					CreatedSpaceMemberID: memberID,
					CreatedAt:            now,
				},
				{
					Number:               2,
					Title:                "提案2",
					Status:               model.SuggestionStatusDraft,
					CreatedSpaceMemberID: memberID,
					CreatedAt:            now,
				},
			},
			userMap: map[model.SpaceMemberID]*model.User{
				memberID: {Name: "テストユーザー", Atname: "testuser"},
			},
			wantCount:       2,
			wantTitles:      []string{"提案1", "提案2"},
			wantCreatorName: []string{"テストユーザー", "テストユーザー"},
		},
		{
			name: "名前が空の場合はアットネームが表示名になる",
			suggestions: []*model.Suggestion{
				{
					Number:               1,
					Title:                "提案",
					Status:               model.SuggestionStatusOpen,
					CreatedSpaceMemberID: memberID,
					CreatedAt:            now,
				},
			},
			userMap: map[model.SpaceMemberID]*model.User{
				memberID: {Name: "", Atname: "testuser"},
			},
			wantCount:       1,
			wantCreatorName: []string{"testuser"},
		},
		{
			name: "UserMapに作成者がいない場合は空文字になる",
			suggestions: []*model.Suggestion{
				{
					Number:               1,
					Title:                "提案",
					Status:               model.SuggestionStatusOpen,
					CreatedSpaceMemberID: model.SpaceMemberID("unknown"),
					CreatedAt:            now,
				},
			},
			userMap:         map[model.SpaceMemberID]*model.User{},
			wantCount:       1,
			wantCreatorName: []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := viewmodel.NewSuggestionsForList(viewmodel.NewSuggestionForListInput{
				Suggestions: tt.suggestions,
				UserMap:     tt.userMap,
			})

			if len(got) != tt.wantCount {
				t.Errorf("len(result) = %d, want %d", len(got), tt.wantCount)
			}

			for i, item := range got {
				if i < len(tt.wantTitles) && item.Title != tt.wantTitles[i] {
					t.Errorf("result[%d].Title = %q, want %q", i, item.Title, tt.wantTitles[i])
				}
				if i < len(tt.wantCreatorName) && item.CreatorName != tt.wantCreatorName[i] {
					t.Errorf("result[%d].CreatorName = %q, want %q", i, item.CreatorName, tt.wantCreatorName[i])
				}
			}
		})
	}
}

func TestNewDraftPagesForSuggestionNew(t *testing.T) {
	t.Parallel()

	t.Run("空のスライスから空のViewModelスライスが生成される", func(t *testing.T) {
		t.Parallel()

		got := viewmodel.NewDraftPagesForSuggestionNew([]*model.DraftPage{})
		if len(got) != 0 {
			t.Errorf("len(result) = %d, want 0", len(got))
		}
	})

	t.Run("下書きタイトルが優先される", func(t *testing.T) {
		t.Parallel()

		draftTitle := "下書きタイトル"
		pageTitle := "ページタイトル"
		drafts := []*model.DraftPage{
			{
				ID:    model.DraftPageID("draft-1"),
				Title: &draftTitle,
				Page: &model.Page{
					ID:     model.PageID("page-1"),
					Title:  &pageTitle,
					Number: 1,
				},
			},
		}

		got := viewmodel.NewDraftPagesForSuggestionNew(drafts)
		if len(got) != 1 {
			t.Fatalf("len(result) = %d, want 1", len(got))
		}
		if got[0].Title != "下書きタイトル" {
			t.Errorf("Title = %q, want %q", got[0].Title, "下書きタイトル")
		}
		if got[0].PageNumber != 1 {
			t.Errorf("PageNumber = %d, want 1", got[0].PageNumber)
		}
	})

	t.Run("下書きタイトルがnilの場合はページタイトルが使われる", func(t *testing.T) {
		t.Parallel()

		pageTitle := "ページタイトル"
		drafts := []*model.DraftPage{
			{
				ID: model.DraftPageID("draft-2"),
				Page: &model.Page{
					ID:     model.PageID("page-2"),
					Title:  &pageTitle,
					Number: 5,
				},
			},
		}

		got := viewmodel.NewDraftPagesForSuggestionNew(drafts)
		if len(got) != 1 {
			t.Fatalf("len(result) = %d, want 1", len(got))
		}
		if got[0].Title != "ページタイトル" {
			t.Errorf("Title = %q, want %q", got[0].Title, "ページタイトル")
		}
	})

	t.Run("両方のタイトルがnilの場合は空文字になる", func(t *testing.T) {
		t.Parallel()

		drafts := []*model.DraftPage{
			{
				ID: model.DraftPageID("draft-3"),
				Page: &model.Page{
					ID:     model.PageID("page-3"),
					Number: 3,
				},
			},
		}

		got := viewmodel.NewDraftPagesForSuggestionNew(drafts)
		if len(got) != 1 {
			t.Fatalf("len(result) = %d, want 1", len(got))
		}
		if got[0].Title != "" {
			t.Errorf("Title = %q, want empty string", got[0].Title)
		}
	})
}

func TestDraftPageForSuggestionNew_DisplayTitle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	t.Run("タイトルがある場合はタイトルを返す", func(t *testing.T) {
		t.Parallel()

		dp := viewmodel.DraftPageForSuggestionNew{
			ID:    model.DraftPageID("draft-1"),
			Title: "テストタイトル",
		}

		got := dp.DisplayTitle(ctx)
		if got != "テストタイトル" {
			t.Errorf("DisplayTitle() = %q, want %q", got, "テストタイトル")
		}
	})

	t.Run("タイトルが空の場合は無題を返す", func(t *testing.T) {
		t.Parallel()

		dp := viewmodel.DraftPageForSuggestionNew{
			ID:    model.DraftPageID("draft-2"),
			Title: "",
		}

		got := dp.DisplayTitle(ctx)
		if got == "" {
			t.Error("DisplayTitle() should not be empty for untitled draft")
		}
	})
}
