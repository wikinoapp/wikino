package model_test

import (
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestAPIPrincipal_IsBoundTo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		principal  *model.APIPrincipal
		identifier model.SpaceIdentifier
		want       bool
	}{
		{
			name:       "束縛先のスペースならtrue",
			principal:  &model.APIPrincipal{Space: &model.Space{Identifier: "alice-wiki"}},
			identifier: "alice-wiki",
			want:       true,
		},
		{
			name:       "束縛先と異なるスペースならfalse",
			principal:  &model.APIPrincipal{Space: &model.Space{Identifier: "alice-wiki"}},
			identifier: "bob-wiki",
			want:       false,
		},
		{
			name:       "スペースを持たない主体ならfalse",
			principal:  &model.APIPrincipal{},
			identifier: "alice-wiki",
			want:       false,
		},
		{
			name:       "主体が無ければfalse",
			principal:  nil,
			identifier: "alice-wiki",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.principal.IsBoundTo(tt.identifier); got != tt.want {
				t.Errorf("IsBoundTo(%q) = %v、期待値 = %v", tt.identifier, got, tt.want)
			}
		})
	}
}
