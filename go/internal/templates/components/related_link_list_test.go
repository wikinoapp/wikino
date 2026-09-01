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

// showLinkList builds the link list of a page detail screen holding two linked pages, only the
// first of which anything links back to.
//
// [Ja] showLinkList は、2 つのリンク先ページを持ち、そのうち先頭のページにだけバックリンクがある
// ページ表示画面のリンク一覧を組み立てる。
func showLinkList() viewmodel.LinkList {
	return viewmodel.LinkList{
		Items: []viewmodel.LinkListItem{
			{
				CardLinkPage: viewmodel.CardLinkPage{Title: "リンク先ページ", Number: 2},
				BacklinkList: viewmodel.BacklinkList{
					Items: []viewmodel.BacklinkListItem{
						{CardLinkPage: viewmodel.CardLinkPage{Title: "関連リンクのページ", Number: 5}},
					},
					Pagination:       viewmodel.NewPagination(1, 1, int(viewmodel.BacklinkLimit)),
					SpaceIdentifier:  viewmodel.SpaceIdentifier("my-space"),
					PageNumber:       1,
					LinkedPageNumber: 2,
					LinkedPageTitle:  "リンク先ページ",
					State:            viewmodel.PageLinkState{Context: viewmodel.PageLinkContextShow},
				},
			},
			{
				CardLinkPage: viewmodel.CardLinkPage{Title: "バックリンクの無いリンク先ページ", Number: 3},
			},
		},
		Pagination:      viewmodel.NewPagination(1, 1, int(viewmodel.LinkLimit)),
		SpaceIdentifier: viewmodel.SpaceIdentifier("my-space"),
		PageNumber:      1,
		State:           viewmodel.PageLinkState{Context: viewmodel.PageLinkContextShow},
	}
}

func renderJa(t *testing.T, render func(context.Context, *bytes.Buffer) error) string {
	t.Helper()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)

	var buf bytes.Buffer
	if err := render(ctx, &buf); err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}

	return buf.String()
}

// TestLinkList_CardsOnly fixes that the links section holds nothing but the linked pages' cards:
// the backlinks of those pages belong to the related-links section.
//
// [Ja] TestLinkList_CardsOnly は、リンクセクションがリンク先ページのカード以外を持たないことを固定
// する。リンク先ページのバックリンクは関連リンクのセクションに属する。
func TestLinkList_CardsOnly(t *testing.T) {
	t.Parallel()

	html := renderJa(t, func(ctx context.Context, buf *bytes.Buffer) error {
		return components.LinkList(showLinkList()).Render(ctx, buf)
	})

	if !strings.Contains(html, `id="page-link-list-content"`) {
		t.Error("full-page fallback anchor for the link list is missing")
	}
	if !strings.Contains(html, `id="page-link-list-pagination"`) {
		t.Error("pagination container for the link list is missing")
	}
	for _, want := range []string{"リンク先ページ", "バックリンクの無いリンク先ページ"} {
		if !strings.Contains(html, want) {
			t.Errorf("links section does not contain the card %q", want)
		}
	}

	// The nested arrangement's per-card wrapper and the backlink cards it held are gone.
	//
	// [Ja] ネスト構成のカードごとのラッパーと、そこに載っていたバックリンクのカードは出ない。
	if strings.Contains(html, `id="page-link-list-item-2"`) {
		t.Error("the links section should not carry the nested backlink anchor")
	}
	if strings.Contains(html, "関連リンクのページ") {
		t.Error("the links section should not render the backlinks of a linked page")
	}
}

// TestRelatedLinkList_GroupsPerLinkedPage fixes that the related-links section bundles the backlinks
// per linked page and drops a linked page nothing links back to.
//
// [Ja] TestRelatedLinkList_GroupsPerLinkedPage は、関連リンクのセクションがバックリンクをリンク先
// ページごとに束ね、どこからもリンクされていないリンク先ページを落とすことを固定する。
func TestRelatedLinkList_GroupsPerLinkedPage(t *testing.T) {
	t.Parallel()

	html := renderJa(t, func(ctx context.Context, buf *bytes.Buffer) error {
		return components.RelatedLinkList(showLinkList()).Render(ctx, buf)
	})

	if !strings.Contains(html, `id="page-related-link-list"`) {
		t.Error("the related-links container is missing")
	}
	if !strings.Contains(html, `id="page-link-list-item-2"`) {
		t.Error("full-page fallback anchor of the nested backlink list is missing from its group")
	}
	if !strings.Contains(html, "via") {
		t.Error("the group subheading should name the linked page it is reached through")
	}
	if !strings.Contains(html, "リンク先ページ") {
		t.Error("the group subheading does not contain the linked page title")
	}

	// The name in the subheading opens the linked page the group hangs off, which no card in the
	// group leads to.
	//
	// [Ja] 小見出しの名前は、グループがぶら下がっているリンク先ページを開く。グループ内のどのカードも
	// そのページへは辿り着かないためである。
	if !strings.Contains(html, `href="/s/my-space/pages/2"`) {
		t.Error("the group subheading should link to the linked page it names")
	}
	if !strings.Contains(html, "関連リンクのページ") {
		t.Error("the group does not contain the backlink card")
	}

	// A linked page without backlinks contributes no group at all, not an empty one.
	//
	// [Ja] バックリンクの無いリンク先ページは、空のグループではなくグループごと出さない。
	if strings.Contains(html, `id="page-link-list-item-3"`) {
		t.Error("a linked page without backlinks should not get a group")
	}
	if strings.Contains(html, "バックリンクの無いリンク先ページ") {
		t.Error("a linked page without backlinks should not be named in the related-links section")
	}
}

