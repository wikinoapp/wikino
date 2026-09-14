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

			if !strings.Contains(html, "編集画面で読み込めるのはここまでです") {
				t.Error("a capped listing does not say where it stops")
			}
			if strings.Contains(html, "編集画面で読み込めるのはここまでです。") {
				t.Error("the Japanese capped notice keeps a full stop the rest of the messages do not use")
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

	if strings.Contains(html, "編集画面で読み込めるのはここまでです") {
		t.Error("a listing that ran out must not claim there is more elsewhere")
	}
	if strings.Contains(html, `role="status"`) {
		t.Error("an initially complete listing must not announce a dynamic update")
	}
}

// TestRelatedPageLists_EndNotice pins the completion status a reader reaches by walking a listing
// to its end: no full stop in Japanese, and a decorative confetti icon whose currentColor follows
// the muted text rather than painting itself black.
//
// [Ja] TestRelatedPageLists_EndNotice は、一覧を最後まで辿った読み手に出る完了状態を固定する。
// 日本語では句点を付けず、装飾の confetti アイコンは自前で黒く塗らず currentColor で抑えた文字色に
// 追従する。
func TestRelatedPageLists_EndNotice(t *testing.T) {
	t.Parallel()

	state := viewmodel.PageLinkState{Context: viewmodel.PageLinkContextShow}
	linkList := viewmodel.LinkList{
		Items: []viewmodel.LinkListItem{{
			CardLinkPage: viewmodel.CardLinkPage{Number: 12},
		}},
		Pagination:      viewmodel.Pagination{Current: 2, Total: 2},
		SpaceIdentifier: viewmodel.SpaceIdentifier("my-space"),
		PageNumber:      1,
		State:           state,
	}
	backlinkList := viewmodel.BacklinkList{
		Items: []viewmodel.BacklinkListItem{{
			CardLinkPage: viewmodel.CardLinkPage{Number: 13},
		}},
		Pagination:       viewmodel.Pagination{Current: 2, Total: 2},
		SpaceIdentifier:  viewmodel.SpaceIdentifier("my-space"),
		PageNumber:       1,
		ParentLinkPage:   1,
		LinkedPageNumber: 12,
		State:            state,
	}

	tests := []struct {
		name        string
		component   templ.Component
		wantFocusID string
	}{
		{
			name:        "link list",
			component:   components.LinkListResponse(linkList),
			wantFocusID: "page-link-list-load-more",
		},
		{
			name:        "nested backlinks",
			component:   components.BacklinkListResponse(backlinkList),
			wantFocusID: "page-backlink-list-12-load-more",
		},
		{
			name:        "page backlinks",
			component:   components.PageBacklinkListResponse(backlinkList),
			wantFocusID: "page-backlink-list-load-more",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := i18n.SetLocale(t.Context(), i18n.LangJa)
			html := renderRelatedPageListComponent(t, ctx, tt.component)

			if !strings.Contains(html, "すべて読み込みました") {
				t.Error("a listing walked to its end does not say so")
			}
			if strings.Contains(html, "すべて読み込みました。") {
				t.Error("the Japanese notice keeps a full stop the rest of the messages do not use")
			}
			if !strings.Contains(html, `id="`+tt.wantFocusID+`"`) || !strings.Contains(html, `tabindex="-1"`) {
				t.Errorf("the notice does not retain a focusable target: want id %q", tt.wantFocusID)
			}
			if !strings.Contains(html, `role="status"`) {
				t.Error("the notice does not announce the completed load")
			}

			icon := endNoticeIcon(t, html)
			if !strings.Contains(icon, `aria-hidden="true"`) {
				t.Error("the confetti icon is not hidden from assistive technology, so role=\"status\" would read more than the text")
			}
			if !strings.Contains(icon, `fill="currentColor"`) {
				t.Errorf("the confetti icon does not inherit the muted text colour: %s", icon)
			}

			// The path fragment is a literal taken from the confetti icon rather than a second
			// lookup in phosphorIcons, so that a black fill restored in the map cannot make both
			// sides of the comparison agree. Without it an unknown name passes every assertion
			// above, because iconSVG falls back to info-regular and that icon is decorative and
			// currentColor too.
			//
			// [Ja] パス断片は phosphorIcons を二度引かず confetti のリテラルから取る。マップの fill が
			// 黒へ戻ったときに比較の両辺が揃ってしまうのを避けるためである。これが無いと、名前を誤っても
			// 上のアサーションはすべて通る。iconSVG が info-regular へフォールバックし、そのアイコンも
			// 装飾かつ currentColor であるためである。
			if !strings.Contains(icon, "M111.49,52.63") {
				t.Error("the notice carries some other icon, so an unknown name silently fell back to info-regular")
			}

			notice := strings.Index(html, `role="status"`)
			if strings.Index(html[notice:], "<svg") > strings.Index(html[notice:], "すべて読み込みました") {
				t.Error("the confetti icon does not lead the text it decorates")
			}
		})
	}
}

