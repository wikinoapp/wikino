package seed

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"unicode"
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

	// すべてのページが公開済みで、かつ「サンドボックス」に置かれている必要がある。
	// 未公開のページは一覧に出ず、これらのページの目的はURLで辿り着くことでは
	// なく、閲覧の途中で出会うことにあるため。
	if got := countPagesInTopic(ctx, t, tx, spaces.wiki.id, topics.sandbox.id); got != len(specs) {
		t.Errorf("「サンドボックス」のページが%d件であることを期待したが%d件だった", len(specs), got)
	}
	for _, spec := range specs {
		if page := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.sandbox.id, spec.title); !page.published {
			t.Errorf("%qが公開済みであることを期待したが未公開だった", spec.title)
		}
	}

	// 生成した本文は、組み立て元のMarkdownではなく保存されたHTMLで確認する。
	// レンダラーがテーブルとして読まなかった表や、閉じられなかったフェンスは、その
	// ページを、見せるために書いたものとはまったく別の見た目にしてしまうため。
	tablePage := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.sandbox.id, wideTablePageTitle)
	tableRow := readPage(ctx, t, tx, spaces.wiki.id, tablePage.id)
	if got := strings.Count(tableRow.bodyHTML, "<th>"); got != wideTableColumns {
		t.Errorf("横に長いテーブルの列が%d列であることを期待したが%d列だった", wideTableColumns, got)
	}
	if got := strings.Count(tableRow.bodyHTML, "<tr>"); got != wideTableRows+1 {
		t.Errorf("横に長いテーブルの行が%d行であることを期待したが%d行だった", wideTableRows+1, got)
	}

	codePage := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.sandbox.id, longCodeBlockPageTitle)
	codeRow := readPage(ctx, t, tx, spaces.wiki.id, codePage.id)
	if !strings.Contains(codeRow.bodyHTML, "<pre><code") {
		t.Errorf("長大なコードブロックがコードブロックとして描画されていない: %q", codeRow.bodyHTML)
	}
	if got := strings.Count(codeRow.body, "\tfmt.Printf("); got != longCodeBlockPrintLines {
		t.Errorf(
			"長大なコードブロックの繰り返し出力行が%d件であることを期待したが%d件だった",
			longCodeBlockPrintLines,
			got,
		)
	}

	// ブロックが長いだけでは縦方向しか確認できない。右端を越える行があって
	// 初めてコードブロックの横スクロールを確認できる。
	wideLine := strings.Repeat("scroll-me-sideways-", longCodeBlockWideLineRepeats)
	if !strings.Contains(codeRow.body, wideLine) {
		t.Error("長大なコードブロックに、右端を越える長さの行が含まれていない")
	}
}

func TestSandboxPageCopyKeepsIntendedLanguages(t *testing.T) {
	t.Parallel()

	// 期待値は実装側のタイトルや説明文から独立させる。シードの解説文を日本語に
	// 保ちつつ、英語の折り返しを確認する2つの内容をASCIIのまま残す必要があるため。
	if wideTablePageTitle != "横に長いテーブル" {
		t.Errorf("横に長いテーブルのタイトルが%qであることを期待したが%qだった", "横に長いテーブル", wideTablePageTitle)
	}

	// セルはここのマーカーに含めない。設計上そこは半角のトークンであり、それを
	// 担保するのはTestWideTableCellsHaveNowhereToBreakのほうである。
	wideTableMarkers := []string{
		fmt.Sprintf("このページには %d 列のテーブル", wideTableColumns),
		"| 列 01 |",
	}
	wideTable := wideTableBody()
	for _, marker := range wideTableMarkers {
		if !strings.Contains(wideTable, marker) {
			t.Errorf("横に長いテーブルの本文に日本語のマーカー%qが含まれていない", marker)
		}
	}

	if longCodeBlockPageTitle != "長いコードブロック" {
		t.Errorf("長いコードブロックのタイトルが%qであることを期待したが%qだった", "長いコードブロック", longCodeBlockPageTitle)
	}

	// マーカーは本文を、ブロックが別々の形で受け止める2つの方向に縛る。横は
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
			t.Errorf("長いコードブロックの本文に日本語のマーカー%qが含まれていない", marker)
		}
	}

	codeParts := strings.Split(longCodeBlock, codeFence)
	if len(codeParts) != 3 {
		t.Fatalf("長いコードブロックの本文にコードフェンスが1組あることを期待したが%d個の区画に分かれた", len(codeParts))
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

	// 各セルはブラウザの改行規則で十分に長い1トークンのままである必要がある。
	// 途中で折り返せるとテーブルが本文幅に収まり、横あふれを確認できなくなるため。
	body := wideTableBody()

	for row := 1; row <= wideTableRows; row++ {
		for column := 1; column <= wideTableColumns; column++ {
			cell := wideTableCell(row, column)

			for _, r := range cell {
				if !isUnbreakableTokenRune(r) {
					t.Errorf("セル%qに折り返しの機会になる文字%qが含まれている", cell, r)
				}
			}
			if len(cell) < wideTableCellMinLength {
				t.Errorf("セル%qが%d文字以上であることを期待したが%d文字だった", cell, wideTableCellMinLength, len(cell))
			}
			if !strings.Contains(body, "| "+cell+" |") {
				t.Errorf("横に長いテーブルの本文にセル%qが含まれていない", cell)
			}
		}
	}
}

