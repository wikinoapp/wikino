package seed

import (
	"context"
	"database/sql"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestCreatePage(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	spaces := buildSeedSpaces(t, tx, "seed-page")
	topics, err := generateTopics(ctx, tx, io.Discard, spaces)
	if err != nil {
		t.Fatalf("トピック生成に失敗: %v", err)
	}

	// The three link forms a body can carry: a page in the current topic, a
	// page in another topic, and a topic that does not exist.
	//
	// [Ja] 本文が持ちうる 3 種類のリンク形式。現在のトピックのページ、別トピックの
	// ページ、存在しないトピック。
	body := "# 見出し\n\n[[Link Target]] / [[" + topicNameHandbook + "/Cross Topic]] / [[Nowhere/Missing]]\n"

	page, err := newPageWriter(tx, spaces.wiki).createPage(ctx, createPageInput{
		topic:  topics.notes,
		author: spaces.wiki.member(roleOwner),
		title:  "Created Page",
		body:   body,
	})
	if err != nil {
		t.Fatalf("ページ作成に失敗: %v", err)
	}

	row := readPage(ctx, t, tx, spaces.wiki.id, page.id)
	if row.title != "Created Page" {
		t.Errorf("タイトルが %q であることを期待したが %q だった", "Created Page", row.title)
	}
	if row.body != body {
		t.Errorf("本文が渡したMarkdownと一致しない: %q", row.body)
	}
	if row.topicID != topics.notes.id {
		t.Errorf("トピックIDが「ノート」と一致しない: %s", row.topicID)
	}
	if !row.published {
		t.Error("ページが公開済みであることを期待したが未公開だった")
	}

	// The body is stored as the HTML the page detail screen serves, so the
	// Markdown must have been through the renderer and not merely copied.
	//
	// [Ja] 本文はページ詳細画面が配信する HTML として保存されるため、Markdown が
	// そのまま複写されたのではなく、レンダラーを通っている必要がある。
	if !strings.Contains(row.bodyHTML, "<h1") {
		t.Errorf("body_htmlに見出しのHTMLが含まれていない: %q", row.bodyHTML)
	}

	// A resolved wiki link becomes an href built from the space identifier and
	// the linked page's number. An unresolved one stays as it was written.
	//
	// [Ja] 解決された Wiki リンクは、スペース識別子とリンク先のページ番号から
	// 組み立てた href になる。解決されなかったものは書いたままの姿で残る。
	target := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.notes.id, "Link Target")
	crossTopic := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.handbook.id, "Cross Topic")

	assertContains(t, row.bodyHTML, hrefOf(spaces.wiki, target.number))
	assertContains(t, row.bodyHTML, hrefOf(spaces.wiki, crossTopic.number))
	assertContains(t, row.bodyHTML, "[[Nowhere/Missing]]")

	// The pages a wiki link points at are created unpublished, the same as
	// when a link is followed from the screen.
	//
	// [Ja] Wiki リンクの指す先のページは未公開で作成される。画面からリンクを
	// 辿ったときと同じ。
	if target.published || crossTopic.published {
		t.Error("リンク先ページが未公開であることを期待したが公開済みだった")
	}

	// Backlinks are read from linked_page_ids, so the column has to name every
	// page that resolved, and only those.
	//
	// [Ja] バックリンクは linked_page_ids から引かれるため、この列は解決された
	// ページをすべて、かつそれだけを名指しする必要がある。
	assertLinkedPageIDs(t, row.linkedPageIDs, []model.PageID{target.id, crossTopic.id})

	// The page is numbered after the pages its body links to. Taking the
	// number first would hand this page a number the linked pages then take.
	//
	// [Ja] ページには、その本文がリンクする先のページより後の番号が付く。先に
	// 番号を取ると、リンク先があとからその番号を取ってしまうため。
	if page.number <= target.number || page.number <= crossTopic.number {
		t.Errorf("ページ番号 %d がリンク先の番号 (%d, %d) より後であることを期待した",
			page.number, target.number, crossTopic.number)
	}

	assertPageRevision(ctx, t, tx, spaces.wiki, spaces.wiki.member(roleOwner), page.id, row)

	// Every page the seed touched gets an editor entry, which is what puts the
	// page on its author's home screen.
	//
	// [Ja] シードが触れたすべてのページに編集者エントリが付く。これがページを
	// 書き手のホーム画面へ載せるもの。
	for _, id := range []model.PageID{page.id, target.id, crossTopic.id} {
		assertPageEditor(ctx, t, tx, spaces.wiki, spaces.wiki.member(roleOwner), id)
	}
}