// TestRelatedPageLists_EnglishNoticesKeepFullStops pins that dropping the Japanese full stop did
// not travel to English, where a declarative sentence ends in a period. It covers both notices the
// same task touched, because a bulk replacement would reach them together.
//
// [Ja] TestRelatedPageLists_EnglishNoticesKeepFullStops は、日本語の句点を落とした判断が英語へ
// 及んでいないことを固定する。英語の平叙文はピリオドで終えるためである。一括置換は 2 つの文言へ
// まとめて届くため、同じタスクで触れた両方を対象にする。
func TestRelatedPageLists_EnglishNoticesKeepFullStops(t *testing.T) {
	t.Parallel()

	endLinkList := viewmodel.LinkList{
		Items: []viewmodel.LinkListItem{{
			CardLinkPage: viewmodel.CardLinkPage{Number: 12},
		}},
		Pagination:      viewmodel.Pagination{Current: 2, Total: 2},
		SpaceIdentifier: viewmodel.SpaceIdentifier("my-space"),
		PageNumber:      1,
		State:           viewmodel.PageLinkState{Context: viewmodel.PageLinkContextShow},
	}
	cappedLinkList := viewmodel.LinkList{
		Items: []viewmodel.LinkListItem{{
			CardLinkPage: viewmodel.CardLinkPage{Number: 12},
		}},
		Pagination:      viewmodel.Pagination{Current: 10, Total: 20},
		LoadMoreCapped:  true,
		SpaceIdentifier: viewmodel.SpaceIdentifier("my-space"),
		PageNumber:      1,
		State:           viewmodel.PageLinkState{Context: viewmodel.PageLinkContextEdit},
	}

	ctx := i18n.SetLocale(t.Context(), i18n.LangEn)

	endHTML := renderRelatedPageListComponent(t, ctx, components.LinkListResponse(endLinkList))
	if !strings.Contains(endHTML, "All items have been loaded.") {
		t.Error("the English completion notice lost the period a declarative sentence ends with")
	}

	cappedHTML := renderRelatedPageListComponent(t, ctx, components.LinkList(cappedLinkList))
	if !strings.Contains(cappedHTML, "This is as far as the editor loads this list.") {
		t.Error("the English capped notice lost the period a declarative sentence ends with")
	}
}

// endNoticeIcon returns the svg element that opens the completion notice, so that a test can assert
// on the icon alone rather than on a page that also carries the cards' own icons.
//
// [Ja] endNoticeIcon は完了の案内を開く svg 要素を返す。カード側のアイコンも載ったページ全体ではなく、
// この案内のアイコンだけをテストが検査できるようにするためである。
func endNoticeIcon(t *testing.T, html string) string {
	t.Helper()

	notice := strings.Index(html, `role="status"`)
	if notice < 0 {
		t.Fatal("the notice is absent, so it carries no icon")
	}

	icon := svgSegment(html[notice:])
	if icon == "" {
		t.Fatal("the notice carries no icon")
	}

	return icon
}