// isUnbreakableTokenRuneは、ブラウザに折り返す機会を与えないトークンにrを
// 置いてよいかを返す。置いてよいのはASCIIの英数字とアンダースコアだけ。
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

	// 長いタイトルこそがそのページである。アプリケーションが課す上限に
	// ちょうど乗るように書いており、見出しや一覧を、誰も到達できない長さではなく
	// メンバーが実際に作れる最悪のケースで確認できるようにしている。
	if got := utf8.RuneCountInString(longTitlePageTitle); got != pageTitleLengthLimit {
		t.Errorf("長いタイトルが%d文字であることを期待したが%d文字だった", pageTitleLengthLimit, got)
	}

	// アプリケーションが弾くタイトルは画面から作れないため、それを持つページは
	// どのメンバーにも作れない状態を見せることになる。ここの規則は
	// internal/validatorのPageUpdateValidatorに対応している。
	for _, spec := range sandboxPageSpecs() {
		if got := utf8.RuneCountInString(spec.title); got > pageTitleLengthLimit {
			t.Errorf("%qが%d文字以内であることを期待したが%d文字だった", spec.title, pageTitleLengthLimit, got)
		}
		if strings.ContainsAny(spec.title, `/\:`) {
			t.Errorf("%qにタイトルとして使えない文字が含まれている", spec.title)
		}
		if strings.ContainsFunc(spec.title, unicode.IsControl) {
			t.Errorf("%qに制御文字が含まれている", spec.title)
		}
		if strings.HasPrefix(spec.title, " ") || strings.HasSuffix(spec.title, " ") {
			t.Errorf("%qの先頭または末尾にスペースがある", spec.title)
		}
	}
}

func TestSandboxPageTitlesCoverBothKindsOfWrapping(t *testing.T) {
	t.Parallel()

	// 長いタイトルは、単語の間に空白を置く言語で書かれた唯一のタイトルであり、
	// 空白で折り返される行は、文字の間で折り返される行とは違う位置で折り返される。
	// 書き換えると、シードが書くタイトルはすべて同じ規則で折り返すことになり、一覧が
	// 崩れる2通りのうち1通りを見られなくなる。
	if utf8.RuneCountInString(longTitlePageTitle) != len(longTitlePageTitle) {
		t.Errorf("%qがASCIIだけで書かれていることを期待したがマルチバイト文字を含んでいた", longTitlePageTitle)
	}
	if !strings.Contains(longTitlePageTitle, " ") {
		t.Errorf("%qが単語の間に空白を持つことを期待したが持っていなかった", longTitlePageTitle)
	}

	// マルチバイトのタイトルは、1文字が1バイトでも1桁でもないために
	// 存在する。ASCIIだけのタイトルでは、そのページに見せるものが無くなる。
	if utf8.RuneCountInString(multibyteTitlePageTitle) == len(multibyteTitlePageTitle) {
		t.Errorf("%qがマルチバイト文字を含むことを期待したがASCIIだけだった", multibyteTitlePageTitle)
	}
}

func TestMultibyteTitlePageBodyWritesInThePoliteRegister(t *testing.T) {
	t.Parallel()

	// シードの日本語の本文は文を です・ます で終える。である で終える本文は
	// 別の書き手が書いたものとして読まれ、このページは他の3ページと同じ一覧で
	// 出会うものになる。
	for _, line := range strings.Split(multibyteTitlePageBody, "\n") {
		if line == "" || strings.HasPrefix(line, "|") || strings.HasPrefix(line, "- ") {
			continue
		}

		for _, sentence := range strings.Split(line, "。") {
			// 文が句点ではなく絵文字で終わることがある。
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

	// これらの本文にWikiリンクがあると、resolverがその名前のページを作成し、
	// 一覧の件数を別の場所で決めているトピックにページが増える。これらのページは
	// レイアウトに負荷をかけるためのもので、それぞれが1箇所だけに負荷をかけるよう
	// 書かれている。
	for _, spec := range sandboxPageSpecs() {
		if got := len(markup.ScanWikilinks(spec.body, topicNameSandbox)); got != 0 {
			t.Errorf("%qの本文にWikiリンクが%d件含まれている", spec.title, got)
		}
	}
}
