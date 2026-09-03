package seed

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGenerateSandboxPages(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	spaces := buildSeedSpaces(t, tx, "seed-sandbox")
	topics, err := generateTopics(ctx, tx, io.Discard, spaces)
	if err != nil {
		t.Fatalf("トピック生成に失敗: %v", err)
	}

	if err := generateSandboxPages(ctx, tx, io.Discard, spaces, topics); err != nil {
		t.Fatalf("表示崩れ確認用ページの生成に失敗: %v", err)
	}

	specs := sandboxPageSpecs()

	// Every page has to be published and to sit in topics.sandbox: an
	// unpublished page is missing from the listings, and the point of these pages
	// is to be met while browsing rather than to be reached by their URL.
	//
	// [Ja] すべてのページが公開済みで、かつ「サンドボックス」に置かれている必要がある。
	// 未公開のページは一覧に出ず、これらのページの目的は URL で辿り着くことでは
	// なく、閲覧の途中で出会うことにあるため。
	if got := countPagesInTopic(ctx, t, tx, spaces.wiki.id, topics.sandbox.id); got != len(specs) {
		t.Errorf("「サンドボックス」のページが %d 件であることを期待したが %d 件だった", len(specs), got)
	}
	for _, spec := range specs {
		if page := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.sandbox.id, spec.title); !page.published {
			t.Errorf("%q が公開済みであることを期待したが未公開だった", spec.title)
		}
	}

	// The generated bodies are checked through the stored HTML rather than
	// through the Markdown they were built from: a table that the renderer did
	// not read as a table, or a fence that did not close, would leave the page
	// looking nothing like what it was written to show.
	//
	// [Ja] 生成した本文は、組み立て元の Markdown ではなく保存された HTML で確認する。
	// レンダラーがテーブルとして読まなかった表や、閉じられなかったフェンスは、その
	// ページを、見せるために書いたものとはまったく別の見た目にしてしまうため。
	tablePage := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.sandbox.id, wideTablePageTitle)
	tableRow := readPage(ctx, t, tx, spaces.wiki.id, tablePage.id)
	if got := strings.Count(tableRow.bodyHTML, "<th>"); got != wideTableColumns {
		t.Errorf("横に長いテーブルの列が %d 列であることを期待したが %d 列だった", wideTableColumns, got)
	}
	if got := strings.Count(tableRow.bodyHTML, "<tr>"); got != wideTableRows+1 {
		t.Errorf("横に長いテーブルの行が %d 行であることを期待したが %d 行だった", wideTableRows+1, got)
	}

	codePage := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.sandbox.id, longCodeBlockPageTitle)
	codeRow := readPage(ctx, t, tx, spaces.wiki.id, codePage.id)
	if !strings.Contains(codeRow.bodyHTML, "<pre><code") {
		t.Errorf("長大なコードブロックがコードブロックとして描画されていない: %q", codeRow.bodyHTML)
	}
	if got := strings.Count(codeRow.body, "\tfmt.Printf("); got != longCodeBlockPrintLines {
		t.Errorf(
			"長大なコードブロックの繰り返し出力行が %d 件であることを期待したが %d 件だった",
			longCodeBlockPrintLines,
			got,
		)
	}

	// The block is long in one direction only unless it also holds a line that
	// reaches past its right edge, and that line is what a code block scrolling
	// sideways has to be provoked by.
	//
	// [Ja] ブロックが長いだけでは縦方向しか確認できない。右端を越える行があって
	// 初めてコードブロックの横スクロールを確認できる。
	wideLine := strings.Repeat("scroll-me-sideways-", longCodeBlockWideLineRepeats)
	if !strings.Contains(codeRow.body, wideLine) {
		t.Error("長大なコードブロックに、右端を越える長さの行が含まれていない")
	}
}

