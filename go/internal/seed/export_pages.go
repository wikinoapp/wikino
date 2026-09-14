package seed

import (
	"context"
	"io"

	"github.com/wikinoapp/wikino/go/internal/query"
)

// Titles of the export pages. Each of them is the point of its own page: what
// the title carries is what the archive has to convert, and the body says which
// name the conversion is expected to arrive at.
//
// The two collision titles are a pair. One is written with the halfwidth
// asterisk and the other with the fullwidth one it is replaced by, so the two
// arrive at the same file name while remaining two titles the topic can hold.
// A pair differing in case alone is what a file system would also fold, but
// pages.title is citext and one topic cannot hold both spellings of it.
//
// [Ja] エクスポート確認用ページのタイトル。どれもそれ自体がページの主題になる。
// タイトルが持つものがアーカイブの変換すべきものであり、その変換がどの名前へ辿り
// 着くはずなのかを本文が述べる。
//
// 衝突する 2 つのタイトルは組になっている。一方は半角のアスタリスクで、もう一方は
// その置き換え先である全角のアスタリスクで書かれており、同じトピックが持てる 2 つの
// タイトルのまま、同じファイル名へ辿り着く。大文字小文字だけが違う組もファイル
// システムは同じ名前として畳むが、pages.title は citext であり、1 つのトピックが
// その両方の綴りを持つことはできない。
const (
	exportForbiddenCharsPageTitle  = `記号 *?"<>|#^[] を含むタイトル`
	exportPercentAndDotPageTitle   = "%%コメント%% と末尾のドット..."
	exportReservedNamePageTitle    = "CON"
	exportCollisionSourcePageTitle = "衝突するタイトル*1"
	exportCollisionTargetPageTitle = "衝突するタイトル＊1"
	exportWikilinkPageTitle        = "Wiki リンクの書き換え"
	exportFrontmatterPageTitle     = "frontmatter のあるページ"
	exportFrontmatterLikePageTitle = "frontmatter に見える本文"
)

// exportMissingLinkPageTitle is the page that only a wiki link names. The
// resolver creates it as an unpublished page, the way publishing from the
// screen does, and an unpublished page is left out of the archive. That is what
// leaves the link in the archive with nowhere to point, which is the state the
// wiki link page exists to show.
//
// [Ja] exportMissingLinkPageTitle は、Wiki リンクだけが名指しするページ。resolver は
// 画面から公開したときと同じくこれを未公開のページとして作成し、未公開のページは
// アーカイブから外れる。それによってアーカイブの中のリンクは行き先を失う。Wiki リンクの
// ページが見せたいのはこの状態である。
const exportMissingLinkPageTitle = "エクスポートに含まれないページ"

// exportPageSpec describes one page written to show what an export converts.
//
// topic is a function rather than a topic because the specs are built before
// the topics exist: the list says which topic each page belongs to, and the
// generator reads it off the topics it is handed.
//
// [Ja] exportPageSpec は、エクスポートが何を変換するのかを見せるために書くページ
// 1 件の内容。
//
// topic をトピックそのものではなく関数にしているのは、仕様の一覧がトピックより先に
// 組み立てられるため。一覧はどのページがどのトピックに属するのかを述べ、生成器は
// 渡されたトピックからそれを読み取る。
type exportPageSpec struct {
	title string
	body  string
	topic func(topics *seededTopics) *seededTopic
}

