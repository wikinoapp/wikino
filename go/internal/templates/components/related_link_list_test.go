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

// showLinkListは、2つのリンク先ページを持ち、そのうち先頭のページにだけバックリンクがある
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

// TestLinkList_CardsOnlyは、リンクセクションがリンク先ページのカード以外を持たないことを固定
// する。リンク先ページのバックリンクは関連リンクのセクションに属する。
func TestLinkList_CardsOnly(t *testing.T) {
	t.Parallel()

	html := renderJa(t, func(ctx context.Context, buf *bytes.Buffer) error {
		return components.LinkList(showLinkList()).Render(ctx, buf)
	})

	if !strings.Contains(html, `id="page-link-list-content"`) {
		t.Error("リンク一覧のページ全体のフォールバックのアンカーが無い")
	}
	if !strings.Contains(html, `id="page-link-list-pagination"`) {
		t.Error("リンク一覧のページネーションのコンテナが無い")
	}
	for _, want := range []string{"リンク先ページ", "バックリンクの無いリンク先ページ"} {
		if !strings.Contains(html, want) {
			t.Errorf("リンクセクションにカード%qが含まれていない", want)
		}
	}

	// ネスト構成のカードごとのラッパーと、そこに載っていたバックリンクのカードは出ない。
	if strings.Contains(html, `id="page-link-list-item-2"`) {
		t.Error("リンクセクションに入れ子のバックリンクのアンカーが含まれている")
	}
	if strings.Contains(html, "関連リンクのページ") {
		t.Error("リンクセクションにリンク先ページのバックリンクが描画されている")
	}
}

// TestRelatedLinkList_GroupsPerLinkedPageは、関連リンクのセクションがバックリンクをリンク先
// ページごとに束ね、どこからもリンクされていないリンク先ページを落とすことを固定する。
func TestRelatedLinkList_GroupsPerLinkedPage(t *testing.T) {
	t.Parallel()

	html := renderJa(t, func(ctx context.Context, buf *bytes.Buffer) error {
		return components.RelatedLinkList(showLinkList()).Render(ctx, buf)
	})

	if !strings.Contains(html, `id="page-related-link-list"`) {
		t.Error("関連リンクのコンテナが無い")
	}
	if !strings.Contains(html, `id="page-link-list-item-2"`) {
		t.Error("入れ子のバックリンク一覧のページ全体のフォールバックのアンカーがグループに無い")
	}
	if !strings.Contains(html, "via") {
		t.Error("グループの小見出しに経由したリンク先ページの名前が無い")
	}
	if !strings.Contains(html, "リンク先ページ") {
		t.Error("グループの小見出しにリンク先ページのタイトルが含まれていない")
	}

	// 小見出しの名前は、グループがぶら下がっているリンク先ページを開く。グループ内のどのカードも
	// そのページへは辿り着かないためである。
	if !strings.Contains(html, `href="/s/my-space/pages/2"`) {
		t.Error("グループの小見出しが示すリンク先ページにリンクしていない")
	}
	if !strings.Contains(html, "関連リンクのページ") {
		t.Error("グループにバックリンクのカードが含まれていない")
	}

	// バックリンクの無いリンク先ページは、空のグループではなくグループごと出さない。
	if strings.Contains(html, `id="page-link-list-item-3"`) {
		t.Error("バックリンクの無いリンク先ページにグループが作られている")
	}
	if strings.Contains(html, "バックリンクの無いリンク先ページ") {
		t.Error("バックリンクの無いリンク先ページが関連リンクセクションに表示されている")
	}
}

// TestRelatedLinkList_EmptyKeepsContainerは、1ページ目が空でもコンテナが残ることを固定する。
// 後続のリンク一覧ページがOOBでグループを追記するには、その対象が存在している必要がある。
func TestRelatedLinkList_EmptyKeepsContainer(t *testing.T) {
	t.Parallel()

	data := showLinkList()
	data.Items = data.Items[1:]

	html := renderJa(t, func(ctx context.Context, buf *bytes.Buffer) error {
		return components.RelatedLinkList(data).Render(ctx, buf)
	})

	if !strings.Contains(html, `id="page-related-link-list"`) {
		t.Error("グループが無いときに関連リンクのコンテナが描画されていない")
	}
	if strings.Contains(html, `id="page-link-list-item-`) {
		t.Error("バックリンクを持つリンク先ページが無いのにグループが描画されている")
	}
}

// TestRelatedLinkList_UntitledLinkedPageは、タイトルの無いリンク先ページがグループの小見出しで
// カードと同じ呼び方をされることを固定する。
func TestRelatedLinkList_UntitledLinkedPage(t *testing.T) {
	t.Parallel()

	data := showLinkList()
	data.Items[0].CardLinkPage.Title = ""

	html := renderJa(t, func(ctx context.Context, buf *bytes.Buffer) error {
		return components.RelatedLinkList(data).Render(ctx, buf)
	})

	if !strings.Contains(html, "無題") {
		t.Error("無題のリンク先ページがカードと同じプレースホルダーになっていない")
	}
}