func TestCreatePageRejectsTopicFromAnotherSpace(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	spaces := buildSeedSpaces(t, tx, "seed-cross-space")
	topics, err := generateTopics(ctx, tx, io.Discard, spaces)
	if err != nil {
		t.Fatalf("トピック生成に失敗: %v", err)
	}

	// pages accepts any (space_id, topic_id) pair the database is given, so the
	// mismatch has to be caught before the INSERT rather than by a constraint.
	//
	// [Ja] pages は渡された (space_id, topic_id) の組をそのまま受け入れるため、
	// 食い違いは制約ではなく INSERT の手前で捕まえる必要がある。
	page, err := newPageWriter(tx, spaces.solo).createPage(ctx, createPageInput{
		topic:  topics.notes,
		author: spaces.solo.member(roleOwner),
		title:  "Cross Space Page",
		body:   "別のスペースのトピックを渡した本文です。\n",
	})
	if err == nil {
		t.Fatalf("別のスペースのトピックを渡したページ作成が失敗することを期待したが成功した: %v", page)
	}
	if !strings.Contains(err.Error(), topics.notes.name) {
		t.Errorf("エラーが食い違ったトピック名を示すことを期待したが %q だった", err)
	}
}

func TestCreatePageDoesNotAddEditorToExistingLinkTarget(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	spaces := buildSeedSpaces(t, tx, "seed-existing-link")
	topics, err := generateTopics(ctx, tx, io.Discard, spaces)
	if err != nil {
		t.Fatalf("トピック生成に失敗: %v", err)
	}

	writer := newPageWriter(tx, spaces.wiki)
	target, err := writer.createPage(ctx, createPageInput{
		topic:  topics.notes,
		author: spaces.wiki.member(roleOwner),
		title:  "Existing Target",
		body:   "ownerが作成したページです。\n",
	})
	if err != nil {
		t.Fatalf("リンク先ページの作成に失敗: %v", err)
	}
	assertPageEditor(ctx, t, tx, spaces.wiki, spaces.wiki.member(roleOwner), target.id)

	if _, err := writer.createPage(ctx, createPageInput{
		topic:  topics.notes,
		author: spaces.wiki.member(roleCollaborator),
		title:  "Linking Page",
		body:   "[[Existing Target]]\n",
	}); err != nil {
		t.Fatalf("リンク元ページの作成に失敗: %v", err)
	}

	// Resolving a link to an existing page does not count as editing that page.
	// Only the member who actually created the target should remain its editor,
	// so the check is on both members: roleOwner still there, roleCollaborator
	// never added.
	//
	// [Ja] 既存ページへのリンク解決は、そのページの編集とは見なさない。リンク先の
	// 編集者には、実際にそのページを作成したメンバーだけが残る必要があるため、
	// 両方のメンバーを確認する。roleOwner が残っていること、roleCollaborator が
	// 増えていないこと。
	assertPageEditor(ctx, t, tx, spaces.wiki, spaces.wiki.member(roleOwner), target.id)

	var count int
	err = tx.QueryRowContext(
		ctx,
		`SELECT count(*) FROM page_editors WHERE page_id = $1 AND space_id = $2 AND space_member_id = $3`,
		string(target.id), string(spaces.wiki.id), string(spaces.wiki.member(roleCollaborator).id),
	).Scan(&count)
	if err != nil {
		t.Fatalf("既存リンク先ページの編集者の取得に失敗: %v", err)
	}
	if count != 0 {
		t.Errorf("既存リンク先ページにリンク元の書き手が追加されていないことを期待したが %d 件だった", count)
	}
}

