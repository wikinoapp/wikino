package seed

import (
	"context"
	"io"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/wikinoapp/wikino/go/internal/exportfile"
	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// exportPageExtension is the extension an export gives a page file. It mirrors
// exportPageExt, which is unexported in the usecase package, and it is what the
// names below are built with.
//
// [Ja] exportPageExtension はエクスポートがページのファイルに与える拡張子。usecase
// パッケージで非公開になっている exportPageExt に対応しており、下の名前はこれを使って
// 組み立てる。
const exportPageExtension = ".md"

func TestGenerateExportPages(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	spaces := buildSeedSpaces(t, tx, "seed-export")
	topics, err := generateTopics(ctx, tx, io.Discard, spaces)
	if err != nil {
		t.Fatalf("トピック生成に失敗: %v", err)
	}

	if err := generateExportPages(ctx, tx, io.Discard, spaces, topics); err != nil {
		t.Fatalf("エクスポート確認用ページの生成に失敗: %v", err)
	}

	// Every page has to be published: an export carries only published pages, so
	// a page left unpublished would be missing from the very archive it was
	// written to appear in.
	//
	// [Ja] すべてのページが公開済みである必要がある。エクスポートが持つのは公開済みの
	// ページだけであり、未公開のまま残ったページは、そこに現れるために書かれた当の
	// アーカイブから抜け落ちる。
	for _, spec := range exportPageSpecs() {
		topic := spec.topic(topics)
		if page := findPageByTitle(ctx, t, tx, spaces.wiki.id, topic.id, spec.title); !page.published {
			t.Errorf("%q が公開済みであることを期待したが未公開だった", spec.title)
		}
	}

	// The topic holds one more page than the specs name it. The wiki link that
	// points at a page nobody wrote is what creates it, and the count is what
	// says the link reached the resolver rather than being left as plain text.
	//
	// [Ja] トピックは仕様が挙げるより 1 件多くページを持つ。誰も書いていないページを
	// 指す Wiki リンクがそれを作る。件数は、そのリンクがプレーンテキストのまま残らず
	// resolver まで届いたことを示す。
	wantExportPages := countSpecsInTopic(topics, topics.export) + 1
	if got := countPagesInTopic(ctx, t, tx, spaces.wiki.id, topics.export.id); got != wantExportPages {
		t.Errorf("「%s」のページが %d 件であることを期待したが %d 件だった", topicNameExport, wantExportPages, got)
	}
	wantSymbolPages := countSpecsInTopic(topics, topics.exportSymbol)
	if got := countPagesInTopic(ctx, t, tx, spaces.wiki.id, topics.exportSymbol.id); got != wantSymbolPages {
		t.Errorf("「%s」のページが %d 件であることを期待したが %d 件だった", topicNameExportSymbol, wantSymbolPages, got)
	}

	// The page the link created has to stay unpublished. A published one would
	// be carried by the archive, and the link in it would resolve: the state the
	// wiki link page exists to show is a link the archive cannot point anywhere.
	//
	// [Ja] リンクが作ったページは未公開のままである必要がある。公開されていれば
	// アーカイブがそれを持ち、リンクは解決してしまう。Wiki リンクのページが見せたいのは、
	// アーカイブがどこへも向けられないリンクという状態である。
	missing := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.export.id, exportMissingLinkPageTitle)
	if missing.published {
		t.Errorf("%q が未公開であることを期待したが公開済みだった", exportMissingLinkPageTitle)
	}

	// The three links outside the code block have to have been resolved. Two of
	// them name pages that already exist, one of them across a topic boundary,
	// and the third names the page created above.
	//
	// [Ja] コードブロックの外にある 3 つのリンクは解決されている必要がある。うち 2 つは
	// 既にあるページを名指しし、片方はトピックをまたぐ。3 つ目は上で作られたページを
	// 名指しする。
	wikilinkPage := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.export.id, exportWikilinkPageTitle)
	wikilinkRow := readPage(ctx, t, tx, spaces.wiki.id, wikilinkPage.id)
	if got := len(wikilinkRow.linkedPageIDs); got != 3 {
		t.Errorf("%q のリンク先が 3 件であることを期待したが %d 件だった", exportWikilinkPageTitle, got)
	}

	percentPage := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.export.id, exportPercentAndDotPageTitle)
	frontmatterPage := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.exportSymbol.id, exportFrontmatterPageTitle)
	for _, target := range []foundPage{percentPage, frontmatterPage, missing} {
		assertContains(t, wikilinkRow.bodyHTML, hrefOf(spaces.wiki, target.number))
	}
}

