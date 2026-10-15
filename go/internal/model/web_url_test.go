package model_test

import (
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

func TestPageWebURL(t *testing.T) {
	t.Parallel()

	space := &model.Space{Identifier: "my-space"}

	if got, want := model.PageWebURL("https://example.com", space, 12), "https://example.com/s/my-space/pages/12"; got != want {
		t.Errorf("PageWebURL() = %q、期待値 = %q", got, want)
	}
	// Web画面のパスヘルパーと同じパスにする
	if got, want := model.PageWebURL("https://example.com", space, 12), "https://example.com"+string(templates.PagePath(viewmodel.SpaceIdentifier("my-space"), 12)); got != want {
		t.Errorf("PageWebURL() = %q、Web画面のパス = %q", got, want)
	}
}

func TestPageEditWebURL(t *testing.T) {
	t.Parallel()

	space := &model.Space{Identifier: "my-space"}

	if got, want := model.PageEditWebURL("https://example.com", space, 12), "https://example.com/s/my-space/pages/12/edit"; got != want {
		t.Errorf("PageEditWebURL() = %q、期待値 = %q", got, want)
	}
	// Web画面のパスヘルパーと同じパスにする
	if got, want := model.PageEditWebURL("https://example.com", space, 12), "https://example.com"+string(templates.PageEditPath(viewmodel.SpaceIdentifier("my-space"), 12)); got != want {
		t.Errorf("PageEditWebURL() = %q、Web画面のパス = %q", got, want)
	}
}