// TestRelatedLinkList_EmptyKeepsContainer fixes that the container survives an empty first page. A
// later link-list page appends its groups into it out of band, which needs the target to exist.
//
// [Ja] TestRelatedLinkList_EmptyKeepsContainer は、1 ページ目が空でもコンテナが残ることを固定する。
// 後続のリンク一覧ページが OOB でグループを追記するには、その対象が存在している必要がある。
func TestRelatedLinkList_EmptyKeepsContainer(t *testing.T) {
	t.Parallel()

	data := showLinkList()
	data.Items = data.Items[1:]

	html := renderJa(t, func(ctx context.Context, buf *bytes.Buffer) error {
		return components.RelatedLinkList(data).Render(ctx, buf)
	})

	if !strings.Contains(html, `id="page-related-link-list"`) {
		t.Error("the related-links container should be rendered even while it holds no group")
	}
	if strings.Contains(html, `id="page-link-list-item-`) {
		t.Error("no group should be rendered when no linked page has backlinks")
	}
}

// TestRelatedLinkList_UntitledLinkedPage fixes that a linked page without a title is named in the
// group subheading the same way the cards name it.
//
// [Ja] TestRelatedLinkList_UntitledLinkedPage は、タイトルの無いリンク先ページがグループの小見出しで
// カードと同じ呼び方をされることを固定する。
func TestRelatedLinkList_UntitledLinkedPage(t *testing.T) {
	t.Parallel()

	data := showLinkList()
	data.Items[0].CardLinkPage.Title = ""

	html := renderJa(t, func(ctx context.Context, buf *bytes.Buffer) error {
		return components.RelatedLinkList(data).Render(ctx, buf)
	})

	if !strings.Contains(html, "無題") {
		t.Error("an untitled linked page should fall back to the placeholder the cards use")
	}
}

// TestLinkListResponse_ShowAppendsRelatedLinkGroups fixes that one link-list page reaches both
// sections of the page detail screen: the cards through the swapped body, the groups out of band.
//
// [Ja] TestLinkListResponse_ShowAppendsRelatedLinkGroups は、リンク一覧の 1 ページがページ表示画面の
// 両方のセクションへ届くことを固定する。カードはスワップされる本体で、グループは OOB で届く。
func TestLinkListResponse_ShowAppendsRelatedLinkGroups(t *testing.T) {
	t.Parallel()

	html := renderJa(t, func(ctx context.Context, buf *bytes.Buffer) error {
		return components.LinkListResponse(showLinkList()).Render(ctx, buf)
	})

	if !strings.Contains(html, `id="page-link-list-pagination"`) {
		t.Error("the swapped body should carry the link list's pagination container")
	}
	if !strings.Contains(html, `id="page-related-link-list" hx-swap-oob="beforeend"`) {
		t.Error("the related-link groups should be appended out of band")
	}
	if !strings.Contains(html, `id="page-link-list-item-2"`) {
		t.Error("the appended group is missing")
	}

	// The editor's shared state element does not exist on the public screen, so nothing tries to
	// swap it there.
	//
	// [Ja] 編集画面で共有する状態要素は公開画面には存在しないため、そこへスワップしようとしない。
	if strings.Contains(html, `id="page-link-list-state"`) {
		t.Error("the public screen has no shared related-page state to advance")
	}
}

// TestLinkListResponse_ShowWithoutRelatedLinks fixes that a link-list page whose linked pages have
// no backlinks sends no out-of-band append at all.
//
// [Ja] TestLinkListResponse_ShowWithoutRelatedLinks は、リンク先ページにバックリンクが無いリンク一覧の
// ページが OOB の追記を一切送らないことを固定する。
func TestLinkListResponse_ShowWithoutRelatedLinks(t *testing.T) {
	t.Parallel()

	data := showLinkList()
	data.Items = data.Items[1:]

	html := renderJa(t, func(ctx context.Context, buf *bytes.Buffer) error {
		return components.LinkListResponse(data).Render(ctx, buf)
	})

	if strings.Contains(html, "hx-swap-oob") {
		t.Error("an empty group set should not produce an out-of-band swap")
	}
	if !strings.Contains(html, "バックリンクの無いリンク先ページ") {
		t.Error("the swapped body should still carry the new card")
	}
}