func TestExportPageSpecsCreateLinkTargetsFirst(t *testing.T) {
	t.Parallel()

	// A wiki link naming a page that has not been created yet has the resolver
	// create it as an unpublished page, and the generator would then fail to
	// publish a page whose title its own link has taken. The order of the specs
	// is what keeps that from happening, and nothing in the generator would
	// catch it being changed.
	//
	// [Ja] まだ作成されていないページを名指しする Wiki リンクは、resolver にそれを
	// 未公開のページとして作らせる。すると生成器は、自分のリンクがタイトルを取って
	// しまったページを公開できずに失敗する。それを防いでいるのは仕様の並び順であり、
	// その順が変わったことに気づけるものは生成器の中に無い。
	created := map[string]bool{}

	for _, spec := range exportPageSpecs() {
		for _, key := range markup.ScanWikilinks(spec.body, topicNameForSpec(spec)) {
			if key.PageTitle == exportMissingLinkPageTitle {
				continue
			}
			if !created[key.TopicName+"/"+key.PageTitle] {
				t.Errorf(
					"%q の Wiki リンクが、まだ作成されていないページ %q を指している",
					spec.title, key.TopicName+"/"+key.PageTitle,
				)
			}
		}

		created[topicNameForSpec(spec)+"/"+spec.title] = true
	}
}

func TestExportWikilinkPageBodyCarriesTheFourKinds(t *testing.T) {
	t.Parallel()

	// Only three of the four reach the resolver: the fourth sits inside a code
	// block, which is the author showing the syntax rather than following it.
	// A body whose fence stopped working would hand the resolver a fourth link
	// and turn the page into one that shows three kinds instead of four.
	//
	// [Ja] resolver まで届くのは 4 つのうち 3 つだけである。4 つ目はコードブロックの
	// 中にあり、それは書き手が記法を辿るのではなく見せている部分だからである。フェンスが
	// 効かなくなった本文は resolver へ 4 つ目のリンクを渡し、そのページを 4 種類ではなく
	// 3 種類を見せるページに変えてしまう。
	keys := markup.ScanWikilinks(exportWikilinkPageBody, topicNameExport)
	if len(keys) != 3 {
		t.Fatalf("コードブロックの外の Wiki リンクが 3 件であることを期待したが %d 件だった", len(keys))
	}

	for i, want := range []markup.WikilinkKey{
		{TopicName: topicNameExport, PageTitle: exportPercentAndDotPageTitle},
		{TopicName: topicNameExportSymbol, PageTitle: exportFrontmatterPageTitle},
		{TopicName: topicNameExport, PageTitle: exportMissingLinkPageTitle},
	} {
		if keys[i].TopicName != want.TopicName || keys[i].PageTitle != want.PageTitle {
			t.Errorf(
				"%d 番目の Wiki リンクが %q を指すことを期待したが %q を指していた",
				i+1,
				want.TopicName+"/"+want.PageTitle,
				keys[i].TopicName+"/"+keys[i].PageTitle,
			)
		}
	}

	// The fourth link is written in the source even though no scan reports it.
	//
	// [Ja] 4 つ目のリンクは、どの走査も報告しないがソースには書かれている。
	fenced := strings.Split(exportWikilinkPageBody, codeFence)
	if len(fenced) != 3 {
		t.Fatalf("本文にコードフェンスが 1 組あることを期待したが %d 個の区画に分かれた", len(fenced))
	}
	if !strings.Contains(fenced[1], "[["+exportPercentAndDotPageTitle+"]]") {
		t.Errorf("コードブロックの中に 4 つ目の Wiki リンクが書かれていない: %q", fenced[1])
	}
}

