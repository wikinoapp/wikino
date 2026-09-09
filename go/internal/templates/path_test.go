package templates_test

import (
	"testing"

	"github.com/wikinoapp/wikino/go/internal/templates"
)

func TestPaginatedPath(t *testing.T) {
	t.Parallel()

	base := templates.SpacePath("foo")

	tests := []struct {
		name string
		page int32
		want templates.Path
	}{
		{name: "1 ページ目はクエリを付けない", page: 1, want: "/s/foo"},
		{name: "2 ページ目以降はクエリを付ける", page: 2, want: "/s/foo?page=2"},
		{name: "0 以下はクエリを付けない", page: 0, want: "/s/foo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := templates.PaginatedPath(base, tt.page); got != tt.want {
				t.Errorf("PaginatedPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

// The closed tab lists a different set of suggestions than the open one, so it gets its own
// address. The open tab keeps the bare path it is linked by everywhere else.
//
// [Ja] クローズタブはオープンタブと載っている編集提案が異なるため、独自のアドレスを持つ。オープン
// タブは他の箇所からリンクされるときと同じクエリ無しのパスのままにする。
func TestSuggestionListTabPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		showClosed bool
		want       templates.Path
	}{
		{name: "オープンタブはクエリを付けない", showClosed: false, want: "/s/foo/topics/1/suggestions"},
		{name: "クローズタブはクエリを付ける", showClosed: true, want: "/s/foo/topics/1/suggestions?tab=closed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := templates.SuggestionListTabPath("foo", 1, tt.showClosed); got != tt.want {
				t.Errorf("SuggestionListTabPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

// A path that already carries a query must stay a valid URL, so the pagination parameter joins it
// with "&" instead of a second "?".
//
// [Ja] 既にクエリを持つパスも妥当な URL のままでなければならないため、ページネーションのパラメータは
// 2 つ目の "?" ではなく "&" で連結する。
func TestPaginatedPath_WithExistingQuery(t *testing.T) {
	t.Parallel()

	base := templates.SearchPathWithSpaceFilter("foo")

	if got, want := templates.PaginatedPath(base, 1), templates.Path("/search?q=space:foo"); got != want {
		t.Errorf("PaginatedPath() = %q, want %q", got, want)
	}
	if got, want := templates.PaginatedPath(base, 2), templates.Path("/search?q=space:foo&page=2"); got != want {
		t.Errorf("PaginatedPath() = %q, want %q", got, want)
	}
}
