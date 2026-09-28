package model_test

import (
	"slices"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestSortAPITokenScopes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		scopes []model.Scope
		want   []model.Scope
	}{
		{
			name:   "文字列の順に保存されたスコープをAPITokenScopesの順に並べる",
			scopes: []model.Scope{model.ScopePageRead, model.ScopePageWrite, model.ScopeTopicRead},
			want:   []model.Scope{model.ScopeTopicRead, model.ScopePageRead, model.ScopePageWrite},
		},
		{
			name:   "重複を除く",
			scopes: []model.Scope{model.ScopePageWrite, model.ScopePageWrite, model.ScopeTopicRead},
			want:   []model.Scope{model.ScopeTopicRead, model.ScopePageWrite},
		},
		{
			name:   "APITokenScopesに無いスコープは元の順のまま末尾に置く",
			scopes: []model.Scope{"unknown:read", model.ScopePageRead, model.ScopeSpaceAdmin},
			want:   []model.Scope{model.ScopePageRead, "unknown:read", model.ScopeSpaceAdmin},
		},
		{
			name:   "空",
			scopes: []model.Scope{},
			want:   []model.Scope{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := model.SortAPITokenScopes(tt.scopes)
			if !slices.Equal(got, tt.want) {
				t.Errorf("SortAPITokenScopes() = %v、期待値 = %v", got, tt.want)
			}
		})
	}
}
