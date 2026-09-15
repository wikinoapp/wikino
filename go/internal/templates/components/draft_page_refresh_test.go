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

	// 再取得URLは状態を持たない。hx-includeがリクエスト時に共有要素から読むため、この描画の
	// あとに進めた一覧も送られる。
	if !strings.Contains(html, `hx-get="/s/my-space/pages/1/draft_page"`) {
		t.Error("下書きの更新URLが描画時の状態に固定されている")
	}
	if !strings.Contains(html, `hx-include="#page-related-page-state"`) {
		t.Error("下書きの更新トリガーが共有の状態を読んでいない")
	}
	if !strings.Contains(html, `hx-sync="#page-related-page-state:drop"`) {
		t.Error("共有の同期グループで自動更新が手動のリクエストに譲っていない")
	}
	if strings.Contains(html, "hx-swap-oob") {
		t.Error("初回の描画がOOBの応答として印付けされている")
	}
	for _, want := range []string{
		`name="context" value="edit"`,
		`name="links_page" value="2"`,
		`name="linked_page_number" value="9"`,
		`name="linked_backlinks_page" value="3"`,
		`name="backlinks_page" value="4"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("共有の状態に%qが無い", want)
		}
	}
}

// TestDraftPageRefreshTrigger_FirstPageStateKeepsElementIDsは、一覧が1ページ目にあるときも
// 各状態要素が存在することを固定する。OOBスワップは既にDOMにあるidしか差し替えられないため、
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
			t.Errorf("最初のページで共有の状態に%qが描画されていない", want)
		}
	}

	// 1ページ目の一覧は空の値を送り、Handlerはこれを1ページ目として読む。ネストした一覧が
	// 1ページ目のカードは開かれていないため、カード番号も空にする。
	for _, want := range []string{
		`name="links_page" value=""`,
		`name="linked_page_number" value=""`,
		`name="linked_backlinks_page" value=""`,
		`name="backlinks_page" value=""`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("最初のページで共有の状態に%qが描画されていない", want)
		}
	}
}

// TestRelatedPageListResponses_AdvanceOnlyTheirOwnStateは、フラグメント応答が自分の進めた
// 一覧の状態だけを書き換えることを固定する。2つのリクエストは同時に走りうるため、共有状態を丸ごと
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
			name:      "リンクの一覧",
			component: components.LinkListResponse(linkList),
			wantID:    `id="page-link-list-state"`,
			want:      []string{`name="links_page" value="2"`},
			notWant:   []string{`name="linked_page_number"`, `name="linked_backlinks_page"`, `name="backlinks_page"`},
		},
		{
			name:      "入れ子のバックリンクの一覧",
			component: components.BacklinkListResponse(backlinkList),
			wantID:    `id="page-nested-backlink-state"`,
			want:      []string{`name="linked_page_number" value="9"`, `name="linked_backlinks_page" value="3"`},
			notWant:   []string{`name="links_page"`, `name="backlinks_page"`},
		},
		{
			name:      "ページのバックリンクの一覧",
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
				t.Errorf("応答が%sを更新していない", tt.wantID)
			}
			if !strings.Contains(html, `hx-swap-oob="outerHTML"`) {
				t.Error("応答が状態の要素をアウトオブバンドで置き換えていない")
			}
			for _, want := range tt.want {
				if !strings.Contains(html, want) {
					t.Errorf("応答が自身の状態%qを進めていない", want)
				}
			}
			for _, notWant := range tt.notWant {
				if strings.Contains(html, notWant) {
					t.Errorf("応答が進めていない一覧の状態を書き換えている: %q", notWant)
				}
			}
		})
	}
}

// TestEditorLoadMoreLinksReadSharedStateは、DOMに残ったリンクが別一覧の進めた状態を読める契約を
// 確認する。フラグメントURLは自身の次ページだけを持ち、一覧間の値はOOB更新される共有状態が
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
			name:        "リンクの一覧",
			component:   components.LinkList(linkList),
			wantGet:     `hx-get="/s/my-space/pages/1/link_list?context=edit&amp;page=3"`,
			wantFocusID: "page-link-list-load-more",
		},
		{
			name:      "入れ子のバックリンク",
			component: components.BacklinkList(nestedBacklinks),
			// カード自身のリンク一覧ページはフラグメント側の名前を使う。このリンクが一緒に送る共有
			// 状態が、別のカードのページで置き換えられないようにするためである。
			wantGet:     `hx-get="/s/my-space/pages/1/links/9/backlink_list?context=edit&amp;page=4&amp;parent_page=2"`,
			wantFocusID: "page-backlink-list-9-load-more",
		},
		{
			name:        "ページのバックリンク",
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
				t.Errorf("さらに読み込むリンクが自身の次のページを分けて指していない: 期待値 = %q", tt.wantGet)
			}
			if !strings.Contains(html, `hx-include="#page-related-page-state"`) {
				t.Error("さらに読み込むリンクが共有のエディタの状態を読んでいない")
			}
			if !strings.Contains(html, `hx-sync="#page-related-page-state:replace"`) {
				t.Error("共有の同期グループで手動のさらに読み込むリクエストが自動更新を置き換えていない")
			}
			if !strings.Contains(html, `id="`+tt.wantFocusID+`"`) {
				t.Errorf("さらに読み込むリンクに固定のフォーカス用id %qが無い", tt.wantFocusID)
			}
		})
	}
}

// TestRelatedPageListResponses_RetainStableFocusTargetは、最終ページでのhtmxフォーカス復元を
// 固定する。応答はフォーカス中の「もっと見る」リンクを置き換えるため、完了状態が同じ安定IDを引き継ぎ、
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
		{name: "リンクの一覧", component: components.LinkListResponse(linkList), focusID: "page-link-list-load-more"},
		{name: "入れ子のバックリンク", component: components.BacklinkListResponse(nestedBacklinks), focusID: "page-backlink-list-9-load-more"},
		{name: "ページのバックリンク", component: components.PageBacklinkListResponse(pageBacklinks), focusID: "page-backlink-list-load-more"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			html := renderDraftPageRefreshComponent(t, tt.component)
			for _, want := range []string{`id="` + tt.focusID + `"`, `tabindex="-1"`, `role="status"`} {
				if !strings.Contains(html, want) {
					t.Errorf("最後の応答に%qが含まれていない", want)
				}
			}
			if strings.Contains(html, "hx-get=") {
				t.Error("最後の応答にさらに読み込むリクエストが描画されている")
			}
		})
	}
}

// TestPaginatedEditorRelatedPageResponsesReplaceWholeListingは、1ページ単位編集画面の応答形を
// 固定する。各フラグメントは安定した一覧ラッパー全体を差し替え、累積上限より先でもhtmxと通常リンクの
// 双方がcontext=edit_paginatedを維持する。
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
		{name: "リンクの一覧", component: components.LinkListResponse(linkList), wrapperID: "page-link-list-content"},
		{name: "入れ子のバックリンク", component: components.BacklinkListResponse(nestedBacklinks), wrapperID: "page-backlink-list-9-content"},
		{name: "ページのバックリンク", component: components.PageBacklinkListResponse(pageBacklinks), wrapperID: "page-backlink-list-content"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			html := renderDraftPageRefreshComponent(t, tt.component)
			if !strings.Contains(html, `id="`+tt.wrapperID+`"`) {
				t.Errorf("ページ単位の応答に一覧全体のラッパー%qが描画されていない", tt.wrapperID)
			}
			if !strings.Contains(html, `hx-target="#`+tt.wrapperID+`"`) {
				t.Errorf("続きのリンクが一覧全体のラッパー%qを置き換えていない", tt.wrapperID)
			}
			if !strings.Contains(html, "context=edit_paginated") {
				t.Error("ページ単位の応答でエディタの文脈が失われている")
			}
			if strings.Contains(html, "読み込めるのはここまで") {
				t.Error("1ページずつのエディタの応答が累積取得の上限を再び適用している")
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
		{name: "リンクの一覧", component: components.LinkListResponse(linkList)},
		{name: "入れ子のバックリンクの一覧", component: components.BacklinkListResponse(backlinkList)},
		{name: "ページのバックリンクの一覧", component: components.PageBacklinkListResponse(backlinkList)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			html := renderDraftPageRefreshComponent(t, tt.component)
			if strings.Contains(html, "hx-swap-oob") {
				t.Error("公開ページのフラグメントがエディタの共有の状態をスワップしている")
			}
			if strings.Contains(html, "page-draft-refresh-trigger") {
				t.Error("公開ページのフラグメントにエディタの下書きの更新トリガーが描画されている")
			}
		})
	}
}

func renderDraftPageRefreshComponent(t *testing.T, component templ.Component) string {
	t.Helper()

	var buf bytes.Buffer
	if err := component.Render(context.Background(), &buf); err != nil {
		t.Fatalf("コンポーネントの描画に失敗: %v", err)
	}

	return buf.String()
}
