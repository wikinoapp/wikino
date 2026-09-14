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

// linesPerBlock is how many lines renderDiffView puts in each block. Every one of the three line
// types is there, because each of them writes its own class attribute in the template.
//
// [Ja] linesPerBlock は renderDiffView が 1 ブロックに入れる行数である。テンプレートでは 3 つの行
// 種別がそれぞれ別のクラス属性を書いているため、どの種別も 1 行ずつ入れている。
const linesPerBlock = 3

// diffPageTitle is the page name the rendered diffs belong to.
//
// [Ja] diffPageTitle は描画する差分が属するページの名前である。
const diffPageTitle = "差分のページ"

// renderDiffView renders a diff of pageTitle with the requested number of blocks using the ja
// locale and returns the HTML.
//
// [Ja] renderDiffView は pageTitle のページの差分を blocks 個のブロックで ja ロケールで描画し、
// HTML を返す。
func renderDiffView(t *testing.T, pageTitle string, blocks int) string {
	t.Helper()

	data := components.DiffViewData{PageTitle: pageTitle}
	for i := range blocks {
		data.Blocks = append(data.Blocks, viewmodel.DiffBlock{
			Lines: []viewmodel.DiffLine{
				{Type: viewmodel.DiffLineEqual, OldNumber: i + 1, NewNumber: i + 1, Content: "変更のない行"},
				{Type: viewmodel.DiffLineDelete, OldNumber: i + 2, Content: "消した行"},
				{Type: viewmodel.DiffLineInsert, NewNumber: i + 2, Content: "足した行"},
			},
		})
	}

	ctx := i18n.SetLocale(context.Background(), "ja")

	var buf bytes.Buffer
	if err := components.DiffView(data).Render(ctx, &buf); err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}
	return buf.String()
}

// TestDiffView_KeepsEveryBlockReachableWithoutAScroller covers the box around each block: it
// clips at the rounded corners but never scrolls, so it does not need a focus stop of its own.
//
// The class attribute is matched whole rather than by substring, so that overflow-x-auto coming
// back is caught even if it is added alongside the classes checked here. tabindex is looked for
// across the whole output: a focus stop that scrolls nothing would be dead weight on every block
// of every diff.
//
// [Ja] TestDiffView_KeepsEveryBlockReachableWithoutAScroller は、各ブロックを包む箱が、角丸で
// 切り抜きつつスクロールはしないこと (したがって自前のフォーカス位置を必要としないこと) を確認する。
//
// class 属性は部分文字列ではなく値を丸ごと照合する。これにより、ここで確認するクラスに足す形で
// overflow-x-auto が戻ってきても捕まえられる。tabindex は出力全体から探す。動かないフォーカス位置は、
// どの差分のどのブロックにも付いて回る無駄になるためである。
func TestDiffView_KeepsEveryBlockReachableWithoutAScroller(t *testing.T) {
	t.Parallel()

	const blocks = 2

	html := renderDiffView(t, diffPageTitle, blocks)

	if strings.Count(html, `class="overflow-clip rounded-md border border-border"`) != blocks {
		t.Error("差分のブロックを包む箱が、スクロールコンテナを作らない切り抜きになっていない")
	}

	if strings.Contains(html, "tabindex") {
		t.Error("スクロールしない箱にフォーカス位置が付いている")
	}

	if strings.Count(html, "whitespace-pre-wrap break-all") != blocks*linesPerBlock {
		t.Error("スクロールを使わずに差分本文を折り返せる指定が揃っていない")
	}
}

// TestDiffView_TellsEachBlockApartByItsCaption covers the caption of the block tables. The diff
// carries no header cells, so the caption is the only thing naming the table, and it has to tell
// the blocks of one diff apart as well as name the page they belong to, because one suggestion
// can put the diffs of several pages on the same screen.
//
// [Ja] TestDiffView_TellsEachBlockApartByItsCaption は、ブロックのテーブルのキャプションを確認する。
// 差分は見出しセルを持たないため、表を名指しするのはキャプションだけであり、1 つの差分の中で
// ブロックを区別できることに加えて、属するページを名指しできる必要がある。1 つの編集提案が複数
// ページの差分を同じ画面に並べるためである。
func TestDiffView_TellsEachBlockApartByItsCaption(t *testing.T) {
	t.Parallel()

	html := renderDiffView(t, diffPageTitle, 3)

	for _, want := range []string{
		`<caption class="sr-only">差分のページ の本文の差分 (3 か所中 1 番目)</caption>`,
		`<caption class="sr-only">差分のページ の本文の差分 (3 か所中 2 番目)</caption>`,
		`<caption class="sr-only">差分のページ の本文の差分 (3 か所中 3 番目)</caption>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("キャプションが見つからない: %s", want)
		}
	}

	if strings.Contains(html, "<th") {
		t.Error("差分のテーブルに見出しセルが付いている")
	}
}

// TestDiffView_NamesAnUntitledPageInItsCaption covers the caption of a page that has no title yet:
// a page keeps none until it is published, and the revision diff of the editor shows exactly such
// a draft. Without the fallback the caption would open with a bare separator.
//
// [Ja] TestDiffView_NamesAnUntitledPageInItsCaption は、まだタイトルが無いページのキャプションを
// 確認する。ページは公開されるまでタイトルを持たないことがあり、エディタのリビジョン差分は
// まさにそうした下書きを表示する。フォールバックが無いと、キャプションが区切り記号から始まる。
func TestDiffView_NamesAnUntitledPageInItsCaption(t *testing.T) {
	t.Parallel()

	html := renderDiffView(t, "", 1)

	const want = `<caption class="sr-only">無題 の本文の差分 (1 か所中 1 番目)</caption>`
	if !strings.Contains(html, want) {
		t.Errorf("タイトルの無いページのキャプションが見つからない: %s", want)
	}
}
