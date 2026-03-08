package viewmodel_test

import (
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
