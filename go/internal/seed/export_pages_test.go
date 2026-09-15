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

// exportPageExtensionはエクスポートがページのファイルに与える拡張子。usecase
// パッケージで非公開になっているexportPageExtに対応しており、下の名前はこれを使って
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

	// すべてのページが公開済みである必要がある。エクスポートが持つのは公開済みの
	// ページだけであり、未公開のまま残ったページは、そこに現れるために書かれた当の
	// アーカイブから抜け落ちる。
	for _, spec := range exportPageSpecs() {
		topic := spec.topic(topics)
		if page := findPageByTitle(ctx, t, tx, spaces.wiki.id, topic.id, spec.title); !page.published {
			t.Errorf("%qが公開済みであることを期待したが未公開だった", spec.title)
		}
	}

	// トピックは仕様が挙げるより1件多くページを持つ。誰も書いていないページを
	// 指すWikiリンクがそれを作る。件数は、そのリンクがプレーンテキストのまま残らず
	// resolverまで届いたことを示す。
	wantExportPages := countSpecsInTopic(topics, topics.export) + 1
	if got := countPagesInTopic(ctx, t, tx, spaces.wiki.id, topics.export.id); got != wantExportPages {
		t.Errorf("「%s」のページが%d件であることを期待したが%d件だった", topicNameExport, wantExportPages, got)
	}
	wantSymbolPages := countSpecsInTopic(topics, topics.exportSymbol)
	if got := countPagesInTopic(ctx, t, tx, spaces.wiki.id, topics.exportSymbol.id); got != wantSymbolPages {
		t.Errorf("「%s」のページが%d件であることを期待したが%d件だった", topicNameExportSymbol, wantSymbolPages, got)
	}

	// リンクが作ったページは未公開のままである必要がある。公開されていれば
	// アーカイブがそれを持ち、リンクは解決してしまう。Wikiリンクのページが見せたいのは、
	// アーカイブがどこへも向けられないリンクという状態である。
	missing := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.export.id, exportMissingLinkPageTitle)
	if missing.published {
		t.Errorf("%qが未公開であることを期待したが公開済みだった", exportMissingLinkPageTitle)
	}

	// コードブロックの外にある3つのリンクは解決されている必要がある。うち2つは
	// 既にあるページを名指しし、片方はトピックをまたぐ。3つ目は上で作られたページを
	// 名指しする。
	wikilinkPage := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.export.id, exportWikilinkPageTitle)
	wikilinkRow := readPage(ctx, t, tx, spaces.wiki.id, wikilinkPage.id)
	if got := len(wikilinkRow.linkedPageIDs); got != 3 {
		t.Errorf("%qのリンク先が3件であることを期待したが%d件だった", exportWikilinkPageTitle, got)
	}

	percentPage := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.export.id, exportPercentAndDotPageTitle)
	frontmatterPage := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.exportSymbol.id, exportFrontmatterPageTitle)
	for _, target := range []foundPage{percentPage, frontmatterPage, missing} {
		assertContains(t, wikilinkRow.bodyHTML, hrefOf(spaces.wiki, target.number))
	}
}

func TestExportPageSpecsCreateLinkTargetsFirst(t *testing.T) {
	t.Parallel()

	// まだ作成されていないページを名指しするWikiリンクは、resolverにそれを
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
					"%qのWikiリンクが、まだ作成されていないページ%qを指している",
					spec.title, key.TopicName+"/"+key.PageTitle,
				)
			}
		}

		created[topicNameForSpec(spec)+"/"+spec.title] = true
	}
}

func TestExportWikilinkPageBodyCarriesTheFourKinds(t *testing.T) {
	t.Parallel()

	// resolverまで届くのは4つのうち3つだけである。4つ目はコードブロックの
	// 中にあり、それは書き手が記法を辿るのではなく見せている部分だからである。フェンスが
	// 効かなくなった本文はresolverへ4つ目のリンクを渡し、そのページを4種類ではなく
	// 3種類を見せるページに変えてしまう。
	keys := markup.ScanWikilinks(exportWikilinkPageBody, topicNameExport)
	if len(keys) != 3 {
		t.Fatalf("コードブロックの外のWikiリンクが3件であることを期待したが%d件だった", len(keys))
	}

	for i, want := range []markup.WikilinkKey{
		{TopicName: topicNameExport, PageTitle: exportPercentAndDotPageTitle},
		{TopicName: topicNameExportSymbol, PageTitle: exportFrontmatterPageTitle},
		{TopicName: topicNameExport, PageTitle: exportMissingLinkPageTitle},
	} {
		if keys[i].TopicName != want.TopicName || keys[i].PageTitle != want.PageTitle {
			t.Errorf(
				"%d番目のWikiリンクが%qを指すことを期待したが%qを指していた",
				i+1,
				want.TopicName+"/"+want.PageTitle,
				keys[i].TopicName+"/"+keys[i].PageTitle,
			)
		}
	}

	// 4つ目のリンクは、どの走査も報告しないがソースには書かれている。
	fenced := strings.Split(exportWikilinkPageBody, codeFence)
	if len(fenced) != 3 {
		t.Fatalf("本文にコードフェンスが1組あることを期待したが%d個の区画に分かれた", len(fenced))
	}
	if !strings.Contains(fenced[1], "[["+exportPercentAndDotPageTitle+"]]") {
		t.Errorf("コードブロックの中に4つ目のWikiリンクが書かれていない: %q", fenced[1])
	}
}

