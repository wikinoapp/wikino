package seed

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/wikinoapp/wikino/go/internal/query"
)

// pageTitleLengthLimitはinternal/validatorのpageTitleMaxLengthに合わせた
// もので、あちらは非公開になっている。シードが書く最も長いタイトルをちょうどこの
// 長さにすることで、見出しや一覧を、任意に長くしたものではなく、アプリケーションが
// 実際に受け付ける最悪のケースで確認できるようにする。
const pageTitleLengthLimit = 200

// 生成する2つの本文の大きさ。ビルダーの中にリテラルで置かず定数にしているのは、
// 各本文がなぜ過大なのかを、出力を数えずに読めるようにするため。テーブルはページが
// 描画されるどの段よりも横に広く、コードブロックはどの画面よりも縦に長い。
const (
	wideTableColumns = 14
	wideTableRows    = 12

	// wideTableCellMinLengthは、テーブルが読まれる段の幅を超えたままでいられる
	// セルの最小の長さ。この長さの半角文字が14列並び、そのどれもブラウザには折り返せ
	// ないため、本文に与えられるどの幅よりも広くなる。
	wideTableCellMinLength = 16

	// longCodeBlockPrintLinesは、コードブロックを画面より縦に長くするために
	// 繰り返す出力文の件数。前後にあるpackage・import・宣言・括弧は数に含めない。
	longCodeBlockPrintLines = 200
	// longCodeBlockWideLineRepeatsは、ブロックの中で1行だけ突出して長い行を
	// 組み立てる。コードブロック内の横スクロールバーは、これによって初めて現れる。
	longCodeBlockWideLineRepeats = 30
)

// codeFenceはコードブロックの開始と終了を表す。定数にしているのは、Goのraw
// string literalが同じ文字で区切られるため、フェンスを含む本文をそれで書けないため。
const codeFence = "```"

// sandboxPageSpecは、レイアウトを普段は行かないところまで押し広げるために
// 書くページ1件の内容。
type sandboxPageSpec struct {
	title string
	body  string
}

// 表示崩れ確認用ページのタイトル。最初の2つはそれ自体がページの主題になる。
// 一方はタイトルが取りうる最大の長さで、もう一方は1バイトでも1桁でもない文字で
// 書かれている。
//
// 長いほうは、単語の間に空白を置く言語で書いたまま残す。それは他のどのページも
// 見せるようには書かれていない折り返しであり、書き換えるとスペース内のタイトルが
// すべて同じ規則で折り返すことになる。
const (
	longTitlePageTitle      = "A Page Whose Title Was Pasted In From A Whole Sentence And Runs On Long Enough To Wrap In Every Listing, Every Breadcrumb And Every Browser Tab That Tries To Show It Without Cutting It Short Somewhere"
	multibyteTitlePageTitle = "🌏 絵文字と日本語が混ざったタイトル 🎉 全角文字の折り返しと省略が正しく効くかを確認するためのページ"
	wideTablePageTitle      = "横に長いテーブル"
	longCodeBlockPageTitle  = "長いコードブロック"
)

// sandboxPageSpecsは「サンドボックス」トピックのページを組み立てる。
//
// パッケージ変数の一覧ではなく呼び出し時に組み立てるのは、2つの本文が上の大きさから
// 生成されるため。起動時に1度だけ作る一覧にすると、その大きさを何が決めているのかが
// 見えなくなる。
func sandboxPageSpecs() []sandboxPageSpec {
	return []sandboxPageSpec{
		{title: longTitlePageTitle, body: longTitlePageBody},
		{title: multibyteTitlePageTitle, body: multibyteTitlePageBody},
		{title: wideTablePageTitle, body: wideTableBody()},
		{title: longCodeBlockPageTitle, body: longCodeBlockBody()},
	}
}

// generateSandboxPagesは「サンドボックス」トピックのページを作成する。各ページは
// レイアウトのどこか1箇所を、通常のページが求める以上に押し広げる。
//
// シードの他のページで崩れない画面は、扱いやすい大きさの内容で崩れないことしか
// 示していない。ここのページは扱いにくい大きさそのものになる。収まり先の無い
// タイトル、別の規則で行が折り返される文字、段よりも広いテーブル、画面よりも長い
// コードブロック。
func generateSandboxPages(
	ctx context.Context,
	dbtx query.DBTX,
	out io.Writer,
	spaces *seededSpaces,
	topics *seededTopics,
) error {
	specs := sandboxPageSpecs()

	bar := newProgress(out, "表示崩れ確認用ページ", len(specs))
	defer bar.finish()

	writer := newPageWriter(dbtx, spaces.wiki)

	for i, spec := range specs {
		author, err := variationAuthor(spaces.wiki, i+1)
		if err != nil {
			return err
		}

		if _, err := writer.createPage(ctx, createPageInput{
			topic:  topics.sandbox,
			author: author,
			title:  spec.title,
			body:   spec.body,
		}); err != nil {
			return err
		}
		bar.advance()
	}

	return nil
}

// longTitlePageBodyは、タイトルこそが主題であるページで何を見るかを述べる。
// 本文自体は短くしている。長い本文にすると、画面が崩れうる箇所が2つになるため。
// 本文はタイトルと同じ言語で書いており、タイトルと本文が別々の読み手に向けて
// 書かれたようには見えないようにしている。
const longTitlePageBody = `This page exists for its title, which is as long as a title is allowed to get.

Look at it in the topic listing, in the space listing, in the heading of this page and in the browser tab. Each of those has a different amount of room, and the title has to be wrapped or cut short in each of them rather than pushing the layout sideways.
`

