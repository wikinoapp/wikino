package components_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
)

// renderTopNav renders c with a ja locale and returns the HTML.
//
// [Ja] renderTopNav は c を ja ロケールで描画し、HTML を返す。
func renderTopNav(t *testing.T, c templ.Component) string {
	t.Helper()

	ctx := i18n.SetLocale(context.Background(), "ja")

	var buf bytes.Buffer
	if err := c.Render(ctx, &buf); err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}
	return buf.String()
}

func TestTopNav_RendersBreadcrumbWithoutSidebarToggle(t *testing.T) {
	t.Parallel()

	data := components.TopNavData{
		Items: []components.BreadcrumbItem{
			{Label: "テストスペース", Path: templates.Path("/s/test")},
		},
	}
	html := renderTopNav(t, components.TopNav(data))

	// The breadcrumb landmark and its item render.
	// [Ja] パンくずのランドマークとその項目が描画される。
	if !strings.Contains(html, `aria-label="パンくずリスト"`) {
		t.Error("breadcrumb nav aria-label not found")
	}
	if !strings.Contains(html, "テストスペース") {
		t.Error("breadcrumb item label not found")
	}

	// The removed sidebar toggle leaves no button or dispatch behind.
	// [Ja] 廃止したサイドバー開閉ボタンは、ボタンも dispatch も残さない。
	if strings.Contains(html, "basecoat:sidebar") {
		t.Error("sidebar toggle dispatch should not be rendered")
	}
	if strings.Contains(html, "サイドバーの開閉") {
		t.Error("sidebar toggle button should not be rendered")
	}
}
