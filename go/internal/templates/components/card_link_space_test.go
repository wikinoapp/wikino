package components_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

func TestCardLinkSpace_スペースへリンクする(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	space := viewmodel.Space{Name: "マイスペース", Identifier: "my-space"}
	if err := components.CardLinkSpace(space).Render(context.Background(), &buf); err != nil {
		t.Fatalf("Render()のエラー = %v", err)
	}

	html := buf.String()
	for _, want := range []string{`href="/s/my-space"`, "マイスペース"} {
		if !strings.Contains(html, want) {
			t.Errorf("出力に %q が含まれていない", want)
		}
	}
}
