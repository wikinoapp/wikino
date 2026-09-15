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

	// 本文が持ちうる3種類のリンク形式。現在のトピックのページ、別トピックの
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
		t.Errorf("タイトルが%qであることを期待したが%qだった", "Created Page", row.title)
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

	// 本文はページ詳細画面が配信するHTMLとして保存されるため、Markdownが
	// そのまま複写されたのではなく、レンダラーを通っている必要がある。
	if !strings.Contains(row.bodyHTML, "<h1") {
		t.Errorf("body_htmlに見出しのHTMLが含まれていない: %q", row.bodyHTML)
	}

	// 解決されたWikiリンクは、スペース識別子とリンク先のページ番号から
	// 組み立てたhrefになる。解決されなかったものは書いたままの姿で残る。
	target := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.notes.id, "Link Target")
	crossTopic := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.handbook.id, "Cross Topic")

	assertContains(t, row.bodyHTML, hrefOf(spaces.wiki, target.number))
	assertContains(t, row.bodyHTML, hrefOf(spaces.wiki, crossTopic.number))
	assertContains(t, row.bodyHTML, "[[Nowhere/Missing]]")

	// Wikiリンクの指す先のページは未公開で作成される。画面からリンクを
	// 辿ったときと同じ。
	if target.published || crossTopic.published {
		t.Error("リンク先ページが未公開であることを期待したが公開済みだった")
	}

	// バックリンクはlinked_page_idsから引かれるため、この列は解決された
	// ページをすべて、かつそれだけを名指しする必要がある。
	assertLinkedPageIDs(t, row.linkedPageIDs, []model.PageID{target.id, crossTopic.id})

	// ページには、その本文がリンクする先のページより後の番号が付く。先に
	// 番号を取ると、リンク先があとからその番号を取ってしまうため。
	if page.number <= target.number || page.number <= crossTopic.number {
		t.Errorf("ページ番号%dがリンク先の番号 (%d, %d) より後であることを期待した",
			page.number, target.number, crossTopic.number)
	}

	assertPageRevision(ctx, t, tx, spaces.wiki, spaces.wiki.member(roleOwner), page.id, row)

	// シードが触れたすべてのページに編集者エントリが付く。これがページを
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

	// pagesは渡された (space_id, topic_id) の組をそのまま受け入れるため、
	// 食い違いは制約ではなくINSERTの手前で捕まえる必要がある。
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
		t.Errorf("エラーが食い違ったトピック名を示すことを期待したが%qだった", err)
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

	// 既存ページへのリンク解決は、そのページの編集とは見なさない。リンク先の
	// 編集者には、実際にそのページを作成したメンバーだけが残る必要があるため、
	// 両方のメンバーを確認する。roleOwnerが残っていること、roleCollaboratorが
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
		t.Errorf("既存リンク先ページにリンク元の書き手が追加されていないことを期待したが%d件だった", count)
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

	// cardinalityをintへ読み取れることでも、保存された配列がNULLではない
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
		t.Errorf("linked_page_idsが空であることを期待したが%d件だった", linkedPageCount)
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

	// このページは目視のために存在するため、重要なのは、並べた記法が
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
			t.Errorf("%sが%sとしてレンダリングされていない", tt.notation, tt.want)
		}
	}

	// レンダラーはハードラップを有効にしているため、段落内の改行は <br> になる。
	// このページが見せるのは記法であって書き手が選んだ折り返しではなく、シードの他の
	// ページはいずれも <br> なしで描画される。ここに <br> が出たなら、段落がまた複数行に
	// またがって書かれたということになる。
	if got := strings.Count(row.bodyHTML, "<br"); got != 0 {
		t.Errorf("段落内の改行が%d箇所 <br> として描画されている", got)
	}

	// 本文は同じトピックのページと別トピックのページへリンクしており、シードは
	// その両方を作成する。これが無いと、このページについてリンク一覧・バックリンク
	// 一覧の画面に出せるものが無くなる。
	sameTopic := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.notes.id, "Wiki リンクの例")
	otherTopic := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.handbook.id, "Wiki リンクの例")
	assertLinkedPageIDs(t, row.linkedPageIDs, []model.PageID{sameTopic.id, otherTopic.id})

	assertContains(t, row.bodyHTML, hrefOf(spaces.wiki, sameTopic.number))
	assertContains(t, row.bodyHTML, hrefOf(spaces.wiki, otherTopic.number))

	// このページは存在しないトピックへのリンクを意図的に含んでおり、未解決の
	// 見た目も画面で確認できるようにしている。件数で確認するのは、Wikiリンクが
	// <pre> の中では置換されず、コードフェンス内の1件は何があっても残るため。
	// 本文中の1件も書いたままであることは、両方を数えて初めて確認できる。
	if got := strings.Count(row.bodyHTML, "[[存在しないトピック/"); got != 2 {
		t.Errorf("未解決のWikiリンクが2箇所残ることを期待したが%d箇所だった", got)
	}

	// ページ内リンクの節はアンカーを例示している。このページの見出しには
	// IDの元になる英字が無く、IDは出現順から採番される。アンカーがページに実在する
	// IDを指していることを確認しておくと、上に見出しが増えたときに、例示が黙って
	// 死んだリンクへ変わるのを防げる。
	assertContains(t, row.bodyHTML, `<a href="#heading"`)
	assertContains(t, row.bodyHTML, `<h2 id="heading">`)
}