func TestCreatePageWithoutLinksStoresEmptyLinkedPageIDs(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	spaces := buildSeedSpaces(t, tx, "seed-page-without-links")
	topics, err := generateTopics(ctx, tx, io.Discard, spaces)
	if err != nil {
		t.Fatalf("トピック生成に失敗: %v", err)
	}

	page, err := newPageWriter(tx, spaces.wiki).createPage(ctx, createPageInput{
		topic:  topics.notes,
		author: spaces.wiki.member(roleOwner),
		title:  "Page Without Links",
		body:   "Wikiリンクを含まない本文です。\n",
	})
	if err != nil {
		t.Fatalf("リンク無しページの作成に失敗: %v", err)
	}

	// Scanning cardinality into an int also proves the stored array is not NULL.
	//
	// [Ja] cardinality を int へ読み取れることでも、保存された配列が NULL ではない
	// ことを確認する。
	var linkedPageCount int
	err = tx.QueryRowContext(
		ctx,
		`SELECT cardinality(linked_page_ids) FROM pages WHERE id = $1 AND space_id = $2`,
		string(page.id), string(spaces.wiki.id),
	).Scan(&linkedPageCount)
	if err != nil {
		t.Fatalf("リンク無しページのlinked_page_idsの取得に失敗: %v", err)
	}
	if linkedPageCount != 0 {
		t.Errorf("linked_page_idsが空であることを期待したが %d 件だった", linkedPageCount)
	}
}

func TestGenerateMarkdownGuide(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	spaces := buildSeedSpaces(t, tx, "seed-md-guide")
	topics, err := generateTopics(ctx, tx, io.Discard, spaces)
	if err != nil {
		t.Fatalf("トピック生成に失敗: %v", err)
	}

	if err := generateMarkdownGuide(ctx, tx, io.Discard, spaces, topics); err != nil {
		t.Fatalf("Markdown記法紹介ページの生成に失敗: %v", err)
	}

	guide := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.notes.id, "Markdown 記法")
	if !guide.published {
		t.Error("Markdown記法紹介ページが公開済みであることを期待したが未公開だった")
	}

	row := readPage(ctx, t, tx, spaces.wiki.id, guide.id)

	// The page exists to be looked at, so what matters is that the notations
	// it lists reach the browser as the elements they are meant to become.
	//
	// [Ja] このページは目視のために存在するため、重要なのは、並べた記法が
	// 意図した要素としてブラウザまで届くこと。
	for _, tt := range []struct {
		notation string
		want     string
	}{
		{notation: "見出し", want: "<h1"},
		{notation: "テーブル", want: "<table"},
		{notation: "引用", want: "<blockquote"},
		{notation: "タスクリスト", want: `type="checkbox"`},
		{notation: "コードブロック", want: "<pre"},
		{notation: "水平線", want: "<hr"},
	} {
		if !strings.Contains(row.bodyHTML, tt.want) {
			t.Errorf("%sが %s としてレンダリングされていない", tt.notation, tt.want)
		}
	}

	// The body links to a page in its own topic and to one in another, and the
	// seed creates both. Without them the link list and backlink screens have
	// nothing to show for this page.
	//
	// [Ja] 本文は同じトピックのページと別トピックのページへリンクしており、シードは
	// その両方を作成する。これが無いと、このページについてリンク一覧・バックリンク
	// 一覧の画面に出せるものが無くなる。
	sameTopic := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.notes.id, "Wiki リンクの例")
	otherTopic := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.handbook.id, "Wiki リンクの例")
	assertLinkedPageIDs(t, row.linkedPageIDs, []model.PageID{sameTopic.id, otherTopic.id})

	assertContains(t, row.bodyHTML, hrefOf(spaces.wiki, sameTopic.number))
	assertContains(t, row.bodyHTML, hrefOf(spaces.wiki, otherTopic.number))

	// The guide deliberately links into a topic that does not exist, so that
	// the unresolved form can be seen on the screen as well. The count is what
	// is checked: wiki links are never replaced inside <pre>, so the one in the
	// code fence survives whatever happens, and only counting both shows that
	// the one in the prose stayed as it was written too.
	//
	// [Ja] このページは存在しないトピックへのリンクを意図的に含んでおり、未解決の
	// 見た目も画面で確認できるようにしている。件数で確認するのは、Wiki リンクが
	// <pre> の中では置換されず、コードフェンス内の 1 件は何があっても残るため。
	// 本文中の 1 件も書いたままであることは、両方を数えて初めて確認できる。
	if got := strings.Count(row.bodyHTML, "[[存在しないトピック/"); got != 2 {
		t.Errorf("未解決のWikiリンクが 2 箇所残ることを期待したが %d 箇所だった", got)
	}

	// The in-page link section demonstrates an anchor, and heading ids are
	// numbered by position because the guide's headings carry no ASCII letters
	// for the id to be built from. Checking that the anchor names an id the
	// page actually has keeps the demonstration from silently turning into a
	// dead link when a heading is added above it.
	//
	// [Ja] ページ内リンクの節はアンカーを例示している。このページの見出しには
	// ID の元になる英字が無く、ID は出現順から採番される。アンカーがページに実在する
	// ID を指していることを確認しておくと、上に見出しが増えたときに、例示が黙って
	// 死んだリンクへ変わるのを防げる。
	assertContains(t, row.bodyHTML, `<a href="#heading"`)
	assertContains(t, row.bodyHTML, `<h2 id="heading">`)
}