func TestExportPageTitlesReachTheirArchiveNames(t *testing.T) {
	t.Parallel()

	// タイトルこそがそのページである。タイトルが持つものがアーカイブの与える名前を
	// 決めており、読みやすさのために書き換えられたタイトルは、もう起きない変換を見せる
	// ページを残す。名前はシード側の期待値として書き出さずexportfileに尋ねる。そうする
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
			t.Errorf("%qのファイル名が%qになることを期待したが%qだった", tt.title, tt.want, got)
		}
	}

	// トピック名も変換される。その半角の "*" だけが、あのトピックがああ綴られて
	// いる理由である。
	if got := exportfile.Name(topicNameExportSymbol, ""); got != "エクスポート＊記号" {
		t.Errorf("トピック名%qのディレクトリ名が%qになることを期待したが%qだった",
			topicNameExportSymbol, "エクスポート＊記号", got)
	}

	// 衝突する組は、1つのトピックのディレクトリのページと同じく、1つのDeduper
	// から名前を受け取る。1つずつ尋ねれば2つ目は連番の付かない名前で出てきてしまい、
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
			t.Errorf("%qのファイル名が%qになることを期待したが%qだった", tt.title, tt.want, got)
		}
	}
}

func TestExportPageTitlesStayWithinWhatTheAppAccepts(t *testing.T) {
	t.Parallel()

	// これらのタイトルは、ファイル名が持てないものを持つように書かれている。
	// そのため「エクスポートが変換するもの」と「アプリケーションが拒否するもの」の境目
	// こそが、これらのタイトルの歩く線になる。アプリケーションが弾くタイトルは画面から
	// 作れないため、それを持つページはどのメンバーにも作れない状態を見せることになる。
	// ここの規則はinternal/validatorのPageUpdateValidatorに対応している。
	titles := make([]string, 0, len(exportPageSpecs())+1)
	for _, spec := range exportPageSpecs() {
		titles = append(titles, spec.title)
	}
	titles = append(titles, exportMissingLinkPageTitle)

	for _, title := range titles {
		if got := utf8.RuneCountInString(title); got > pageTitleLengthLimit {
			t.Errorf("%qが%d文字以内であることを期待したが%d文字だった", title, pageTitleLengthLimit, got)
		}
		if strings.ContainsAny(title, `/\:`) {
			t.Errorf("%qにタイトルとして使えない文字が含まれている", title)
		}
		if strings.ContainsFunc(title, unicode.IsControl) {
			t.Errorf("%qに制御文字が含まれている", title)
		}
		if strings.HasPrefix(title, " ") || strings.HasSuffix(title, " ") {
			t.Errorf("%qの先頭または末尾にスペースがある", title)
		}
	}
}

// countSpecsInTopicは、仕様がtopicへ書き込むエクスポート確認用ページを数える。
func countSpecsInTopic(topics *seededTopics, topic *seededTopic) int {
	count := 0
	for _, spec := range exportPageSpecs() {
		if spec.topic(topics) == topic {
			count++
		}
	}

	return count
}

// topicNameForSpecは、仕様が書き込まれるトピックの名前を返す。トピックを持たない
// Wikiリンクは、この名前を補って読まれる。
func topicNameForSpec(spec exportPageSpec) string {
	export := &seededTopic{name: topicNameExport}
	exportSymbol := &seededTopic{name: topicNameExportSymbol}

	return spec.topic(&seededTopics{export: export, exportSymbol: exportSymbol}).name
}

func TestExportPageBodiesReachTheirArchiveForms(t *testing.T) {
	t.Parallel()

	// はじめからfrontmatterを持つ本文は、前に2つ目のブロックを書かれるのではなく、
	// 持っているブロックへキーを受け取る必要があり、書き手が置いたキーは残る必要がある。
	// YAMLとして読めなくなった本文も、下のページと同じくエクスポートはされる。そして
	// 失われるのは、まさにこの2ページの違いである。
	withFrontmatter := exportfile.WithFrontmatter(exportFrontmatterPageBody, exportFrontmatterPageTitle)
	if got := strings.Count(withFrontmatter, "\n---\n"); got != 1 {
		t.Errorf("%qのfrontmatterが1ブロックであることを期待したが区切りが%d箇所あった: %q",
			exportFrontmatterPageTitle, got+1, withFrontmatter)
	}
	for _, want := range []string{
		`wikino_title: "` + exportFrontmatterPageTitle + `"`,
		"tags:\n  - エクスポート\n",
	} {
		if !strings.Contains(withFrontmatter, want) {
			t.Errorf("%qのfrontmatterに%qが含まれていない: %q", exportFrontmatterPageTitle, want, withFrontmatter)
		}
	}

	// frontmatterに見えるだけの本文は、区切りをその位置に保ったまま、その前に
	// 自分のブロックを受け取る必要がある。
	frontmatterLike := exportfile.WithFrontmatter(exportFrontmatterLikePageBody, exportFrontmatterLikePageTitle)
	wantPrefix := "---\nwikino_title: \"" + exportFrontmatterLikePageTitle + "\"\n---\n\n"
	if !strings.HasPrefix(frontmatterLike, wantPrefix) {
		t.Errorf("%qが%qで始まることを期待したが%qだった", exportFrontmatterLikePageTitle, wantPrefix, frontmatterLike)
	}
	if !strings.HasSuffix(frontmatterLike, exportFrontmatterLikePageBody) {
		t.Errorf("%qの本文が原文のまま残ることを期待したが%qだった", exportFrontmatterLikePageTitle, frontmatterLike)
	}

	// 4つのWikiリンクのうち、アーカイブにページがある2つはそのパスへ書き換え
	// られ、残る2つはそのまま残る。一方はアーカイブが持たないページを名指しし、もう
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
			t.Errorf("書き換え後の%qに%qが含まれていない: %q", exportWikilinkPageTitle, want, rewritten)
		}
	}
}