func TestExportPageTitlesReachTheirArchiveNames(t *testing.T) {
	t.Parallel()

	// Each title is the page. What it carries decides the name the archive
	// gives it, and a title edited for readability would leave the page showing
	// a conversion that no longer happens. The names are asked of exportfile
	// rather than written out as expectations of the seed, so that this holds
	// the titles to the conversion as it is rather than to a copy of it.
	//
	// [Ja] タイトルこそがそのページである。タイトルが持つものがアーカイブの与える名前を
	// 決めており、読みやすさのために書き換えられたタイトルは、もう起きない変換を見せる
	// ページを残す。名前はシード側の期待値として書き出さず exportfile に尋ねる。そうする
	// ことで、変換の写しではなく変換そのものにタイトルを縛れる。
	for _, tt := range []struct {
		title string
		want  string
	}{
		{title: exportForbiddenCharsPageTitle, want: "記号 ＊？”＜＞｜＃＾［］ を含むタイトル.md"},
		{title: exportPercentAndDotPageTitle, want: "％％コメント％％ と末尾のドット.md"},
		{title: exportReservedNamePageTitle, want: "CON_.md"},
	} {
		if got := exportfile.Name(tt.title, exportPageExtension); got != tt.want {
			t.Errorf("%q のファイル名が %q になることを期待したが %q だった", tt.title, tt.want, got)
		}
	}

	// The topic name is converted as well, and the halfwidth "*" in it is the
	// only reason that topic is spelled the way it is.
	//
	// [Ja] トピック名も変換される。その半角の "*" だけが、あのトピックがああ綴られて
	// いる理由である。
	if got := exportfile.Name(topicNameExportSymbol, ""); got != "エクスポート＊記号" {
		t.Errorf("トピック名 %q のディレクトリ名が %q になることを期待したが %q だった",
			topicNameExportSymbol, "エクスポート＊記号", got)
	}

	// The colliding pair is handed its names by one Deduper, the way the pages
	// of one topic directory are. Asked one at a time the second would come out
	// without its counter, and the collision the pair exists to show would not
	// appear.
	//
	// [Ja] 衝突する組は、1 つのトピックのディレクトリのページと同じく、1 つの Deduper
	// から名前を受け取る。1 つずつ尋ねれば 2 つ目は連番の付かない名前で出てきてしまい、
	// この組が見せようとしている衝突は現れない。
	deduper := exportfile.NewDeduper()
	for _, tt := range []struct {
		title string
		want  string
	}{
		{title: exportCollisionSourcePageTitle, want: "衝突するタイトル＊1.md"},
		{title: exportCollisionTargetPageTitle, want: "衝突するタイトル＊1-2.md"},
	} {
		if got := deduper.Unique(tt.title, exportPageExtension); got != tt.want {
			t.Errorf("%q のファイル名が %q になることを期待したが %q だった", tt.title, tt.want, got)
		}
	}
}

func TestExportPageTitlesStayWithinWhatTheAppAccepts(t *testing.T) {
	t.Parallel()

	// These titles are written to carry what a file name may not, so the line
	// between "converted by the export" and "refused by the application" is
	// exactly what they walk along. A title the application would reject cannot
	// be reached from the screens, and a page carrying one would show a state no
	// member can produce. The rules mirror internal/validator's
	// PageUpdateValidator.
	//
	// [Ja] これらのタイトルは、ファイル名が持てないものを持つように書かれている。
	// そのため「エクスポートが変換するもの」と「アプリケーションが拒否するもの」の境目
	// こそが、これらのタイトルの歩く線になる。アプリケーションが弾くタイトルは画面から
	// 作れないため、それを持つページはどのメンバーにも作れない状態を見せることになる。
	// ここの規則は internal/validator の PageUpdateValidator に対応している。
	titles := make([]string, 0, len(exportPageSpecs())+1)
	for _, spec := range exportPageSpecs() {
		titles = append(titles, spec.title)
	}
	titles = append(titles, exportMissingLinkPageTitle)

	for _, title := range titles {
		if got := utf8.RuneCountInString(title); got > pageTitleLengthLimit {
			t.Errorf("%q が %d 文字以内であることを期待したが %d 文字だった", title, pageTitleLengthLimit, got)
		}
		if strings.ContainsAny(title, `/\:`) {
			t.Errorf("%q にタイトルとして使えない文字が含まれている", title)
		}
		if strings.ContainsFunc(title, unicode.IsControl) {
			t.Errorf("%q に制御文字が含まれている", title)
		}
		if strings.HasPrefix(title, " ") || strings.HasSuffix(title, " ") {
			t.Errorf("%q の先頭または末尾にスペースがある", title)
		}
	}
}

// countSpecsInTopic counts the export pages the specs write into topic.
//
// [Ja] countSpecsInTopic は、仕様が topic へ書き込むエクスポート確認用ページを数える。
func countSpecsInTopic(topics *seededTopics, topic *seededTopic) int {
	count := 0
	for _, spec := range exportPageSpecs() {
		if spec.topic(topics) == topic {
			count++
		}
	}

	return count
}

