package seed

import (
	"context"
	"io"

	"github.com/wikinoapp/wikino/go/internal/query"
)

// エクスポート確認用ページのタイトル。どれもそれ自体がページの主題になる。
// タイトルが持つものがアーカイブの変換すべきものであり、その変換がどの名前へ辿り
// 着くはずなのかを本文が述べる。
//
// 衝突する2つのタイトルは組になっている。一方は半角のアスタリスクで、もう一方は
// その置き換え先である全角のアスタリスクで書かれており、同じトピックが持てる2つの
// タイトルのまま、同じファイル名へ辿り着く。大文字小文字だけが違う組もファイル
// システムは同じ名前として畳むが、pages.titleはcitextであり、1つのトピックが
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

// exportMissingLinkPageTitleは、Wikiリンクだけが名指しするページ。resolverは
// 画面から公開したときと同じくこれを未公開のページとして作成し、未公開のページは
// アーカイブから外れる。それによってアーカイブの中のリンクは行き先を失う。Wikiリンクの
// ページが見せたいのはこの状態である。
const exportMissingLinkPageTitle = "エクスポートに含まれないページ"

// exportPageSpecは、エクスポートが何を変換するのかを見せるために書くページ
// 1件の内容。
//
// topicをトピックそのものではなく関数にしているのは、仕様の一覧がトピックより先に
// 組み立てられるため。一覧はどのページがどのトピックに属するのかを述べ、生成器は
// 渡されたトピックからそれを読み取る。
type exportPageSpec struct {
	title string
	body  string
	topic func(topics *seededTopics) *seededTopic
}

// exportPageSpecsは、エクスポートの2つのトピックのページを、作成する順に
// 並べる。
//
// これは読む順ではない。Wikiリンクのページを末尾に置いているのは、このページが
// 両方のトピックのページへリンクしており、まだ書かれていないページへのリンクは
// resolverにそれを未公開のページとして作成させるためである。そうなると生成器は、
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

// generateExportPagesはエクスポートの2つのトピックのページを作成する。各ページ
// は、エクスポートがZIPを作るまでに行う変換のうち1つを持つ。
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

// exportForbiddenCharsPageBodyは、タイトルが持つ2組の文字について述べる。
// ファイルシステムが受け付けないものと、ObsidianがWikiリンクの中で構文として
// 読むものである。どちらも全角になって出てくる。
const exportForbiddenCharsPageBody = `このページのタイトルには、ファイルシステムが受け付けない記号と、Obsidian が Wiki リンクの中で構文として読む記号が並んでいます。

エクスポートした ZIP では、これらがまとめて全角に置き換わり、ファイル名は「記号 ＊？”＜＞｜＃＾［］ を含むタイトル.md」になります。置き換える前のタイトルは frontmatter の wikino_title に残るため、ファイル名から失われた記号はそちらで読めます。
`

// exportPercentAndDotPageBodyは、このタイトルに両方とも当てはまるという点の
// ほかに互いの関係が無い2つの変換を扱う。Obsidianがコメントとして読む連続した
// パーセント記号と、Windowsが渡されたファイル名から落とす末尾のドットである。
const exportPercentAndDotPageBody = `このページのタイトルには、連続するパーセント記号と、末尾のドットが入っています。

Obsidian は Wiki リンクの中の連続するパーセント記号をコメントの開始と終了として読むため、エクスポートはこれを全角に置き換えます。末尾のドットは Windows がファイル名から落としてしまうので、変換の時点で取り除きます。ファイル名は「％％コメント％％ と末尾のドット.md」になります。
`

// exportReservedNamePageBodyは、ファイル名に付く接尾辞が何のためのものかを
// 述べる。タイトル自体はありふれた語であり、アーカイブがそれを別の綴りにしなければ
// ならないことは画面のどこからも窺えない。
const exportReservedNamePageBody = `「CON」は、Windows がファイルではなくデバイスとして解決する名前です。

そのままのファイル名では、展開したフォルダーを Windows で開いたときに、このページがファイルとして扱われません。エクスポートは接尾辞を付けて「CON_.md」にします。変えるのはファイル名だけで、ページのタイトルは「CON」のまま残ります。
`

// exportCollisionSourcePageBodyは、アスタリスクが置き換えられるほうの半分。
// 2つが書き出される順を述べているのは、どちらが素の名前を保つのかをその順が
// 決めるためである。
const exportCollisionSourcePageBody = `このページと「` + exportCollisionTargetPageTitle + `」は、アスタリスクが半角か全角かだけが違うタイトルの組です。

エクスポートは半角のアスタリスクを全角へ置き換えるため、この 2 つは同じファイル名になろうとします。先に書き出すほうが「` + exportCollisionTargetPageTitle + `.md」を取り、あとから来るほうに連番が付きます。先になるのはページ番号が小さいこのページです。
`

// exportCollisionTargetPageBodyは譲るほうの半分。自分のタイトルには置き換えられる
// ものが無く、衝突するのはもう一方のタイトルがこれへ変換されるためである。連番は拡張子
// より前に付き、それによってこのファイルはMarkdownのファイルのままでいられる。
const exportCollisionTargetPageBody = `このページのタイトルは、はじめから全角のアスタリスクで書かれています。

置き換えるものが無いため変換はタイトルをそのまま通しますが、「` + exportCollisionSourcePageTitle + `」が先に同じ名前を取っているので、このページのファイル名は「` + exportCollisionTargetPageTitle + `-2.md」になります。連番は拡張子の前に付き、拡張子は「.md」のまま残ります。
`

// exportWikilinkPageBodyは、アーカイブが区別しなければならない4種類のWiki
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

// exportFrontmatterPageBodyは既にfrontmatterのブロックを持っており、
// エクスポートはその前に2つ目のブロックを書くのではなく、このブロックへキーを足す
// 必要がある。
//
// wikino_titleを書き手のキーより前ではなく後ろに置いている。シードは段落が2行目へ
// 折り返された本文を書かず、スカラーのキーが2行続くとちょうどそのように読めてしまう。
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

// exportFrontmatterLikePageBodyは区切りで始まるが、その間にmappingを持たない。
// そのためエクスポートはこれをそのままの位置に残し、自分のブロックをその前に書く。
// 区切りに挟まれているのはキーと値ではなく1つの文であり、この違いこそがこのページの
// 存在する理由である。
const exportFrontmatterLikePageBody = `---
ここは frontmatter に見えますが、キーと値の組になっていないため YAML の mapping ではありません
---

区切りに挟まれていても、mapping として読めない文章は frontmatter になりません。

エクスポートはこの部分を本文のまま残し、新しい frontmatter をその前に付けます。ZIP の中のファイルは wikino_title を持つブロックで始まり、そのあとに上の区切りと文章が続きます。
`
