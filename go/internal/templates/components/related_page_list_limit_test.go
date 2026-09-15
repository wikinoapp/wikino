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
		t.Fatalf("コンポーネントの描画に失敗: %v", err)
	}

	return buf.String()
}

// TestRelatedPageLists_LoadMoreCappedNoticeは、累積上限で止まった一覧が、下書き由来の
// 1ページ単位編集画面にある最初の省略ページへ案内することを固定する。安定したフォーカスIDにより、
// 直前の「もっと見る」リンクを案内へ差し替えたときもhtmxのフォーカス先を維持する。
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
			name:        "リンクの一覧",
			component:   components.LinkList(linkList),
			wantURL:     "/s/my-space/pages/1/edit?context=edit_paginated&amp;links_page=11#page-link-list-content",
			wantFocusID: "page-link-list-load-more",
		},
		{
			name:        "入れ子のバックリンク",
			component:   components.BacklinkList(backlinkList),
			wantURL:     "/s/my-space/pages/1/edit?context=edit_paginated&amp;linked_backlinks_page=11&amp;linked_page_number=12&amp;links_page=1#page-link-list-item-12",
			wantFocusID: "page-backlink-list-12-load-more",
		},
		{
			name:        "ページのバックリンク",
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
				t.Error("上限で打ち切った一覧にどこで止まるかの説明が無い")
			}
			if strings.Contains(html, "編集画面で読み込めるのはここまでです。") {
				t.Error("日本語の打ち切りの通知に、ほかのメッセージが使わない句点が残っている")
			}
			if !strings.Contains(html, "1ページずつ続きを見る") {
				t.Error("上限で打ち切った一覧に1ページずつの続きの説明が無い")
			}
			if !strings.Contains(html, `href="`+tt.wantURL+`"`) {
				t.Errorf("上限で打ち切った一覧に下書きを元にした続きへのリンクが無い: 期待値 = %q", tt.wantURL)
			}
			if !strings.Contains(html, `id="`+tt.wantFocusID+`"`) || !strings.Contains(html, `tabindex="-1"`) {
				t.Errorf("上限で打ち切った一覧にフォーカス可能な移動先が残っていない: 期待値 = id %q", tt.wantFocusID)
			}
			if strings.Contains(html, "/link_list?") || strings.Contains(html, "/backlinks?") {
				t.Error("上限で打ち切った一覧が累積取得のフラグメントのエンドポイントを提示し続けている")
			}
		})
	}
}

// TestRelatedPageLists_UncappedListingHasNoNoticeは、本当に最後まで出た一覧には案内が付かない
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
		t.Error("最後まで表示した一覧が別の場所に続きがあると示している")
	}
	if strings.Contains(html, `role="status"`) {
		t.Error("初めから全件表示の一覧が動的な更新を通知している")
	}
}

// TestRelatedPageLists_EndNoticeは、一覧を最後まで辿った読み手に出る完了状態を固定する。
// 日本語では句点を付けず、装飾のconfettiアイコンは自前で黒く塗らずcurrentColorで抑えた文字色に
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
			name:        "リンクの一覧",
			component:   components.LinkListResponse(linkList),
			wantFocusID: "page-link-list-load-more",
		},
		{
			name:        "入れ子のバックリンク",
			component:   components.BacklinkListResponse(backlinkList),
			wantFocusID: "page-backlink-list-12-load-more",
		},
		{
			name:        "ページのバックリンク",
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
				t.Error("最後まで辿った一覧に終端の表示が無い")
			}
			if strings.Contains(html, "すべて読み込みました。") {
				t.Error("日本語の通知に、ほかのメッセージが使わない句点が残っている")
			}
			if !strings.Contains(html, `id="`+tt.wantFocusID+`"`) || !strings.Contains(html, `tabindex="-1"`) {
				t.Errorf("通知にフォーカス可能な移動先が残っていない: 期待値 = id %q", tt.wantFocusID)
			}
			if !strings.Contains(html, `role="status"`) {
				t.Error("通知が読み込みの完了を知らせていない")
			}

			icon := endNoticeIcon(t, html)
			if !strings.Contains(icon, `aria-hidden="true"`) {
				t.Error("紙吹雪のアイコンが支援技術から隠されておらず、role=\"status\"がテキスト以外も読み上げてしまう")
			}
			if !strings.Contains(icon, `fill="currentColor"`) {
				t.Errorf("紙吹雪のアイコンが控えめな文字色を継承していない: %s", icon)
			}

			// パス断片はphosphorIconsを二度引かずconfettiのリテラルから取る。マップのfillが
			// 黒へ戻ったときに比較の両辺が揃ってしまうのを避けるためである。これが無いと、名前を誤っても
			// 上のアサーションはすべて通る。iconSVGがinfo-regularへフォールバックし、そのアイコンも
			// 装飾かつcurrentColorであるためである。
			if !strings.Contains(icon, "M111.49,52.63") {
				t.Error("通知に別のアイコンが付いており、未知の名前が暗黙にinfo-regularに置き換わっている")
			}

			notice := strings.Index(html, `role="status"`)
			if strings.Index(html[notice:], "<svg") > strings.Index(html[notice:], "すべて読み込みました") {
				t.Error("紙吹雪のアイコンが装飾するテキストの前にない")
			}
		})
	}
}

// TestRelatedPageLists_EnglishNoticesKeepFullStopsは、日本語の句点を落とした判断が英語へ
// 及んでいないことを固定する。英語の平叙文はピリオドで終えるためである。一括置換は2つの文言へ
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
		t.Error("英語の完了の通知で平叙文の末尾のピリオドが失われている")
	}

	cappedHTML := renderRelatedPageListComponent(t, ctx, components.LinkList(cappedLinkList))
	if !strings.Contains(cappedHTML, "This is as far as the editor loads this list.") {
		t.Error("英語の打ち切りの通知で平叙文の末尾のピリオドが失われている")
	}
}

// endNoticeIconは完了の案内を開くsvg要素を返す。カード側のアイコンも載ったページ全体ではなく、
// この案内のアイコンだけをテストが検査できるようにするためである。
func endNoticeIcon(t *testing.T, html string) string {
	t.Helper()

	notice := strings.Index(html, `role="status"`)
	if notice < 0 {
		t.Fatal("通知が無いため、アイコンも無い")
	}

	icon := svgSegment(html[notice:])
	if icon == "" {
		t.Fatal("通知にアイコンが無い")
	}

	return icon
}