// editLinkList builds the same link list under one of the editor's two pagination contexts, so the
// arrangement the page detail screen fixed can be checked against how the editor advances it.
//
// [Ja] editLinkList は同じリンク一覧を編集画面の 2 つのページング文脈のいずれかで組み立てる。ページ
// 表示画面が固定した並べ方を、編集画面がそれをどう進めるかに対して確認できるようにする。
func editLinkList(context viewmodel.PageLinkContext) viewmodel.LinkList {
	data := showLinkList()
	state := viewmodel.PageLinkState{Context: context}

	data.State = state
	for i := range data.Items {
		data.Items[i].BacklinkList.State = state
	}

	return data
}

// TestLinkListResponse_EditAppendsRelatedLinkGroups fixes that the cumulative editor advances the
// two link sections the same way the page detail screen does, and additionally advances its own
// share of the shared related-page state.
//
// [Ja] TestLinkListResponse_EditAppendsRelatedLinkGroups は、累積表示の編集画面が 2 つのリンクの
// セクションをページ表示画面と同じように進め、加えて共有する関連ページ状態のうち自分の分も進める
// ことを固定する。
func TestLinkListResponse_EditAppendsRelatedLinkGroups(t *testing.T) {
	t.Parallel()

	html := renderJa(t, func(ctx context.Context, buf *bytes.Buffer) error {
		return components.LinkListResponse(editLinkList(viewmodel.PageLinkContextEdit)).Render(ctx, buf)
	})

	if !strings.Contains(html, `id="page-link-list-pagination"`) {
		t.Error("the swapped body should carry the link list's pagination container")
	}
	if strings.Contains(html, `id="page-link-list-content"`) {
		t.Error("a cumulative reply appends to the listing instead of replacing it")
	}
	if !strings.Contains(html, `id="page-related-link-list" hx-swap-oob="beforeend"`) {
		t.Error("the related-link groups should be appended out of band")
	}
	if !strings.Contains(html, `id="page-link-list-item-2"`) {
		t.Error("the appended group is missing")
	}
	if !strings.Contains(html, `id="page-link-list-state"`) {
		t.Error("the editor's shared link-list state should be advanced")
	}
}

// TestLinkListResponse_PaginatedEditReplacesRelatedLinkGroups fixes that the one-page editor
// replaces both sections. It renders a single link-list page, so the groups of the page it leaves
// have to go with the cards of that page.
//
// [Ja] TestLinkListResponse_PaginatedEditReplacesRelatedLinkGroups は、1 ページ単位の編集画面が
// 両方のセクションを差し替えることを固定する。リンク一覧を 1 ページずつ描画するため、離れるページの
// グループも、そのページのカードと一緒に落ちる必要がある。
func TestLinkListResponse_PaginatedEditReplacesRelatedLinkGroups(t *testing.T) {
	t.Parallel()

	html := renderJa(t, func(ctx context.Context, buf *bytes.Buffer) error {
		return components.LinkListResponse(editLinkList(viewmodel.PageLinkContextEditPaginated)).Render(ctx, buf)
	})

	if !strings.Contains(html, `id="page-link-list-content"`) {
		t.Error("the one-page editor should replace the whole links section")
	}
	if !strings.Contains(html, `id="page-related-link-list" hx-swap-oob="innerHTML"`) {
		t.Error("the one-page editor should replace the related-links section rather than append to it")
	}
	if !strings.Contains(html, `id="page-link-list-item-2"`) {
		t.Error("the replacing group is missing")
	}
	if !strings.Contains(html, `id="page-link-list-state"`) {
		t.Error("the editor's shared link-list state should be advanced in the one-page mode too")
	}
}

// TestLinkListResponse_PaginatedEditClearsRelatedLinkGroups fixes that the one-page editor still
// sends the replacing swap when the new page contributes no group. Without it the section would keep
// showing the groups of the page the reader has left.
//
// [Ja] TestLinkListResponse_PaginatedEditClearsRelatedLinkGroups は、新しいページがグループを 1 つも
// 生まないときも 1 ページ単位の編集画面が差し替えのスワップを送ることを固定する。送らないと、
// セクションには閲覧者が離れたページのグループが残り続けてしまう。
func TestLinkListResponse_PaginatedEditClearsRelatedLinkGroups(t *testing.T) {
	t.Parallel()

	data := editLinkList(viewmodel.PageLinkContextEditPaginated)
	data.Items = data.Items[1:]

	html := renderJa(t, func(ctx context.Context, buf *bytes.Buffer) error {
		return components.LinkListResponse(data).Render(ctx, buf)
	})

	if !strings.Contains(html, `id="page-related-link-list" hx-swap-oob="innerHTML"`) {
		t.Error("a page without groups should still clear the related-links section")
	}
	if strings.Contains(html, `id="page-link-list-item-`) {
		t.Error("the clearing swap should carry no group")
	}
}