func TestSandboxPageCopyKeepsIntendedLanguages(t *testing.T) {
	t.Parallel()

	// These expectations intentionally stay independent of the production
	// titles and prose. The seed needs Japanese explanatory copy while keeping
	// the two contents used to exercise English wrapping in ASCII.
	//
	// [Ja] 期待値は実装側のタイトルや説明文から独立させる。シードの解説文を日本語に
	// 保ちつつ、英語の折り返しを確認する 2 つの内容を ASCII のまま残す必要があるため。
	if wideTablePageTitle != "横に長いテーブル" {
		t.Errorf("横に長いテーブルのタイトルが %q であることを期待したが %q だった", "横に長いテーブル", wideTablePageTitle)
	}

	// The cells are not among these markers: they are half-width tokens by
	// design, and TestWideTableCellsHaveNowhereToBreak is what holds them to it.
	//
	// [Ja] セルはここのマーカーに含めない。設計上そこは半角のトークンであり、それを
	// 担保するのは TestWideTableCellsHaveNowhereToBreak のほうである。
	wideTableMarkers := []string{
		fmt.Sprintf("このページには %d 列のテーブル", wideTableColumns),
		"| 列 01 |",
	}
	wideTable := wideTableBody()
	for _, marker := range wideTableMarkers {
		if !strings.Contains(wideTable, marker) {
			t.Errorf("横に長いテーブルの本文に日本語のマーカー %q が含まれていない", marker)
		}
	}

	if longCodeBlockPageTitle != "長いコードブロック" {
		t.Errorf("長いコードブロックのタイトルが %q であることを期待したが %q だった", "長いコードブロック", longCodeBlockPageTitle)
	}

	// The markers hold the body to the two directions the block absorbs
	// differently: sideways it scrolls on its own, downwards the page is what
	// scrolls. A body that names the wrong one describes a screen the page
	// cannot show.
	//
	// [Ja] マーカーは本文を、ブロックが別々の形で受け止める 2 つの方向に縛る。横は
	// ブロック自身がスクロールし、縦はページのほうがスクロールする。どちらかを取り
	// 違えた本文は、このページが見せられない画面を説明することになる。
	longCodeBlockMarkers := []string{
		"下のコードブロックには",
		"ブロック自身が横にスクロール",
		"スクロールするのはページのほう",
	}
	longCodeBlock := longCodeBlockBody()
	for _, marker := range longCodeBlockMarkers {
		if !strings.Contains(longCodeBlock, marker) {
			t.Errorf("長いコードブロックの本文に日本語のマーカー %q が含まれていない", marker)
		}
	}

	codeParts := strings.Split(longCodeBlock, codeFence)
	if len(codeParts) != 3 {
		t.Fatalf("長いコードブロックの本文にコードフェンスが 1 組あることを期待したが %d 個の区画に分かれた", len(codeParts))
	}
	fencedCode := codeParts[1]
	if utf8.RuneCountInString(fencedCode) != len(fencedCode) {
		t.Error("長いコードブロックのコード本文がASCIIだけで書かれていない")
	}
	if !strings.Contains(fencedCode, "// wideLine is the one line that reaches past the right edge of the block.") {
		t.Error("長いコードブロックのコード本文に英語の説明が含まれていない")
	}

	if utf8.RuneCountInString(longTitlePageBody) != len(longTitlePageBody) {
		t.Error("長い英文タイトルのページ本文がASCIIだけで書かれていない")
	}
	if !strings.Contains(longTitlePageBody, "This page exists for its title") {
		t.Error("長い英文タイトルのページ本文に英語の説明が含まれていない")
	}
}

func TestWideTableCellsHaveNowhereToBreak(t *testing.T) {
	t.Parallel()

	// Every cell must remain one sufficiently long token under the browser's
	// line-breaking rules, otherwise the table may collapse back into the body
	// width and stop exercising horizontal overflow.
	//
	// [Ja] 各セルはブラウザの改行規則で十分に長い 1 トークンのままである必要がある。
	// 途中で折り返せるとテーブルが本文幅に収まり、横あふれを確認できなくなるため。
	body := wideTableBody()

	for row := 1; row <= wideTableRows; row++ {
		for column := 1; column <= wideTableColumns; column++ {
			cell := wideTableCell(row, column)

			for _, r := range cell {
				if !isUnbreakableTokenRune(r) {
					t.Errorf("セル %q に折り返しの機会になる文字 %q が含まれている", cell, r)
				}
			}
			if len(cell) < wideTableCellMinLength {
				t.Errorf("セル %q が %d 文字以上であることを期待したが %d 文字だった", cell, wideTableCellMinLength, len(cell))
			}
			if !strings.Contains(body, "| "+cell+" |") {
				t.Errorf("横に長いテーブルの本文にセル %q が含まれていない", cell)
			}
		}
	}
}

// isUnbreakableTokenRune reports whether r may sit in a token that a browser is
// given no opportunity to break: an ASCII letter, digit or underscore.
//
// [Ja] isUnbreakableTokenRune は、ブラウザに折り返す機会を与えないトークンに r を
// 置いてよいかを返す。置いてよいのは ASCII の英数字とアンダースコアだけ。
func isUnbreakableTokenRune(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z':
		return true
	case r >= 'A' && r <= 'Z':
		return true
	case r >= '0' && r <= '9':
		return true
	case r == '_':
		return true
	default:
		return false
	}
}

