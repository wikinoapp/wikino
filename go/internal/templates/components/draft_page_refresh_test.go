package components_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

func TestDraftPageRefreshTrigger_InitialState(t *testing.T) {
	t.Parallel()

	state := viewmodel.PageLinkState{
		Context:            viewmodel.PageLinkContextEdit,
		LinkPage:           2,
		LinkedPageNumber:   9,
		LinkedBacklinkPage: 3,
		PageBacklinkPage:   4,
	}

	html := renderDraftPageRefreshComponent(t, components.DraftPageRefreshTrigger(
		viewmodel.SpaceIdentifier("my-space"),
		1,
		state,
	))

	// The refresh URL carries no state of its own; hx-include reads it from the shared element at
	// request time, so a listing advanced after this render is still sent.
	//
	// [Ja] 再取得 URL は状態を持たない。hx-include がリクエスト時に共有要素から読むため、この描画の
	// あとに進めた一覧も送られる。
	if !strings.Contains(html, `hx-get="/s/my-space/pages/1/draft_page"`) {
		t.Error("draft refresh URL must not pin the state it was rendered with")
	}
	if !strings.Contains(html, `hx-include="#page-related-page-state"`) {
		t.Error("draft refresh trigger does not read the shared state")
	}
	if !strings.Contains(html, `hx-sync="#page-related-page-state:drop"`) {
		t.Error("automatic refresh should yield to a manual request in the shared sync group")
	}
	if strings.Contains(html, "hx-swap-oob") {
		t.Error("the initial render must not be marked as an OOB response")
	}
	for _, want := range []string{
		`name="context" value="edit"`,
		`name="links_page" value="2"`,
		`name="linked_page_number" value="9"`,
		`name="linked_backlinks_page" value="3"`,
		`name="backlinks_page" value="4"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("shared state does not expose %q", want)
		}
	}
}

// TestDraftPageRefreshTrigger_FirstPageStateKeepsElementIDs pins that every state element exists
// even while its listing sits on its first page. An out-of-band swap can only replace an id that is
// already in the DOM, so a listing whose element were dropped could never be advanced.
//
// [Ja] TestDraftPageRefreshTrigger_FirstPageStateKeepsElementIDs は、一覧が 1 ページ目にあるときも
// 各状態要素が存在することを固定する。OOB スワップは既に DOM にある id しか差し替えられないため、
// 要素が落ちた一覧は以後進められなくなる。
func TestDraftPageRefreshTrigger_FirstPageStateKeepsElementIDs(t *testing.T) {
	t.Parallel()

	html := renderDraftPageRefreshComponent(t, components.DraftPageRefreshTrigger(
		viewmodel.SpaceIdentifier("my-space"),
		1,
		viewmodel.PageLinkState{Context: viewmodel.PageLinkContextEdit},
	))

	for _, want := range []string{
		`id="page-link-list-state"`,
		`id="page-nested-backlink-state"`,
		`id="page-backlink-list-state"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("shared state does not render %q on the first page", want)
		}
	}

	// A first-page listing sends an empty value, which the handlers read as page one. The card
	// number is empty as well, because a card whose nested list is on its first page is not open.
	//
	// [Ja] 1 ページ目の一覧は空の値を送り、Handler はこれを 1 ページ目として読む。ネストした一覧が
	// 1 ページ目のカードは開かれていないため、カード番号も空にする。
	for _, want := range []string{
		`name="links_page" value=""`,
		`name="linked_page_number" value=""`,
		`name="linked_backlinks_page" value=""`,
		`name="backlinks_page" value=""`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("shared state does not render %q on the first page", want)
		}
	}
}

