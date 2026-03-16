package viewmodel_test

import (
	"testing"
	"time"

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