func TestSandboxPageTitlesStayWithinWhatTheAppAccepts(t *testing.T) {
	t.Parallel()

	// The long title is the page: it is written to sit exactly on the limit the
	// application enforces, so that the headings and the listings are looked at
	// in the worst case a member could actually create rather than in a longer
	// one nothing can reach.
	//
	// [Ja] 長いタイトルこそがそのページである。アプリケーションが課す上限に
	// ちょうど乗るように書いており、見出しや一覧を、誰も到達できない長さではなく
	// メンバーが実際に作れる最悪のケースで確認できるようにしている。
	if got := utf8.RuneCountInString(longTitlePageTitle); got != pageTitleLengthLimit {
		t.Errorf("長いタイトルが %d 文字であることを期待したが %d 文字だった", pageTitleLengthLimit, got)
	}

	// A title the application would reject cannot be reached from the screens,
	// so a page carrying one would show a state that no member can produce.
	// The rules mirror internal/validator's PageUpdateValidator.
	//
	// [Ja] アプリケーションが弾くタイトルは画面から作れないため、それを持つページは
	// どのメンバーにも作れない状態を見せることになる。ここの規則は
	// internal/validator の PageUpdateValidator に対応している。
	for _, spec := range sandboxPageSpecs() {
		if got := utf8.RuneCountInString(spec.title); got > pageTitleLengthLimit {
			t.Errorf("%q が %d 文字以内であることを期待したが %d 文字だった", spec.title, pageTitleLengthLimit, got)
		}
		if strings.ContainsAny(spec.title, `/\:*?"<>|`) {
			t.Errorf("%q にタイトルとして使えない文字が含まれている", spec.title)
		}
		if strings.HasPrefix(spec.title, " ") || strings.HasSuffix(spec.title, " ") ||
			strings.HasPrefix(spec.title, ".") || strings.HasSuffix(spec.title, ".") {
			t.Errorf("%q の先頭または末尾にスペースまたはドットがある", spec.title)
		}
	}
}

func TestSandboxPageTitlesCoverBothKindsOfWrapping(t *testing.T) {
	t.Parallel()

	// The long title is the only one written in a language that puts spaces
	// between words, and a line broken at a space breaks where a line broken
	// between characters does not. Rewriting it would leave every title the
	// seed writes wrapping by the same rules, and one of the two ways a
	// listing can be wrong would go unseen.
	//
	// [Ja] 長いタイトルは、単語の間に空白を置く言語で書かれた唯一のタイトルであり、
	// 空白で折り返される行は、文字の間で折り返される行とは違う位置で折り返される。
	// 書き換えると、シードが書くタイトルはすべて同じ規則で折り返すことになり、一覧が
	// 崩れる 2 通りのうち 1 通りを見られなくなる。
	if utf8.RuneCountInString(longTitlePageTitle) != len(longTitlePageTitle) {
		t.Errorf("%q がASCIIだけで書かれていることを期待したがマルチバイト文字を含んでいた", longTitlePageTitle)
	}
	if !strings.Contains(longTitlePageTitle, " ") {
		t.Errorf("%q が単語の間に空白を持つことを期待したが持っていなかった", longTitlePageTitle)
	}

	// The multibyte title exists to be neither one byte nor one column per
	// character. An ASCII-only title would leave the page with nothing to show.
	//
	// [Ja] マルチバイトのタイトルは、1 文字が 1 バイトでも 1 桁でもないために
	// 存在する。ASCII だけのタイトルでは、そのページに見せるものが無くなる。
	if utf8.RuneCountInString(multibyteTitlePageTitle) == len(multibyteTitlePageTitle) {
		t.Errorf("%q がマルチバイト文字を含むことを期待したがASCIIだけだった", multibyteTitlePageTitle)
	}
}

func TestMultibyteTitlePageBodyWritesInThePoliteRegister(t *testing.T) {
	t.Parallel()

	// The seed's Japanese bodies close their sentences with です / ます. A page
	// that closes with である reads as written by someone else, and this one is
	// met in the same listing as the other three.
	//
	// [Ja] シードの日本語の本文は文を です・ます で終える。である で終える本文は
	// 別の書き手が書いたものとして読まれ、このページは他の 3 ページと同じ一覧で
	// 出会うものになる。
	for _, line := range strings.Split(multibyteTitlePageBody, "\n") {
		if line == "" || strings.HasPrefix(line, "|") || strings.HasPrefix(line, "- ") {
			continue
		}

		for _, sentence := range strings.Split(line, "。") {
			// A sentence may close with an emoji instead of a full stop.
			//
			// [Ja] 文が句点ではなく絵文字で終わることがある。
			sentence = strings.TrimRight(sentence, " 🎉")
			if sentence == "" {
				continue
			}
			if !strings.HasSuffix(sentence, "です") && !strings.HasSuffix(sentence, "ます") {
				t.Errorf("マルチバイトタイトルのページの文が です・ます で終わっていない: %q", sentence)
			}
		}
	}
}

func TestSandboxPageBodiesCarryNoWikilinks(t *testing.T) {
	t.Parallel()

	// A wiki link in one of these bodies would have the resolver create the
	// page it names, adding a page to a topic whose listing counts are decided
	// elsewhere. These pages stress the layout, and each of them is written to
	// stress exactly one part of it.
	//
	// [Ja] これらの本文に Wiki リンクがあると、resolver がその名前のページを作成し、
	// 一覧の件数を別の場所で決めているトピックにページが増える。これらのページは
	// レイアウトに負荷をかけるためのもので、それぞれが 1 箇所だけに負荷をかけるよう
	// 書かれている。
	for _, spec := range sandboxPageSpecs() {
		if got := len(markup.ScanWikilinks(spec.body, topicNameSandbox)); got != 0 {
			t.Errorf("%q の本文にWikiリンクが %d 件含まれている", spec.title, got)
		}
	}
}
