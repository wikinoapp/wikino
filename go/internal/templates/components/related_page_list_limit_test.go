package components_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

func renderRelatedPageListComponent(t *testing.T, ctx context.Context, component templ.Component) string {
	t.Helper()

	var buf bytes.Buffer
	if err := component.Render(ctx, &buf); err != nil {
		t.Fatalf("render component: %v", err)
	}

	return buf.String()
}

// TestRelatedPageLists_LoadMoreCappedNotice pins that a listing stopped at the cumulative limit
// offers the first withheld page in the draft-backed, one-page editor. The stable focus id also
// gives htmx a destination when the preceding load-more link is replaced by the notice.
//
// [Ja] TestRelatedPageLists_LoadMoreCappedNotice は、累積上限で止まった一覧が、下書き由来の
// 1 ページ単位編集画面にある最初の省略ページへ案内することを固定する。安定したフォーカス ID により、
// 直前の「もっと見る」リンクを案内へ差し替えたときも htmx のフォーカス先を維持する。
func TestRelatedPageLists_LoadMoreCappedNotice(t *testing.T) {
	t.Parallel()

	state := viewmodel.PageLinkState{Context: viewmodel.PageLinkContextEdit}
	linkList := viewmodel.LinkList{
		Items: []viewmodel.LinkListItem{{
			CardLinkPage: viewmodel.CardLinkPage{Number: 12},
		}},
		Pagination:      viewmodel.Pagination{Current: 10, Total: 20},
		LoadMoreCapped:  true,
		SpaceIdentifier: viewmodel.SpaceIdentifier("my-space"),
		PageNumber:      1,
		State:           state,
	}
	backlinkList := viewmodel.BacklinkList{
		Items: []viewmodel.BacklinkListItem{{
			CardLinkPage: viewmodel.CardLinkPage{Number: 13},
		}},
		Pagination:       viewmodel.Pagination{Current: 10, Total: 20},
		LoadMoreCapped:   true,
		SpaceIdentifier:  viewmodel.SpaceIdentifier("my-space"),
		PageNumber:       1,
		ParentLinkPage:   1,
		LinkedPageNumber: 12,
		State:            state,
	}

	tests := []struct {
		name        string
		component   templ.Component
		wantURL     string
		wantFocusID string
	}{
		{
			name:        "link list",
			component:   components.LinkList(linkList),
			wantURL:     "/s/my-space/pages/1/edit?context=edit_paginated&amp;links_page=11#page-link-list-content",
			wantFocusID: "page-link-list-load-more",
		},
		{
			name:        "nested backlinks",
			component:   components.BacklinkList(backlinkList),
			wantURL:     "/s/my-space/pages/1/edit?context=edit_paginated&amp;linked_backlinks_page=11&amp;linked_page_number=12&amp;links_page=1#page-link-list-item-12",
			wantFocusID: "page-backlink-list-12-load-more",
		},
		{
			name:        "page backlinks",
			component:   components.PageBacklinkList(backlinkList),
			wantURL:     "/s/my-space/pages/1/edit?backlinks_page=11&amp;context=edit_paginated#page-backlink-list-content",
			wantFocusID: "page-backlink-list-load-more",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := i18n.SetLocale(t.Context(), i18n.LangJa)
			html := renderRelatedPageListComponent(t, ctx, tt.component)

			if !strings.Contains(html, "編集画面で読み込めるのはここまでです。") {
				t.Error("a capped listing does not say where it stops")
			}
			if !strings.Contains(html, "1ページずつ続きを見る") {
				t.Error("a capped listing does not describe the one-page continuation")
			}
			if !strings.Contains(html, `href="`+tt.wantURL+`"`) {
				t.Errorf("a capped listing does not link to its draft-backed continuation: want %q", tt.wantURL)
			}
			if !strings.Contains(html, `id="`+tt.wantFocusID+`"`) || !strings.Contains(html, `tabindex="-1"`) {
				t.Errorf("a capped listing does not retain a focusable target: want id %q", tt.wantFocusID)
			}
			if strings.Contains(html, "/link_list?") || strings.Contains(html, "/backlinks?") {
				t.Error("a capped listing must not keep offering the cumulative fragment endpoint")
			}
		})
	}
}

// TestRelatedPageLists_UncappedListingHasNoNotice pins that a listing which genuinely ran out has
// no notice, so the notice keeps meaning "there is more elsewhere".
//
// [Ja] TestRelatedPageLists_UncappedListingHasNoNotice は、本当に最後まで出た一覧には案内が付かない
// ことを固定する。案内が「続きが別の場所にある」という意味を保つためである。
func TestRelatedPageLists_UncappedListingHasNoNotice(t *testing.T) {
	t.Parallel()

	linkList := viewmodel.LinkList{
		Items: []viewmodel.LinkListItem{{
			CardLinkPage: viewmodel.CardLinkPage{Number: 12},
		}},
		Pagination:      viewmodel.Pagination{Current: 1, Total: 1},
		SpaceIdentifier: viewmodel.SpaceIdentifier("my-space"),
		PageNumber:      1,
		State:           viewmodel.PageLinkState{Context: viewmodel.PageLinkContextEdit},
	}

	ctx := i18n.SetLocale(t.Context(), i18n.LangJa)
	html := renderRelatedPageListComponent(t, ctx, components.LinkList(linkList))

	if strings.Contains(html, "編集画面で読み込めるのはここまでです。") {
		t.Error("a listing that ran out must not claim there is more elsewhere")
	}
	if strings.Contains(html, `role="status"`) {
		t.Error("an initially complete listing must not announce a dynamic update")
	}
}