// exportPageSpecs lists the pages of the two export topics in the order they
// are created.
//
// The order is not the order they are read in. The wiki link page sits last
// because it links to pages of both topics, and a link to a page that has not
// been written yet would have the resolver create it as an unpublished page:
// the generator would then try to publish a page whose title its own link has
// already taken. Everything the page links to is therefore created before it.
//
// [Ja] exportPageSpecs は、エクスポートの 2 つのトピックのページを、作成する順に
// 並べる。
//
// これは読む順ではない。Wiki リンクのページを末尾に置いているのは、このページが
// 両方のトピックのページへリンクしており、まだ書かれていないページへのリンクは
// resolver にそれを未公開のページとして作成させるためである。そうなると生成器は、
// 自分のリンクが既にタイトルを取ってしまったページを公開しようとすることになる。
// そのため、このページがリンクする先はすべて先に作成する。
func exportPageSpecs() []exportPageSpec {
	toExport := func(topics *seededTopics) *seededTopic { return topics.export }
	toExportSymbol := func(topics *seededTopics) *seededTopic { return topics.exportSymbol }

	return []exportPageSpec{
		{title: exportForbiddenCharsPageTitle, body: exportForbiddenCharsPageBody, topic: toExport},
		{title: exportPercentAndDotPageTitle, body: exportPercentAndDotPageBody, topic: toExport},
		{title: exportReservedNamePageTitle, body: exportReservedNamePageBody, topic: toExport},
		{title: exportCollisionSourcePageTitle, body: exportCollisionSourcePageBody, topic: toExport},
		{title: exportCollisionTargetPageTitle, body: exportCollisionTargetPageBody, topic: toExport},
		{title: exportFrontmatterPageTitle, body: exportFrontmatterPageBody, topic: toExportSymbol},
		{title: exportFrontmatterLikePageTitle, body: exportFrontmatterLikePageBody, topic: toExportSymbol},
		{title: exportWikilinkPageTitle, body: exportWikilinkPageBody, topic: toExport},
	}
}