// topicNameForSpec names the topic a spec is written into, which is what a wiki
// link without a topic in it is read against.
//
// [Ja] topicNameForSpec は、仕様が書き込まれるトピックの名前を返す。トピックを持たない
// Wiki リンクは、この名前を補って読まれる。
func topicNameForSpec(spec exportPageSpec) string {
	export := &seededTopic{name: topicNameExport}
	exportSymbol := &seededTopic{name: topicNameExportSymbol}

	return spec.topic(&seededTopics{export: export, exportSymbol: exportSymbol}).name
}

func TestExportPageBodiesReachTheirArchiveForms(t *testing.T) {
	t.Parallel()

	// The body that already carries frontmatter has to receive the key into the
	// block it has rather than have a second block written in front of it, and
	// the key the author wrote has to survive. A body whose YAML stopped parsing
	// would still export, as the page below does, and the difference between the
	// two pages is exactly what would be lost.
	//
	// [Ja] はじめから frontmatter を持つ本文は、前に 2 つ目のブロックを書かれるのではなく、
	// 持っているブロックへキーを受け取る必要があり、書き手が置いたキーは残る必要がある。
	// YAML として読めなくなった本文も、下のページと同じくエクスポートはされる。そして
	// 失われるのは、まさにこの 2 ページの違いである。
	withFrontmatter := exportfile.WithFrontmatter(exportFrontmatterPageBody, exportFrontmatterPageTitle)
	if got := strings.Count(withFrontmatter, "\n---\n"); got != 1 {
		t.Errorf("%q の frontmatter が 1 ブロックであることを期待したが区切りが %d 箇所あった: %q",
			exportFrontmatterPageTitle, got+1, withFrontmatter)
	}
	for _, want := range []string{
		`wikino_title: "` + exportFrontmatterPageTitle + `"`,
		"tags:\n  - エクスポート\n",
	} {
		if !strings.Contains(withFrontmatter, want) {
			t.Errorf("%q の frontmatter に %q が含まれていない: %q", exportFrontmatterPageTitle, want, withFrontmatter)
		}
	}

	// The body that only looks like frontmatter has to keep its delimiters where
	// they are and receive a block of its own in front of them.
	//
	// [Ja] frontmatter に見えるだけの本文は、区切りをその位置に保ったまま、その前に
	// 自分のブロックを受け取る必要がある。
	frontmatterLike := exportfile.WithFrontmatter(exportFrontmatterLikePageBody, exportFrontmatterLikePageTitle)
	wantPrefix := "---\nwikino_title: \"" + exportFrontmatterLikePageTitle + "\"\n---\n\n"
	if !strings.HasPrefix(frontmatterLike, wantPrefix) {
		t.Errorf("%q が %q で始まることを期待したが %q だった", exportFrontmatterLikePageTitle, wantPrefix, frontmatterLike)
	}
	if !strings.HasSuffix(frontmatterLike, exportFrontmatterLikePageBody) {
		t.Errorf("%q の本文が原文のまま残ることを期待したが %q だった", exportFrontmatterLikePageTitle, frontmatterLike)
	}

	// Of the four wiki links, the two with a page in the archive are rewritten
	// onto its path, and the other two are left as they are: one names a page
	// the archive does not carry, and one is the author showing the syntax.
	//
	// [Ja] 4 つの Wiki リンクのうち、アーカイブにページがある 2 つはそのパスへ書き換え
	// られ、残る 2 つはそのまま残る。一方はアーカイブが持たないページを名指しし、もう
	// 一方は書き手が記法を見せているものである。
	const (
		percentPath     = "エクスポート/％％コメント％％ と末尾のドット.md"
		frontmatterPath = "エクスポート＊記号/frontmatter のあるページ.md"
	)
	rewritten := exportfile.RewriteWikilinks(exportWikilinkPageBody, topicNameExport, map[exportfile.PageRef]string{
		{TopicName: topicNameExport, PageTitle: exportPercentAndDotPageTitle}:     percentPath,
		{TopicName: topicNameExportSymbol, PageTitle: exportFrontmatterPageTitle}: frontmatterPath,
	})
	for _, want := range []string{
		"[[" + percentPath + "]]",
		"[[" + frontmatterPath + "]]",
		"[[" + exportMissingLinkPageTitle + "]]",
		codeFence + "\n[[" + exportPercentAndDotPageTitle + "]]\n" + codeFence,
	} {
		if !strings.Contains(rewritten, want) {
			t.Errorf("書き換え後の %q に %q が含まれていない: %q", exportWikilinkPageTitle, want, rewritten)
		}
	}
}
