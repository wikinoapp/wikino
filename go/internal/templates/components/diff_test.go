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
		data.Diff.Blocks = append(data.Diff.Blocks, viewmodel.DiffBlock{
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

// renderBodyDiffは新旧の本文の差分をjaロケールで描画し、HTMLを返す。
func renderBodyDiff(t *testing.T, diff viewmodel.BodyDiff) string {
	t.Helper()

	ctx := i18n.SetLocale(context.Background(), "ja")

	var buf bytes.Buffer
	if err := components.DiffView(components.DiffViewData{PageTitle: diffPageTitle, Diff: diff}).Render(ctx, &buf); err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}
	return buf.String()
}

// TestDiffView_EmbedsSourcesForHighlightingは、新旧の本文を持つ差分が、ブラウザで記法の色を当てる
// ための本文と目印を、差分のまとまりの要素の中に埋めることを確認する。編集提案の画面には複数の
// ページの差分が並ぶため、idで探さない形にする。本文はscript要素の外へ出ないようエスケープする。
func TestDiffView_EmbedsSourcesForHighlighting(t *testing.T) {
	t.Parallel()

	html := renderBodyDiff(t, viewmodel.NewBodyDiff("<b>旧</b>\n", "</script>新\n"))

	const wantPrefix = `<div class="flex flex-col gap-4" data-markdown-diff><script type="application/json">`
	if !strings.HasPrefix(html, wantPrefix) {
		t.Errorf("差分のまとまりの直下に本文が埋められていない: %s", html)
	}
	if strings.Contains(html, " id=") {
		t.Error("同じ画面に並ぶ差分で重なるidが付いている")
	}
	if !strings.Contains(html, `{"old":"\u003cb\u003e旧\u003c/b\u003e\n","new":"\u003c/script\u003e新\n"}`) {
		t.Errorf("新旧の本文がエスケープして埋められていない: %s", html)
	}
	if strings.Count(html, "</script>") != 1 {
		t.Error("本文がscript要素の外へ出ている")
	}
}

// TestDiffView_OmitsSourcesWithoutThemは、新旧の本文を持たない差分 (本文が大きすぎるときなど) では、
// 本文も目印も埋めず、色の無い差分だけを描画することを確認する。
func TestDiffView_OmitsSourcesWithoutThem(t *testing.T) {
	t.Parallel()

	html := renderDiffView(t, diffPageTitle, 1)

	if strings.Contains(html, "data-markdown-diff ") || strings.Contains(html, "data-markdown-diff>") {
		t.Error("本文を持たない差分に色付けの目印が付いている")
	}
	if strings.Contains(html, "<script") {
		t.Error("本文を持たない差分にscript要素がある")
	}
}

// TestDiffView_TellsEachCellItsSourceLineは、本文のセルが、旧・新のどちらの本文の何行目かを持つことを
// 確認する。削除の行は旧の本文にだけあり、追加と変更なしの行は新の本文の行番号で引く。
func TestDiffView_TellsEachCellItsSourceLine(t *testing.T) {
	t.Parallel()

	html := renderBodyDiff(t, viewmodel.NewBodyDiff("同じ\n消す\n残す\n", "同じ\n足す\n足す2\n残す\n"))

	for _, want := range []string{
		`data-markdown-diff-source="new" data-markdown-diff-line="1">同じ</td>`,
		`data-markdown-diff-source="old" data-markdown-diff-line="2">消す</td>`,
		`data-markdown-diff-source="new" data-markdown-diff-line="2">足す</td>`,
		`data-markdown-diff-source="new" data-markdown-diff-line="3">足す2</td>`,
		`data-markdown-diff-source="new" data-markdown-diff-line="4">残す</td>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("本文のセルが見つからない: %s", want)
		}
	}
}

// TestDiffView_KeepsChangedLineTextInTheNormalColorは、追加・削除の行の本文の文字に赤・緑の色を付けない
// ことを確認する。記法の色と混ざらないよう、追加・削除は背景色・行番号の色・記号で示す。
func TestDiffView_KeepsChangedLineTextInTheNormalColor(t *testing.T) {
	t.Parallel()

	html := renderDiffView(t, diffPageTitle, 1)

	for _, color := range []string{"text-red-700", "text-green-700", "text-red-300", "text-green-300"} {
		if strings.Contains(html, color) {
			t.Errorf("本文の文字に色 %s が付いている", color)
		}
	}
	for _, kept := range []string{"bg-red-50", "bg-green-50", "text-red-400", "text-green-400", "text-red-500", "text-green-500"} {
		if !strings.Contains(html, kept) {
			t.Errorf("追加・削除を示す色 %s が無くなっている", kept)
		}
	}
}