// generateExportPages creates the pages of the two export topics, each of which
// carries one of the conversions an export makes on its way to a ZIP.
//
// Every conversion has a unit test of its own, but the other seed generators
// write no title a conversion applies to. Without these pages, an archive made
// right after a seed would keep every page title unchanged as its file name.
// These pages make those conversions observable from the browser.
//
// [Ja] generateExportPages はエクスポートの 2 つのトピックのページを作成する。各ページ
// は、エクスポートが ZIP を作るまでに行う変換のうち 1 つを持つ。
//
// どの変換にも単体テストがある一方、ほかのシード生成器は変換の対象になるタイトルを
// ひとつも書かない。これらのページが無ければ、シード直後のアーカイブでは、すべての
// ページタイトルが変更されずにファイル名になる。これらのページによって、それらの変換を
// ブラウザから観測できる。
func generateExportPages(
	ctx context.Context,
	dbtx query.DBTX,
	out io.Writer,
	spaces *seededSpaces,
	topics *seededTopics,
) error {
	specs := exportPageSpecs()

	bar := newProgress(out, "エクスポート確認用ページ", len(specs))
	defer bar.finish()

	writer := newPageWriter(dbtx, spaces.wiki)

	for i, spec := range specs {
		author, err := variationAuthor(spaces.wiki, i+1)
		if err != nil {
			return err
		}

		if _, err := writer.createPage(ctx, createPageInput{
			topic:  spec.topic(topics),
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

// exportForbiddenCharsPageBody explains the two groups of characters its title
// carries: the ones a file system refuses, and the ones Obsidian reads as
// syntax inside a wiki link. Both groups come out fullwidth.
//
// [Ja] exportForbiddenCharsPageBody は、タイトルが持つ 2 組の文字について述べる。
// ファイルシステムが受け付けないものと、Obsidian が Wiki リンクの中で構文として
// 読むものである。どちらも全角になって出てくる。
const exportForbiddenCharsPageBody = `このページのタイトルには、ファイルシステムが受け付けない記号と、Obsidian が Wiki リンクの中で構文として読む記号が並んでいます。

エクスポートした ZIP では、これらがまとめて全角に置き換わり、ファイル名は「記号 ＊？”＜＞｜＃＾［］ を含むタイトル.md」になります。置き換える前のタイトルは frontmatter の wikino_title に残るため、ファイル名から失われた記号はそちらで読めます。
`

// exportPercentAndDotPageBody covers the two conversions that have nothing to
// do with each other beyond both applying to this one title: a run of percent
// signs, which Obsidian reads as a comment, and a trailing dot, which Windows
// drops from a file name it is given.
//
// [Ja] exportPercentAndDotPageBody は、このタイトルに両方とも当てはまるという点の
// ほかに互いの関係が無い 2 つの変換を扱う。Obsidian がコメントとして読む連続した
// パーセント記号と、Windows が渡されたファイル名から落とす末尾のドットである。
const exportPercentAndDotPageBody = `このページのタイトルには、連続するパーセント記号と、末尾のドットが入っています。

Obsidian は Wiki リンクの中の連続するパーセント記号をコメントの開始と終了として読むため、エクスポートはこれを全角に置き換えます。末尾のドットは Windows がファイル名から落としてしまうので、変換の時点で取り除きます。ファイル名は「％％コメント％％ と末尾のドット.md」になります。
`

// exportReservedNamePageBody says what the suffix on its file name is for. The
// title itself is an ordinary word, and nothing on the screen suggests that the
// archive has to spell it differently.
//
// [Ja] exportReservedNamePageBody は、ファイル名に付く接尾辞が何のためのものかを
// 述べる。タイトル自体はありふれた語であり、アーカイブがそれを別の綴りにしなければ
// ならないことは画面のどこからも窺えない。
const exportReservedNamePageBody = `「CON」は、Windows がファイルではなくデバイスとして解決する名前です。

そのままのファイル名では、展開したフォルダーを Windows で開いたときに、このページがファイルとして扱われません。エクスポートは接尾辞を付けて「CON_.md」にします。変えるのはファイル名だけで、ページのタイトルは「CON」のまま残ります。
`

// exportCollisionSourcePageBody is the half whose asterisk is replaced. It
// names the order the two are written in, since that order is what decides
// which of them keeps the plain name.
//
// [Ja] exportCollisionSourcePageBody は、アスタリスクが置き換えられるほうの半分。
// 2 つが書き出される順を述べているのは、どちらが素の名前を保つのかをその順が
// 決めるためである。
const exportCollisionSourcePageBody = `このページと「` + exportCollisionTargetPageTitle + `」は、アスタリスクが半角か全角かだけが違うタイトルの組です。

エクスポートは半角のアスタリスクを全角へ置き換えるため、この 2 つは同じファイル名になろうとします。先に書き出すほうが「` + exportCollisionTargetPageTitle + `.md」を取り、あとから来るほうに連番が付きます。先になるのはページ番号が小さいこのページです。
`

// exportCollisionTargetPageBody is the half that has to give way. Nothing in
// its own title is replaced; it collides because the other title is converted
// into it. The counter goes before the extension, which is what keeps the file
// a Markdown file.
//
// [Ja] exportCollisionTargetPageBody は譲るほうの半分。自分のタイトルには置き換えられる
// ものが無く、衝突するのはもう一方のタイトルがこれへ変換されるためである。連番は拡張子
// より前に付き、それによってこのファイルは Markdown のファイルのままでいられる。
const exportCollisionTargetPageBody = `このページのタイトルは、はじめから全角のアスタリスクで書かれています。

置き換えるものが無いため変換はタイトルをそのまま通しますが、「` + exportCollisionSourcePageTitle + `」が先に同じ名前を取っているので、このページのファイル名は「` + exportCollisionTargetPageTitle + `-2.md」になります。連番は拡張子の前に付き、拡張子は「.md」のまま残ります。
`

// exportWikilinkPageBody holds the four kinds of wiki link an archive has to
// tell apart: one naming a page whose file name no longer spells its title, one
// naming a page of another topic, one naming a page the archive does not carry,
// and one the author is showing as syntax rather than following.
//
// The links are written out of the title constants so that renaming a page
// cannot leave a link here pointing at a title nothing holds any more.
//
// [Ja] exportWikilinkPageBody は、アーカイブが区別しなければならない 4 種類の Wiki
// リンクを持つ。ファイル名がもうタイトルを綴っていないページを指すもの、別のトピックの
// ページを指すもの、アーカイブが持たないページを指すもの、そして書き手が辿るためでは
// なく記法として見せているものである。
//
// リンクはタイトルの定数から組み立てる。ページの名前を変えたときに、ここのリンクだけが
// もうどこにも無いタイトルを指したまま残らないようにするためである。
const exportWikilinkPageBody = `このページの本文には 4 種類の Wiki リンクが並んでいます。エクスポートした ZIP では、行き先のあるものだけが、隣にあるファイルを指す形へ書き換わります。

- 変換されたタイトルを指すリンク: [[` + exportPercentAndDotPageTitle + `]]
- 別のトピックのページを指すリンク: [[` + topicNameExportSymbol + `/` + exportFrontmatterPageTitle + `]]
- 存在しないページを指すリンク: [[` + exportMissingLinkPageTitle + `]]

上の 2 つは、変換を経たファイル名を持つファイルへのリンクになります。3 つ目が指しているのは、このリンクによって作られた未公開のページです。未公開のページは ZIP に含まれないため、このリンクは書き換えられず、原文のまま残ります。

4 つ目はコードブロックの中にあります。書き手が記法そのものを見せている部分なので、エクスポートも画面も、ここをリンクとしては扱いません。

` + codeFence + `
[[` + exportPercentAndDotPageTitle + `]]
` + codeFence + `
`

// exportFrontmatterPageBody already carries a frontmatter block, which the
// export has to add its key to rather than write a second block in front of.
//
// wikino_title sits after the key the author wrote rather than before it. The
// seed writes no body whose paragraph is wrapped onto a second line, and two
// scalar keys in a row would read as exactly that; a key whose value is a list
// is followed by list items instead.
//
// [Ja] exportFrontmatterPageBody は既に frontmatter のブロックを持っており、
// エクスポートはその前に 2 つ目のブロックを書くのではなく、このブロックへキーを足す
// 必要がある。
//
// wikino_title を書き手のキーより前ではなく後ろに置いている。シードは段落が 2 行目へ
// 折り返された本文を書かず、スカラーのキーが 2 行続くとちょうどそのように読めてしまう。
// 値がリストであるキーの後ろに続くのはリストの項目になる。
const exportFrontmatterPageBody = `---
tags:
  - エクスポート
  - frontmatter
wikino_title: "書き手が置いたタイトル"
---

このページの本文は、はじめから frontmatter を持っています。

エクスポートは新しいブロックを足すのではなく、この frontmatter の wikino_title をページのタイトルで置き換えます。書き手が置いた tags はそのまま残るため、ZIP の中で変わるのは wikino_title の値だけになります。
`

// exportFrontmatterLikePageBody opens with the delimiter but holds no mapping
// between them, so the export leaves it where it is and writes its own block in
// front. What sits between the delimiters is a sentence rather than a key and a
// value, which is the difference the page exists to show.
//
// [Ja] exportFrontmatterLikePageBody は区切りで始まるが、その間に mapping を持たない。
// そのためエクスポートはこれをそのままの位置に残し、自分のブロックをその前に書く。
// 区切りに挟まれているのはキーと値ではなく 1 つの文であり、この違いこそがこのページの
// 存在する理由である。
const exportFrontmatterLikePageBody = `---
ここは frontmatter に見えますが、キーと値の組になっていないため YAML の mapping ではありません
---

区切りに挟まれていても、mapping として読めない文章は frontmatter になりません。

エクスポートはこの部分を本文のまま残し、新しい frontmatter をその前に付けます。ZIP の中のファイルは wikino_title を持つブロックで始まり、そのあとに上の区切りと文章が続きます。
`
