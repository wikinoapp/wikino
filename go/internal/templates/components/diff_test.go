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

// linesPerBlockはrenderDiffViewが1ブロックに入れる行数である。テンプレートでは3つの行
// 種別がそれぞれ別のクラス属性を書いているため、どの種別も1行ずつ入れている。
const linesPerBlock = 3

// diffPageTitleは描画する差分が属するページの名前である。
const diffPageTitle = "差分のページ"

// renderDiffViewはpageTitleのページの差分をblocks個のブロックでjaロケールで描画し、
// HTMLを返す。
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

// TestDiffView_KeepsEveryBlockReachableWithoutAScrollerは、各ブロックを包む箱が、角丸で
// 切り抜きつつスクロールはしないこと (したがって自前のフォーカス位置を必要としないこと) を確認する。
//
// class属性は部分文字列ではなく値を丸ごと照合する。これにより、ここで確認するクラスに足す形で
// overflow-x-autoが戻ってきても捕まえられる。tabindexは出力全体から探す。動かないフォーカス位置は、
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

// TestDiffView_TellsEachBlockApartByItsCaptionは、ブロックのテーブルのキャプションを確認する。
// 差分は見出しセルを持たないため、表を名指しするのはキャプションだけであり、1つの差分の中で
// ブロックを区別できることに加えて、属するページを名指しできる必要がある。1つの編集提案が複数
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

// TestDiffView_NamesAnUntitledPageInItsCaptionは、まだタイトルが無いページのキャプションを
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
