package viewmodel_test

import (
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

func TestNewSpace(t *testing.T) {
	t.Parallel()

	space := &model.Space{
		ID:         "space-id",
		Identifier: "my-space",
		Name:       "マイスペース",
		Plan:       model.PlanFree,
	}

	vm := viewmodel.NewSpace(space)

	if vm.Name != "マイスペース" {
		t.Errorf("Name = %q, want %q", vm.Name, "マイスペース")
	}

	if vm.Identifier != "my-space" {
		t.Errorf("Identifier = %q, want %q", vm.Identifier, "my-space")
	}
}

func TestSpace_IconBackgroundColor_Deterministic(t *testing.T) {
	t.Parallel()

	identifiers := []string{"my-space", "another-space", "tech-blog", "a"}

	for _, id := range identifiers {
		space := viewmodel.Space{Identifier: viewmodel.SpaceIdentifier(id)}

		first := space.IconBackgroundColor()
		for i := 0; i < 5; i++ {
			got := space.IconBackgroundColor()
			if got != first {
				t.Errorf("IconBackgroundColor for %q is not deterministic: first=%q, got=%q", id, first, got)
			}
		}

		if !strings.HasPrefix(first, "#") {
			t.Errorf("IconBackgroundColor for %q = %q, want a hex color starting with #", id, first)
		}
	}
}

func TestSpace_IconBackgroundColor_Distribution(t *testing.T) {
	t.Parallel()

	identifiers := []string{
		"alpha", "beta", "gamma", "delta", "epsilon",
		"zeta", "eta", "theta", "iota", "kappa",
		"lambda", "mu", "nu", "xi", "omicron",
		"pi", "rho", "sigma", "tau", "upsilon",
		"phi", "chi", "psi", "omega",
	}

	seen := make(map[string]struct{})
	for _, id := range identifiers {
		space := viewmodel.Space{Identifier: viewmodel.SpaceIdentifier(id)}
		seen[space.IconBackgroundColor()] = struct{}{}
	}

	// At least 5 distinct colors are expected when 24 identifiers are mapped to a 12-color palette.
	// [Ja] 24 個の identifier を 12 色パレットに割り当てたとき、最低でも 5 色は分散することを期待する。
	if len(seen) < 5 {
		t.Errorf("expected colors to be distributed across the palette, got %d distinct colors", len(seen))
	}
}

func TestSpace_IconLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		identifier string
		want       string
	}{
		{name: "ASCII の先頭文字を大文字化", identifier: "my-space", want: "M"},
		{name: "大文字はそのまま", identifier: "Tech", want: "T"},
		{name: "数字はそのまま", identifier: "123", want: "1"},
		{name: "日本語はそのまま", identifier: "あいうえお", want: "あ"},
		{name: "空文字列は空文字列", identifier: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			space := viewmodel.Space{Identifier: viewmodel.SpaceIdentifier(tt.identifier)}
			if got := space.IconLabel(); got != tt.want {
				t.Errorf("IconLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}
