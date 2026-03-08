package components_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

func TestPaginationNav(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		pagination       viewmodel.Pagination
		previousPath     string
		nextPath         string
		wantRendered     bool
		wantPreviousLink bool
		wantNextLink     bool
	}{
		{
			name: "前後ページありの場合は両方のリンクが表示される",
			pagination: viewmodel.Pagination{
				Current:     2,
				Total:       3,
				HasNext:     true,
				HasPrevious: true,
			},
			previousPath:     "/s/test/topics/1?page=1",
			nextPath:         "/s/test/topics/1?page=3",
			wantRendered:     true,
			wantPreviousLink: true,
			wantNextLink:     true,
		},
		{
			name: "次ページのみの場合は前へボタンが無効になる",
			pagination: viewmodel.Pagination{
				Current:     1,
				Total:       2,
				HasNext:     true,
				HasPrevious: false,
			},
			previousPath:     "",
			nextPath:         "/s/test/topics/1?page=2",
			wantRendered:     true,
			wantPreviousLink: false,
			wantNextLink:     true,
		},
		{
			name: "前ページのみの場合は次へボタンが無効になる",
			pagination: viewmodel.Pagination{
				Current:     2,
				Total:       2,
				HasNext:     false,
				HasPrevious: true,
			},
			previousPath:     "/s/test/topics/1?page=1",
			nextPath:         "",
			wantRendered:     true,
			wantPreviousLink: true,
			wantNextLink:     false,
		},
		{
			name: "1ページしかない場合はナビゲーション自体が表示されない",
			pagination: viewmodel.Pagination{
				Current:     1,
				Total:       1,
				HasNext:     false,
				HasPrevious: false,
			},
			previousPath: "",
			nextPath:     "",
			wantRendered: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, "ja")

			var buf bytes.Buffer
			err := components.PaginationNav(tt.pagination, tt.previousPath, tt.nextPath).Render(ctx, &buf)
			if err != nil {
				t.Fatalf("レンダリングに失敗: %v", err)
			}

			html := buf.String()

			if !tt.wantRendered {
				if strings.Contains(html, "<nav") {
					t.Error("ナビゲーションが表示されるべきではないが、表示されている")
				}
				return
			}

			if !strings.Contains(html, "<nav") {
				t.Fatal("ナビゲーションが表示されるべきだが、表示されていない")
			}

			if !strings.Contains(html, "前へ") {
				t.Error("「前へ」テキストが含まれていない")
			}
			if !strings.Contains(html, "次へ") {
				t.Error("「次へ」テキストが含まれていない")
			}

			if tt.wantPreviousLink {
				if !strings.Contains(html, tt.previousPath) {
					t.Errorf("前ページへのリンク %q が含まれていない", tt.previousPath)
				}
			} else {
				if strings.Contains(html, `<a`) && strings.Contains(html, "前へ") {
					// disabled button が使われているか確認
					if !strings.Contains(html, "disabled") {
						t.Error("前へボタンが無効になっていない")
					}
				}
			}

			if tt.wantNextLink {
				if !strings.Contains(html, tt.nextPath) {
					t.Errorf("次ページへのリンク %q が含まれていない", tt.nextPath)
				}
			} else {
				if !strings.Contains(html, "disabled") {
					t.Error("次へボタンが無効になっていない")
				}
			}
		})
	}
}

func TestPaginationNav_英語ロケール(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "en")

	pagination := viewmodel.Pagination{
		Current:     1,
		Total:       2,
		HasNext:     true,
		HasPrevious: false,
	}

	var buf bytes.Buffer
	err := components.PaginationNav(pagination, "", "/page?page=2").Render(ctx, &buf)
	if err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}

	html := buf.String()

	if !strings.Contains(html, "Prev") {
		t.Error("英語の「Prev」テキストが含まれていない")
	}
	if !strings.Contains(html, "Next") {
		t.Error("英語の「Next」テキストが含まれていない")
	}
}