func TestMarkdownGuideBody(t *testing.T) {
	t.Parallel()

	if markdownGuideBody == "" {
		t.Fatal("埋め込まれたMarkdown記法紹介ページの本文が空")
	}

	// The guide is a runtime asset that teaches literal Markdown syntax. Keep
	// formatters from normalizing distinct spellings into the same notation.
	//
	// [Ja] このガイドは Markdown 構文そのものを教える実行時アセット。フォーマッタが
	// 異なる書き方を同じ記法へ正規化しないよう、文字どおりの構文を守る。
	for _, tt := range []struct {
		name string
		want string
	}{
		{name: "強調の区切り文字", want: "*イタリック体* または _イタリック体_\n**太字** または __太字__\n***太字イタリック*** または ___太字イタリック___"},
		{name: "順序なしリストの記号", want: "* 別の記号でも可能\n+ こちらも使用可能"},
		{name: "水平線の記号", want: "```markdown\n---\n***\n___\n```"},
		{name: "エスケープの改行", want: "\\# これは見出しになりません\n\\* これはリストになりません"},
		{name: "折りたたみ内のリスト", want: "ここに折りたたまれる内容を記述\n- リスト項目も使えます\n- **Markdown**も使用可能"},
	} {
		if !strings.Contains(markdownGuideBody, tt.want) {
			t.Errorf("本文に%sの原文が含まれていない: %q", tt.name, tt.want)
		}
	}

	// The image section of the source document was dropped: the seed creates
	// no attachments, so an image reference would render as a broken image on
	// the one page whose purpose is to show notations rendering correctly.
	//
	// [Ja] 元ドキュメントの画像の節は落としている。シードは添付ファイルを作らない
	// ため、画像の参照は壊れた画像として表示されてしまう。記法が正しく表示される
	// ことを示すためのページで、それは起こしたくない。
	if strings.Contains(markdownGuideBody, "/attachments/") {
		t.Error("本文が添付ファイルを参照している。シードは添付ファイルを作らないため表示が壊れる")
	}
}

// pageRow is what the seed wrote to the pages table.
//
// [Ja] pageRow は、シードが pages テーブルへ書いた内容。
type pageRow struct {
	title         string
	body          string
	bodyHTML      string
	topicID       model.TopicID
	published     bool
	linkedPageIDs []model.PageID
}

// readPage reads back the stored page.
//
// [Ja] readPage は保存されたページを読み戻す。
func readPage(ctx context.Context, t *testing.T, tx *sql.Tx, spaceID model.SpaceID, pageID model.PageID) pageRow {
	t.Helper()

	var (
		row     pageRow
		topicID string
		linked  []string
	)
	err := tx.QueryRowContext(
		ctx,
		`SELECT title, body, body_html, topic_id, published_at IS NOT NULL, linked_page_ids
         FROM pages WHERE id = $1 AND space_id = $2`,
		string(pageID), string(spaceID),
	).Scan(&row.title, &row.body, &row.bodyHTML, &topicID, &row.published, pq.Array(&linked))
	if err != nil {
		t.Fatalf("ページの取得に失敗: %v", err)
	}

	row.topicID = model.TopicID(topicID)
	for _, id := range linked {
		row.linkedPageIDs = append(row.linkedPageIDs, model.PageID(id))
	}

	return row
}

// foundPage is a page located by its title.
//
// [Ja] foundPage はタイトルで見つけたページ。
type foundPage struct {
	id        model.PageID
	number    model.PageNumber
	published bool
}

// findPageByTitle locates a page by the title a wiki link named it with.
//
// [Ja] findPageByTitle は、Wiki リンクが名指ししたタイトルでページを見つける。
func findPageByTitle(
	ctx context.Context,
	t *testing.T,
	tx *sql.Tx,
	spaceID model.SpaceID,
	topicID model.TopicID,
	title string,
) foundPage {
	t.Helper()

	var (
		id     string
		number int32
		found  foundPage
	)
	err := tx.QueryRowContext(
		ctx,
		`SELECT id, number, published_at IS NOT NULL
         FROM pages WHERE space_id = $1 AND topic_id = $2 AND title = $3`,
		string(spaceID), string(topicID), title,
	).Scan(&id, &number, &found.published)
	if err != nil {
		t.Fatalf("ページ %s の取得に失敗: %v", title, err)
	}

	found.id = model.PageID(id)
	found.number = model.PageNumber(number)

	return found
}