// TestLinkListResponse_ShowAppendsRelatedLinkGroupsは、リンク一覧の1ページがページ表示画面の
// 両方のセクションへ届くことを固定する。カードはスワップされる本体で、グループはOOBで届く。
func TestLinkListResponse_ShowAppendsRelatedLinkGroups(t *testing.T) {
	t.Parallel()

	html := renderJa(t, func(ctx context.Context, buf *bytes.Buffer) error {
		return components.LinkListResponse(showLinkList()).Render(ctx, buf)
	})

	if !strings.Contains(html, `id="page-link-list-pagination"`) {
		t.Error("スワップした本文にリンク一覧のページネーションのコンテナが無い")
	}
	if !strings.Contains(html, `id="page-related-link-list" hx-swap-oob="beforeend"`) {
		t.Error("関連リンクのグループがアウトオブバンドで追加されていない")
	}
	if !strings.Contains(html, `id="page-link-list-item-2"`) {
		t.Error("追加されたグループが無い")
	}

	// 編集画面で共有する状態要素は公開画面には存在しないため、そこへスワップしようとしない。
	if strings.Contains(html, `id="page-link-list-state"`) {
		t.Error("公開画面に進めるべき共有の関連ページの状態がある")
	}
}

// TestLinkListResponse_ShowWithoutRelatedLinksは、リンク先ページにバックリンクが無いリンク一覧の
// ページがOOBの追記を一切送らないことを固定する。
func TestLinkListResponse_ShowWithoutRelatedLinks(t *testing.T) {
	t.Parallel()

	data := showLinkList()
	data.Items = data.Items[1:]

	html := renderJa(t, func(ctx context.Context, buf *bytes.Buffer) error {
		return components.LinkListResponse(data).Render(ctx, buf)
	})

	if strings.Contains(html, "hx-swap-oob") {
		t.Error("グループが空なのにアウトオブバンドのスワップが生成されている")
	}
	if !strings.Contains(html, "バックリンクの無いリンク先ページ") {
		t.Error("スワップした本文に新しいカードが無い")
	}
}

// editLinkListは同じリンク一覧を編集画面の2つのページング文脈のいずれかで組み立てる。ページ
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

// TestLinkListResponse_EditAppendsRelatedLinkGroupsは、累積表示の編集画面が2つのリンクの
// セクションをページ表示画面と同じように進め、加えて共有する関連ページ状態のうち自分の分も進める
// ことを固定する。
func TestLinkListResponse_EditAppendsRelatedLinkGroups(t *testing.T) {
	t.Parallel()

	html := renderJa(t, func(ctx context.Context, buf *bytes.Buffer) error {
		return components.LinkListResponse(editLinkList(viewmodel.PageLinkContextEdit)).Render(ctx, buf)
	})

	if !strings.Contains(html, `id="page-link-list-pagination"`) {
		t.Error("スワップした本文にリンク一覧のページネーションのコンテナが無い")
	}
	if strings.Contains(html, `id="page-link-list-content"`) {
		t.Error("累積取得の応答が一覧を置き換えずに追加している")
	}
	if !strings.Contains(html, `id="page-related-link-list" hx-swap-oob="beforeend"`) {
		t.Error("関連リンクのグループがアウトオブバンドで追加されていない")
	}
	if !strings.Contains(html, `id="page-link-list-item-2"`) {
		t.Error("追加されたグループが無い")
	}
	if !strings.Contains(html, `id="page-link-list-state"`) {
		t.Error("エディタの共有のリンク一覧の状態が進んでいない")
	}
}

// TestLinkListResponse_PaginatedEditReplacesRelatedLinkGroupsは、1ページ単位の編集画面が
// 両方のセクションを差し替えることを固定する。リンク一覧を1ページずつ描画するため、離れるページの
// グループも、そのページのカードと一緒に落ちる必要がある。
func TestLinkListResponse_PaginatedEditReplacesRelatedLinkGroups(t *testing.T) {
	t.Parallel()

	html := renderJa(t, func(ctx context.Context, buf *bytes.Buffer) error {
		return components.LinkListResponse(editLinkList(viewmodel.PageLinkContextEditPaginated)).Render(ctx, buf)
	})

	if !strings.Contains(html, `id="page-link-list-content"`) {
		t.Error("1ページずつのエディタがリンクセクション全体を置き換えていない")
	}
	if !strings.Contains(html, `id="page-related-link-list" hx-swap-oob="innerHTML"`) {
		t.Error("1ページずつのエディタが関連リンクセクションを置き換えずに追加している")
	}
	if !strings.Contains(html, `id="page-link-list-item-2"`) {
		t.Error("置き換えるグループが無い")
	}
	if !strings.Contains(html, `id="page-link-list-state"`) {
		t.Error("1ページずつのモードでエディタの共有のリンク一覧の状態が進んでいない")
	}
}

// TestLinkListResponse_PaginatedEditClearsRelatedLinkGroupsは、新しいページがグループを1つも
// 生まないときも1ページ単位の編集画面が差し替えのスワップを送ることを固定する。送らないと、
// セクションには閲覧者が離れたページのグループが残り続けてしまう。
func TestLinkListResponse_PaginatedEditClearsRelatedLinkGroups(t *testing.T) {
	t.Parallel()

	data := editLinkList(viewmodel.PageLinkContextEditPaginated)
	data.Items = data.Items[1:]

	html := renderJa(t, func(ctx context.Context, buf *bytes.Buffer) error {
		return components.LinkListResponse(data).Render(ctx, buf)
	})

	if !strings.Contains(html, `id="page-related-link-list" hx-swap-oob="innerHTML"`) {
		t.Error("グループの無いページで関連リンクセクションが空にされていない")
	}
	if strings.Contains(html, `id="page-link-list-item-`) {
		t.Error("空にするスワップにグループが含まれている")
	}
}