func TestMarkdownGuideBody(t *testing.T) {
	t.Parallel()

	if markdownGuideBody == "" {
		t.Fatal("埋め込まれたMarkdown記法紹介ページの本文が空")
	}

	// このガイドはMarkdown構文そのものを教える実行時アセット。フォーマッタが
	// 異なる書き方を同じ記法へ正規化しないよう、文字どおりの構文を守る。
	for _, tt := range []struct {
		name string
		want string
	}{
		{name: "強調の区切り文字", want: "*イタリック体* または _イタリック体_\n\n**太字** または __太字__\n\n***太字イタリック*** または ___太字イタリック___"},
		{name: "順序なしリストの記号", want: "* 別の記号でも可能\n+ こちらも使用可能"},
		{name: "水平線の記号", want: "```markdown\n---\n***\n___\n```"},
		{name: "エスケープの改行", want: "\\# これは見出しになりません\n\n\\* これはリストになりません"},
		{name: "折りたたみ内のリスト", want: "ここに折りたたまれる内容を記述\n\n- リスト項目も使えます\n- **Markdown**も使用可能"},
	} {
		if !strings.Contains(markdownGuideBody, tt.want) {
			t.Errorf("本文に%sの原文が含まれていない: %q", tt.name, tt.want)
		}
	}

	// 元ドキュメントの画像の節は落としている。シードは添付ファイルを作らない
	// ため、画像の参照は壊れた画像として表示されてしまう。記法が正しく表示される
	// ことを示すためのページで、それは起こしたくない。
	if strings.Contains(markdownGuideBody, "/attachments/") {
		t.Error("本文が添付ファイルを参照している。シードは添付ファイルを作らないため表示が壊れる")
	}
}

// pageRowは、シードがpagesテーブルへ書いた内容。
type pageRow struct {
	title         string
	body          string
	bodyHTML      string
	topicID       model.TopicID
	published     bool
	linkedPageIDs []model.PageID
}

// readPageは保存されたページを読み戻す。
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

// foundPageはタイトルで見つけたページ。
type foundPage struct {
	id        model.PageID
	number    model.PageNumber
	published bool
}

// findPageByTitleは、Wikiリンクが名指ししたタイトルでページを見つける。
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
		t.Fatalf("ページ%sの取得に失敗: %v", title, err)
	}

	found.id = model.PageID(id)
	found.number = model.PageNumber(number)

	return found
}

// hrefOfは、解決されたWikiリンクがなるはずのリンクを組み立てる。
func hrefOf(space *seededSpace, number model.PageNumber) string {
	return `<a href="/s/` + string(space.identifier) + `/pages/` + strconv.Itoa(int(number)) + `">`
}

// assertContainsは、見つからなかった部分文字列を、それを探したHTMLと
// 併せて報告する。失敗した内容からHTMLを復元するのは難しいため。
func assertContains(t *testing.T, html string, want string) {
	t.Helper()

	if !strings.Contains(html, want) {
		t.Errorf("body_htmlに%qが含まれていない: %q", want, html)
	}
}

// assertLinkedPageIDsは、保存されたリンク先を意図した集合と、順序を無視して
// 比較する。
func assertLinkedPageIDs(t *testing.T, stored []model.PageID, want []model.PageID) {
	t.Helper()

	got := make(map[model.PageID]bool, len(stored))
	for _, id := range stored {
		got[id] = true
	}
	if len(got) != len(want) {
		t.Errorf("linked_page_idsが%d件であることを期待したが%d件だった (%v)", len(want), len(got), stored)
	}
	for _, id := range want {
		if !got[id] {
			t.Errorf("linked_page_idsにページ%sが含まれていない", id)
		}
	}
}

// assertPageRevisionは、公開がページと同じスナップショットを持つリビジョンを
// 残し、渡したauthorに紐づいていることを確認する。
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
		t.Errorf("ページリビジョンが1件であることを期待したが%d件だった", count)
	}
	if title != page.title || body != page.body || bodyHTML != page.bodyHTML {
		t.Error("ページリビジョンの内容がページと一致しない")
	}
	if model.SpaceMemberID(spaceMemberID) != author.id {
		t.Errorf("ページリビジョンが書き手に紐づいていない: %s", spaceMemberID)
	}
}

// assertPageEditorは、ページが渡したmemberに紐づいていることを確認する。
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
		t.Errorf("ページ%sの編集者が1件であることを期待したが%d件だった", pageID, count)
	}
}