// multibyteTitlePageBodyは、テキストがラテン文字で書かれていないときに何が
// 変わるのかを確認する。行の折り返しが空白に従わなくなり、1文字が1バイトでも
// 1桁でもなくなる。シードの他の日本語の本文にもそうした文字は含まれるが、この本文は
// そのどちらも紛れ込む余地が無いように組み立ててある。空白をひとつも含まない段落、
// 全角の見出しを持つテーブル、そして絵文字。
const multibyteTitlePageBody = `このページはタイトルと本文の両方に、日本語と絵文字を含んでいます 🎉

日本語の文章は単語の間に空白を持たないため、英語の文章とは違う位置で折り返されます。次の段落は空白をひとつも含まないひとつづきの文であり、本文の幅に収まらないときに横へはみ出さずに折り返せているかを確認するためのものです。

長い文をここに置くのは折り返しの位置を確認するためであってここに書かれている内容そのものに意味があるわけではないのですがこうして空白をひとつも入れずに書き続けると英語の文章では起きない折り返し方をするのでその結果として本文の幅からはみ出していないかどうかを目で見て確かめられるようになるという話です。

| 項目           | 内容                                       |
| -------------- | ------------------------------------------ |
| 文字種         | ひらがな・カタカナ・漢字・絵文字 🌏        |
| 確認したいこと | 全角文字を含むテーブルの列幅               |
| 備考           | 見出しと本文で字幅が揃っているかも見ておく |

- 箇条書きの中の絵文字 🍣 と全角文字
- **強調**を含む日本語の行
- ` + "`コード`" + `を含む日本語の行
`

// wideTableBodyは、横あふれがページ本文の内側に収まっているかを確認できる幅の
// テーブルを組み立てる。各セルはASCIIの英数字とアンダースコアからなる十分に長い
// 1トークンであり、通常の改行機会を持たないため、列を合わせた幅が保たれる。
func wideTableBody() string {
	var b strings.Builder

	fmt.Fprintf(&b, `このページには %d 列のテーブルがあり、ページの幅では表示しきれません。

テーブルはテーブル自身がスクロールする必要があります。代わりにページのほうが広がると、画面にある他のものまで一緒に横へ引きずられ、ページを読むことすらできなくなります。

セルに置いてあるのは、途中で折り返せない半角のトークンです。日本語と数字だけのセルは 1 文字ずつ折り返してどの幅にも収まってしまい、テーブルが段の幅を超えなくなります。

`, wideTableColumns)

	b.WriteString("|")
	for column := 1; column <= wideTableColumns; column++ {
		fmt.Fprintf(&b, " 列 %02d |", column)
	}

	b.WriteString("\n|")
	for column := 1; column <= wideTableColumns; column++ {
		b.WriteString(" --------- |")
	}
	b.WriteString("\n")

	for row := 1; row <= wideTableRows; row++ {
		b.WriteString("|")
		for column := 1; column <= wideTableColumns; column++ {
			fmt.Fprintf(&b, " %s |", wideTableCell(row, column))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// wideTableCellはそのテーブルのセルを1つ書く。トークンは自分がどのセルなのかを
// 名乗るため、横へ移動した先がどの行のどの列なのかをセル自身から読み取れる。ASCIIの
// 英数字とアンダースコアにより、ブラウザの改行規則でも1つの折り返されないトークンに
// 保たれる。
func wideTableCell(row, column int) string {
	return fmt.Sprintf("no_break_r%02d_c%02d", row, column)
}

// longCodeBlockBodyは、画面よりも縦に長く、1行だけ画面よりも横に広い行を持つ
// コードブロックのページを組み立てる。
//
// ブロックは縦と横を別々の形で受け止めるため、本文にはどちらがどうなのかを書いている。
// 横はブロック自身がスクロールするためページの幅は保たれる。縦は高さに上限が無く、
// 行数のぶんだけ伸びるため、スクロールするのはページのほうになる。
func longCodeBlockBody() string {
	var b strings.Builder

	fmt.Fprintf(&b, `下のコードブロックには、%d 行の繰り返しの出力と、ページの幅をはるかに超える 1 行が含まれています。

横と縦では受け止め方が違います。長い 1 行はブロック自身が横にスクロールして受け止めるため、ページの幅は広がりません。縦にはブロックの高さに上限が無く、行数のぶんだけ伸びていくため、スクロールするのはページのほうになります。

`, longCodeBlockPrintLines)

	b.WriteString(codeFence + "go\n")
	b.WriteString("package main\n\n")
	b.WriteString(`import "fmt"` + "\n\n")
	b.WriteString("// wideLine is the one line that reaches past the right edge of the block.\n")
	b.WriteString(`const wideLine = "` + strings.Repeat("scroll-me-sideways-", longCodeBlockWideLineRepeats) + `"` + "\n\n")
	b.WriteString("func main() {\n")
	b.WriteString("\tfmt.Println(wideLine)\n")
	for number := 1; number <= longCodeBlockPrintLines; number++ {
		fmt.Fprintf(&b, "\tfmt.Printf(\"line %03d of %d\\n\")\n", number, longCodeBlockPrintLines)
	}
	b.WriteString("}\n")
	b.WriteString(codeFence + "\n")

	return b.String()
}
