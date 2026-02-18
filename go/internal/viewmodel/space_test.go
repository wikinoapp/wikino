package viewmodel_test

import (
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

func TestNewSpaceHeader(t *testing.T) {
	t.Parallel()

	space := &model.Space{
		ID:         "space-id",
		Identifier: "my-space",
		Name:       "マイスペース",
		Plan:       model.PlanFree,
	}

	header := viewmodel.NewSpaceHeader(space)

	if header.Name != "マイスペース" {
		t.Errorf("Name = %q, want %q", header.Name, "マイスペース")
	}

	if header.Identifier != "my-space" {
		t.Errorf("Identifier = %q, want %q", header.Identifier, "my-space")
	}
}