// TestRelatedPageListResponses_AdvanceOnlyTheirOwnState pins that a fragment response rewrites the
// state of the listing it advanced and nothing else. Two requests can be in flight at once, so a
// response that replaced the whole shared state would write back the values it read before the
// other request advanced them.
//
// [Ja] TestRelatedPageListResponses_AdvanceOnlyTheirOwnState は、フラグメント応答が自分の進めた
// 一覧の状態だけを書き換えることを固定する。2 つのリクエストは同時に走りうるため、共有状態を丸ごと
// 差し替える応答は、もう一方が進める前に読んだ値を書き戻してしまう。
func TestRelatedPageListResponses_AdvanceOnlyTheirOwnState(t *testing.T) {
	t.Parallel()

	state := viewmodel.PageLinkState{
		Context:            viewmodel.PageLinkContextEdit,
		LinkPage:           2,
		LinkedPageNumber:   9,
		LinkedBacklinkPage: 3,
		PageBacklinkPage:   4,
	}
	linkList := viewmodel.LinkList{
		SpaceIdentifier: viewmodel.SpaceIdentifier("my-space"),
		PageNumber:      1,
		State:           state,
	}
	backlinkList := viewmodel.BacklinkList{
		SpaceIdentifier:  viewmodel.SpaceIdentifier("my-space"),
		PageNumber:       1,
		ParentLinkPage:   2,
		LinkedPageNumber: 9,
		State:            state,
	}

	tests := []struct {
		name      string
		component templ.Component
		wantID    string
		want      []string
		notWant   []string
	}{
		{
			name:      "link list",
			component: components.LinkListResponse(linkList),
			wantID:    `id="page-link-list-state"`,
			want:      []string{`name="links_page" value="2"`},
			notWant:   []string{`name="linked_page_number"`, `name="linked_backlinks_page"`, `name="backlinks_page"`},
		},
		{
			name:      "nested backlink list",
			component: components.BacklinkListResponse(backlinkList),
			wantID:    `id="page-nested-backlink-state"`,
			want:      []string{`name="linked_page_number" value="9"`, `name="linked_backlinks_page" value="3"`},
			notWant:   []string{`name="links_page"`, `name="backlinks_page"`},
		},
		{
			name:      "page backlink list",
			component: components.PageBacklinkListResponse(backlinkList),
			wantID:    `id="page-backlink-list-state"`,
			want:      []string{`name="backlinks_page" value="4"`},
			notWant:   []string{`name="links_page"`, `name="linked_page_number"`, `name="linked_backlinks_page"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			html := renderDraftPageRefreshComponent(t, tt.component)
			if !strings.Contains(html, tt.wantID) {
				t.Errorf("response does not update %s", tt.wantID)
			}
			if !strings.Contains(html, `hx-swap-oob="outerHTML"`) {
				t.Error("response does not replace its state element out of band")
			}
			for _, want := range tt.want {
				if !strings.Contains(html, want) {
					t.Errorf("response does not advance its own state %q", want)
				}
			}
			for _, notWant := range tt.notWant {
				if strings.Contains(html, notWant) {
					t.Errorf("response rewrites the state of a listing it did not advance: %q", notWant)
				}
			}
		})
	}
}

// TestEditorLoadMoreLinksReadSharedState verifies the contract that lets links already in the DOM
// observe state advanced by another list: fragment URLs contain only their own next page, and the
// shared OOB-updated state supplies every cross-list value at request time.
//
// [Ja] TestEditorLoadMoreLinksReadSharedState は、DOM に残ったリンクが別一覧の進めた状態を読める契約を
// 確認する。フラグメント URL は自身の次ページだけを持ち、一覧間の値は OOB 更新される共有状態が
// リクエスト時に渡す。
func TestEditorLoadMoreLinksReadSharedState(t *testing.T) {
	t.Parallel()

	state := viewmodel.PageLinkState{
		Context:            viewmodel.PageLinkContextEdit,
		LinkPage:           2,
		LinkedPageNumber:   9,
		LinkedBacklinkPage: 3,
		PageBacklinkPage:   4,
	}
	linkList := viewmodel.LinkList{
		Items: []viewmodel.LinkListItem{{
			CardLinkPage: viewmodel.CardLinkPage{Number: 12},
		}},
		Pagination:      viewmodel.NewPagination(2, 60, int(viewmodel.LinkLimit)),
		SpaceIdentifier: viewmodel.SpaceIdentifier("my-space"),
		PageNumber:      1,
		State:           state,
	}
	nestedBacklinks := viewmodel.BacklinkList{
		Pagination:       viewmodel.NewPagination(3, 60, int(viewmodel.BacklinkLimit)),
		SpaceIdentifier:  viewmodel.SpaceIdentifier("my-space"),
		PageNumber:       1,
		ParentLinkPage:   2,
		LinkedPageNumber: 9,
		State:            state,
	}
	pageBacklinks := viewmodel.BacklinkList{
		Pagination:      viewmodel.NewPagination(4, 80, int(viewmodel.PageBacklinkLimit)),
		SpaceIdentifier: viewmodel.SpaceIdentifier("my-space"),
		PageNumber:      1,
		State:           state,
	}

	tests := []struct {
		name        string
		component   templ.Component
		wantGet     string
		wantFocusID string
	}{
		{
			name:        "link list",
			component:   components.LinkList(linkList),
			wantGet:     `hx-get="/s/my-space/pages/1/link_list?context=edit&amp;page=3"`,
			wantFocusID: "page-link-list-load-more",
		},
		{
			name:      "nested backlinks",
			component: components.BacklinkList(nestedBacklinks),
			// The card's own link-list page uses the fragment-scoped name, so the shared state this
			// link also sends cannot replace it with the page of another card.
			//
			// [Ja] カード自身のリンク一覧ページはフラグメント側の名前を使う。このリンクが一緒に送る共有
			// 状態が、別のカードのページで置き換えられないようにするためである。
			wantGet:     `hx-get="/s/my-space/pages/1/links/9/backlink_list?context=edit&amp;page=4&amp;parent_page=2"`,
			wantFocusID: "page-backlink-list-9-load-more",
		},
		{
			name:        "page backlinks",
			component:   components.PageBacklinkList(pageBacklinks),
			wantGet:     `hx-get="/s/my-space/pages/1/backlinks?context=edit&amp;page=5"`,
			wantFocusID: "page-backlink-list-load-more",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			html := renderDraftPageRefreshComponent(t, tt.component)
			if !strings.Contains(html, tt.wantGet) {
				t.Errorf("load-more link does not isolate its own next page: want %q", tt.wantGet)
			}
			if !strings.Contains(html, `hx-include="#page-related-page-state"`) {
				t.Error("load-more link does not read the shared editor state")
			}
			if !strings.Contains(html, `hx-sync="#page-related-page-state:replace"`) {
				t.Error("manual load-more request should replace an automatic refresh in the shared sync group")
			}
			if !strings.Contains(html, `id="`+tt.wantFocusID+`"`) {
				t.Errorf("load-more link does not expose stable focus id %q", tt.wantFocusID)
			}
		})
	}
}

// TestRelatedPageListResponses_RetainStableFocusTarget pins the final-page side of htmx focus
// restoration. The response replaces the focused load-more link, so its completion status must
// inherit that link's stable id and remain programmatically focusable.
//
// [Ja] TestRelatedPageListResponses_RetainStableFocusTarget は、最終ページでの htmx フォーカス復元を
// 固定する。応答はフォーカス中の「もっと見る」リンクを置き換えるため、完了状態が同じ安定 ID を引き継ぎ、
// プログラムからフォーカス可能なままでなければならない。
func TestRelatedPageListResponses_RetainStableFocusTarget(t *testing.T) {
	t.Parallel()

	state := viewmodel.PageLinkState{
		Context:            viewmodel.PageLinkContextEdit,
		LinkPage:           2,
		LinkedPageNumber:   9,
		LinkedBacklinkPage: 2,
		PageBacklinkPage:   2,
	}
	linkList := viewmodel.LinkList{
		Items:      []viewmodel.LinkListItem{{CardLinkPage: viewmodel.CardLinkPage{Number: 20}}},
		Pagination: viewmodel.Pagination{Current: 2, Total: 2},
		State:      state,
	}
	nestedBacklinks := viewmodel.BacklinkList{
		Items:            []viewmodel.BacklinkListItem{{CardLinkPage: viewmodel.CardLinkPage{Number: 21}}},
		Pagination:       viewmodel.Pagination{Current: 2, Total: 2},
		LinkedPageNumber: 9,
		State:            state,
	}
	pageBacklinks := viewmodel.BacklinkList{
		Items:      []viewmodel.BacklinkListItem{{CardLinkPage: viewmodel.CardLinkPage{Number: 22}}},
		Pagination: viewmodel.Pagination{Current: 2, Total: 2},
		State:      state,
	}

	tests := []struct {
		name      string
		component templ.Component
		focusID   string
	}{
		{name: "link list", component: components.LinkListResponse(linkList), focusID: "page-link-list-load-more"},
		{name: "nested backlinks", component: components.BacklinkListResponse(nestedBacklinks), focusID: "page-backlink-list-9-load-more"},
		{name: "page backlinks", component: components.PageBacklinkListResponse(pageBacklinks), focusID: "page-backlink-list-load-more"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			html := renderDraftPageRefreshComponent(t, tt.component)
			for _, want := range []string{`id="` + tt.focusID + `"`, `tabindex="-1"`, `role="status"`} {
				if !strings.Contains(html, want) {
					t.Errorf("final response does not contain %q", want)
				}
			}
			if strings.Contains(html, "hx-get=") {
				t.Error("final response must not render another load-more request")
			}
		})
	}
}

// TestPaginatedEditorRelatedPageResponsesReplaceWholeListing pins the one-page editor response
// shape. Every fragment replaces its whole stable listing wrapper and keeps context=edit_paginated
// in both htmx and ordinary-link navigation beyond the cumulative limit.
//
// [Ja] TestPaginatedEditorRelatedPageResponsesReplaceWholeListing は、1 ページ単位編集画面の応答形を
// 固定する。各フラグメントは安定した一覧ラッパー全体を差し替え、累積上限より先でも htmx と通常リンクの
// 双方が context=edit_paginated を維持する。
func TestPaginatedEditorRelatedPageResponsesReplaceWholeListing(t *testing.T) {
	t.Parallel()

	state := viewmodel.PageLinkState{
		Context:            viewmodel.PageLinkContextEditPaginated,
		LinkPage:           11,
		LinkedPageNumber:   9,
		LinkedBacklinkPage: 11,
		PageBacklinkPage:   11,
	}
	linkList := viewmodel.LinkList{
		Items:           []viewmodel.LinkListItem{{CardLinkPage: viewmodel.CardLinkPage{Number: 20}}},
		Pagination:      viewmodel.Pagination{Current: 11, Total: 12, HasNext: true},
		SpaceIdentifier: viewmodel.SpaceIdentifier("my-space"),
		PageNumber:      1,
		State:           state,
	}
	nestedBacklinks := viewmodel.BacklinkList{
		Items:            []viewmodel.BacklinkListItem{{CardLinkPage: viewmodel.CardLinkPage{Number: 21}}},
		Pagination:       viewmodel.Pagination{Current: 11, Total: 12, HasNext: true},
		SpaceIdentifier:  viewmodel.SpaceIdentifier("my-space"),
		PageNumber:       1,
		ParentLinkPage:   11,
		LinkedPageNumber: 9,
		State:            state,
	}
	pageBacklinks := viewmodel.BacklinkList{
		Items:           []viewmodel.BacklinkListItem{{CardLinkPage: viewmodel.CardLinkPage{Number: 22}}},
		Pagination:      viewmodel.Pagination{Current: 11, Total: 12, HasNext: true},
		SpaceIdentifier: viewmodel.SpaceIdentifier("my-space"),
		PageNumber:      1,
		State:           state,
	}

	tests := []struct {
		name      string
		component templ.Component
		wrapperID string
	}{
		{name: "link list", component: components.LinkListResponse(linkList), wrapperID: "page-link-list-content"},
		{name: "nested backlinks", component: components.BacklinkListResponse(nestedBacklinks), wrapperID: "page-backlink-list-9-content"},
		{name: "page backlinks", component: components.PageBacklinkListResponse(pageBacklinks), wrapperID: "page-backlink-list-content"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			html := renderDraftPageRefreshComponent(t, tt.component)
			if !strings.Contains(html, `id="`+tt.wrapperID+`"`) {
				t.Errorf("paginated response does not render whole-list wrapper %q", tt.wrapperID)
			}
			if !strings.Contains(html, `hx-target="#`+tt.wrapperID+`"`) {
				t.Errorf("successor link does not replace whole-list wrapper %q", tt.wrapperID)
			}
			if !strings.Contains(html, "context=edit_paginated") {
				t.Error("paginated response loses its editor context")
			}
			if strings.Contains(html, "読み込めるのはここまで") {
				t.Error("one-page editor response must not reapply the cumulative limit")
			}
		})
	}
}

func TestRelatedPageListResponses_OmitSharedStateOnShow(t *testing.T) {
	t.Parallel()

	linkList := viewmodel.LinkList{
		State: viewmodel.PageLinkState{Context: viewmodel.PageLinkContextShow},
	}
	backlinkList := viewmodel.BacklinkList{
		State: viewmodel.PageLinkState{Context: viewmodel.PageLinkContextShow},
	}

	tests := []struct {
		name      string
		component templ.Component
	}{
		{name: "link list", component: components.LinkListResponse(linkList)},
		{name: "nested backlink list", component: components.BacklinkListResponse(backlinkList)},
		{name: "page backlink list", component: components.PageBacklinkListResponse(backlinkList)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			html := renderDraftPageRefreshComponent(t, tt.component)
			if strings.Contains(html, "hx-swap-oob") {
				t.Error("public page fragment must not swap the editor's shared state")
			}
			if strings.Contains(html, "page-draft-refresh-trigger") {
				t.Error("public page fragment must not render the editor's draft refresh trigger")
			}
		})
	}
}

func renderDraftPageRefreshComponent(t *testing.T, component templ.Component) string {
	t.Helper()

	var buf bytes.Buffer
	if err := component.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render component: %v", err)
	}

	return buf.String()
}