// hrefOf builds the link a resolved wiki link is expected to become.
//
// [Ja] hrefOf は、解決された Wiki リンクがなるはずのリンクを組み立てる。
func hrefOf(space *seededSpace, number model.PageNumber) string {
	return `<a href="/s/` + string(space.identifier) + `/pages/` + strconv.Itoa(int(number)) + `">`
}

// assertContains reports a missing substring together with the HTML it was
// looked for in, which is otherwise hard to reconstruct from a failure.
//
// [Ja] assertContains は、見つからなかった部分文字列を、それを探した HTML と
// 併せて報告する。失敗した内容から HTML を復元するのは難しいため。
func assertContains(t *testing.T, html string, want string) {
	t.Helper()

	if !strings.Contains(html, want) {
		t.Errorf("body_htmlに %q が含まれていない: %q", want, html)
	}
}

// assertLinkedPageIDs checks the stored link targets against the intended set,
// ignoring order.
//
// [Ja] assertLinkedPageIDs は、保存されたリンク先を意図した集合と、順序を無視して
// 比較する。
func assertLinkedPageIDs(t *testing.T, stored []model.PageID, want []model.PageID) {
	t.Helper()

	got := make(map[model.PageID]bool, len(stored))
	for _, id := range stored {
		got[id] = true
	}
	if len(got) != len(want) {
		t.Errorf("linked_page_idsが %d 件であることを期待したが %d 件だった (%v)", len(want), len(got), stored)
	}
	for _, id := range want {
		if !got[id] {
			t.Errorf("linked_page_idsにページ %s が含まれていない", id)
		}
	}
}

// assertPageRevision checks that publishing left a revision holding the same
// snapshot as the page, attributed to the given author.
//
// [Ja] assertPageRevision は、公開がページと同じスナップショットを持つリビジョンを
// 残し、渡した author に紐づいていることを確認する。
func assertPageRevision(
	ctx context.Context,
	t *testing.T,
	tx *sql.Tx,
	space *seededSpace,
	author *seededSpaceMember,
	pageID model.PageID,
	page pageRow,
) {
	t.Helper()

	var (
		count         int
		title         string
		body          string
		bodyHTML      string
		spaceMemberID string
	)
	err := tx.QueryRowContext(
		ctx,
		`SELECT count(*) OVER (), title, body, body_html, space_member_id
         FROM page_revisions WHERE page_id = $1 AND space_id = $2`,
		string(pageID), string(space.id),
	).Scan(&count, &title, &body, &bodyHTML, &spaceMemberID)
	if err != nil {
		t.Fatalf("ページリビジョンの取得に失敗: %v", err)
	}

	if count != 1 {
		t.Errorf("ページリビジョンが 1 件であることを期待したが %d 件だった", count)
	}
	if title != page.title || body != page.body || bodyHTML != page.bodyHTML {
		t.Error("ページリビジョンの内容がページと一致しない")
	}
	if model.SpaceMemberID(spaceMemberID) != author.id {
		t.Errorf("ページリビジョンが書き手に紐づいていない: %s", spaceMemberID)
	}
}

// assertPageEditor checks that the page is attributed to the given member.
//
// [Ja] assertPageEditor は、ページが渡した member に紐づいていることを確認する。
func assertPageEditor(
	ctx context.Context,
	t *testing.T,
	tx *sql.Tx,
	space *seededSpace,
	member *seededSpaceMember,
	pageID model.PageID,
) {
	t.Helper()

	var count int
	err := tx.QueryRowContext(
		ctx,
		`SELECT count(*) FROM page_editors WHERE page_id = $1 AND space_id = $2 AND space_member_id = $3`,
		string(pageID), string(space.id), string(member.id),
	).Scan(&count)
	if err != nil {
		t.Fatalf("ページ編集者の取得に失敗: %v", err)
	}
	if count != 1 {
		t.Errorf("ページ %s の編集者が 1 件であることを期待したが %d 件だった", pageID, count)
	}
}
