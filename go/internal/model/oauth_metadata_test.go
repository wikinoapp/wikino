package model_test

import (
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestSpaceAPIResourceMetadataURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		identifier model.SpaceIdentifier
		want       string
	}{
		{name: "well-knownの名前をオリジンとリソースのパスの間に挟む", identifier: "seed-wiki", want: "https://example.com/.well-known/oauth-protected-resource/api/v1/spaces/seed-wiki"},
		{name: "識別子をパスとしてエスケープする", identifier: `a"b`, want: "https://example.com/.well-known/oauth-protected-resource/api/v1/spaces/a%22b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := model.SpaceAPIResourceMetadataURL("https://example.com", tt.identifier); got != tt.want {
				t.Errorf("SpaceAPIResourceMetadataURL(%q) = %q、期待値 = %q", tt.identifier, got, tt.want)
			}
		})
	}
}

func TestSpaceIdentifierFromAPIPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		path   string
		want   model.SpaceIdentifier
		wantOK bool
	}{
		{name: "スペースのAPI", path: "/api/v1/spaces/seed-wiki", want: "seed-wiki", wantOK: true},
		{name: "スペース配下のAPI", path: "/api/v1/spaces/seed-wiki/pages/1", want: "seed-wiki", wantOK: true},
		{name: "エスケープを戻す", path: "/api/v1/spaces/a%22b/pages", want: `a"b`, wantOK: true},
		{name: "識別子が無い", path: "/api/v1/spaces/", wantOK: false},
		{name: "識別子が空", path: "/api/v1/spaces//pages", wantOK: false},
		{name: "エスケープが壊れている", path: "/api/v1/spaces/%zz", wantOK: false},
		{name: "スペースに属さないパス", path: "/api/v1/openapi.yaml", wantOK: false},
		{name: "Webの画面", path: "/s/seed-wiki", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := model.SpaceIdentifierFromAPIPath(tt.path)
			if ok != tt.wantOK || got != tt.want {
				t.Errorf("SpaceIdentifierFromAPIPath(%q) = (%q, %v)、期待値 = (%q, %v)", tt.path, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
