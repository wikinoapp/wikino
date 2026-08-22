package seed

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"regexp"
	"testing"

	"github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

func TestGenerateLinkHub(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	spaces := buildSeedSpaces(t, tx, "seed-link-hub")
	topics, err := generateTopics(ctx, tx, io.Discard, spaces)
	if err != nil {
		t.Fatalf("トピック生成に失敗: %v", err)
	}

	// Small amounts, but enough of them for both accounts to take a turn in
	// each group and for the link targets to have an order worth checking.
	//
	// [Ja] 件数は小さくしつつ、各組で両方のアカウントが 1 回ずつ担当し、リンク先の
	// 並び順を確認できる程度にする。
	amt := amounts{linkHubTargets: 4, linkHubBacklinks: 3, nestedBacklinks: 2}

	if err := generateLinkHub(ctx, tx, io.Discard, amt, spaces, topics); err != nil {
		t.Fatalf("リンク集中ページの生成に失敗: %v", err)
	}

	hub := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.notes.id, linkHubTitle)
	if !hub.published {
		t.Errorf("%sが公開済みであることを期待したが未公開だった", linkHubTitle)
	}

	// The link list is what the hub exists for, so every target the body names
	// has to have become a page and to be recorded on the hub's row: the list
	// is read from linked_page_ids.
	//
	// [Ja] リンク一覧こそがハブの存在理由であるため、本文が名指しするリンク先は
	// すべてページになり、ハブの行に記録されている必要がある。一覧は
	// linked_page_ids から引かれるため。
	targetIDs := make([]model.PageID, 0, amt.linkHubTargets)
	for number := 1; number <= amt.linkHubTargets; number++ {
		target := findPageByTitle(
			ctx, t, tx, spaces.wiki.id, topics.notes.id, fmt.Sprintf(linkTargetTitleFormat, number),
		)

		// The targets are created by the links pointing at them, so they stay
		// unpublished. That also keeps them out of the space and topic
		// listings, whose counts are decided elsewhere.
		//
		// [Ja] リンク先は、そこを指すリンクによって作成されるため未公開のままに
		// なる。これにより、件数を別の場所で決めているスペース・トピックの一覧にも
		// 入らない。
		if target.published {
			t.Errorf("リンク先 %d が未公開であることを期待したが公開済みだった", number)
		}
		targetIDs = append(targetIDs, target.id)
	}
	assertLinkedPageIDs(t, readPage(ctx, t, tx, spaces.wiki.id, hub.id).linkedPageIDs, targetIDs)

	// The nested backlinks are only reachable without paging the link list if
	// they hang on the target that list shows first. That ordering comes from
	// the database, not from the seed, so it is checked rather than assumed.
	//
	// [Ja] ネストしたバックリンクは、リンク一覧が最初に見せるリンク先に付いている
	// ときだけ、リンク一覧をページ送りせずに辿り着ける。この並び順を決めるのは
	// シードではなくデータベースであるため、前提にせず確認する。
	nestedTarget := fmt.Sprintf(linkTargetTitleFormat, amt.linkHubTargets)
	if got := firstLinkedPageTitle(ctx, t, tx, spaces.wiki.id, targetIDs); got != nestedTarget {
		t.Errorf("リンク一覧の先頭が %q であることを期待したが %q だった", nestedTarget, got)
	}

	// Each group of source pages feeds one listing, so what matters is how many
	// pages link to the hub and how many link to the target the nested list
	// hangs under.
	//
	// [Ja] リンク元ページの各組はそれぞれ 1 つの一覧を養うため、重要なのはハブへ
	// リンクするページ数と、ネストした一覧が付くリンク先へリンクするページ数。
	if got := countPagesLinkingTo(ctx, t, tx, spaces.wiki.id, hub.id); got != amt.linkHubBacklinks {
		t.Errorf("%sへのバックリンクが %d 件であることを期待したが %d 件だった",
			linkHubTitle, amt.linkHubBacklinks, got)
	}
	// The hub links to that target as well, so one more page links to it than
	// the nested list ever shows: the list is rendered on the hub's own screen,
	// and a backlink list leaves out the page it is shown on. The count that
	// was chosen to leave a partial last page is the one without the hub.
	//
	// [Ja] ハブもそのリンク先へリンクしているため、そこへリンクするページは、
	// ネストした一覧が見せる件数より 1 件多くなる。この一覧はハブ自身の画面に
	// 描画されるもので、バックリンク一覧は、それが描画されているページを除くため。
	// 端数の最終ページを作るために選んだ件数は、ハブを含まないほうの件数。
	nestedTargetID := targetIDs[len(targetIDs)-1]
	wantNestedLinks := amt.nestedBacklinks + 1
	if got := countPagesLinkingTo(ctx, t, tx, spaces.wiki.id, nestedTargetID); got != wantNestedLinks {
		t.Errorf("%sへのバックリンクが %d 件であることを期待したが %d 件だった", nestedTarget, wantNestedLinks, got)
	}

	// The source pages carry the seed's own text, so they have to be published:
	// an unpublished page still backlinks, but opening one from the listing
	// would lead to a page that reads as never written.
	//
	// [Ja] リンク元ページはシードが書いた本文を持つため、公開済みである必要がある。
	// 未公開でもバックリンクは張られるが、一覧から開くと、一度も書かれていない
	// ページとして表示されてしまう。
	for _, tt := range []struct {
		title  string
		editor *seededSpaceMember
	}{
		{title: fmt.Sprintf(hubBacklinkTitleFormat, 1), editor: spaces.wiki.member(roleOwner)},
		{title: fmt.Sprintf(hubBacklinkTitleFormat, 2), editor: spaces.wiki.member(roleCollaborator)},
		{title: fmt.Sprintf(nestedBacklinkTitleFormat, 1), editor: spaces.wiki.member(roleOwner)},
		{title: fmt.Sprintf(nestedBacklinkTitleFormat, 2), editor: spaces.wiki.member(roleCollaborator)},
	} {
		page := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.notes.id, tt.title)
		if !page.published {
			t.Errorf("%sが公開済みであることを期待したが未公開だった", tt.title)
		}
		assertPageEditor(ctx, t, tx, spaces.wiki, tt.editor, page.id)
	}

	// Nothing else was created along the way. A stray page would mean a body
	// linked somewhere it was not meant to, and the listing counts the seed is
	// built around would no longer hold.
	//
	// [Ja] 途中で他のページは作られていない。余分なページがあれば、本文が意図
	// しない先へリンクしたということであり、シードが基準にしている一覧の件数が
	// 成り立たなくなる。
	wantTotal := 1 + amt.linkHubTargets + amt.linkHubBacklinks + amt.nestedBacklinks
	if got := countPagesInSpace(ctx, t, tx, spaces.wiki.id); got != wantTotal {
		t.Errorf("スペース全体のページが %d 件であることを期待したが %d 件だった", wantTotal, got)
	}
}

func TestLinkHubBodiesLinkOnlyWhereIntended(t *testing.T) {
	t.Parallel()

	// Every link in these bodies creates or claims a page, and the listings are
	// checked against exact counts. A link written where none was meant would
	// move a count without failing anything at generation time, so the bodies
	// are read for what they link to before they are ever rendered.
	//
	// [Ja] これらの本文のリンクはいずれもページを作るか既存のページを掴むもので、
	// 一覧は正確な件数で確認している。意図していない場所に書かれたリンクは、生成
	// 時には何も失敗させずに件数だけを動かすため、本文がどこへリンクするのかを
	// レンダリングより前に読んで確認する。
	const targets = 3

	hubLinks := markup.ScanWikilinks(linkHubBody(targets), topicNameNotes)
	if len(hubLinks) != targets {
		t.Fatalf("ハブの本文のWikiリンクが %d 件であることを期待したが %d 件だった", targets, len(hubLinks))
	}
	for i, link := range hubLinks {
		want := fmt.Sprintf(linkTargetTitleFormat, i+1)
		if link.PageTitle != want || link.TopicName != topicNameNotes {
			t.Errorf("ハブの %d 番目のリンクが %s/%s であることを期待したが %s/%s だった",
				i+1, topicNameNotes, want, link.TopicName, link.PageTitle)
		}
	}

	nestedTarget := fmt.Sprintf(linkTargetTitleFormat, targets)

	for _, tt := range []struct {
		name string
		body string
		want string
	}{
		{
			name: "ハブへのリンク元",
			body: hubBacklinkBody(fmt.Sprintf(hubBacklinkTitleFormat, 1)),
			want: linkHubTitle,
		},
		{
			// This one names linkHubTitle in its prose as well. Only the target
			// may be a link: a link to the hub would add this page to the hub's
			// backlink list, which is counted.
			//
			// [Ja] こちらは本文で「リンクハブ」にも言及している。リンクにしてよいのは
			// リンク先だけ。ハブへのリンクにすると、このページが件数を数えている
			// ハブのバックリンク一覧へ入ってしまう。
			name: "リンク先へのリンク元",
			body: nestedBacklinkBody(fmt.Sprintf(nestedBacklinkTitleFormat, 1), nestedTarget),
			want: nestedTarget,
		},
	} {
		links := markup.ScanWikilinks(tt.body, topicNameNotes)
		if len(links) != 1 {
			t.Errorf("%sの本文のWikiリンクが 1 件であることを期待したが %d 件だった", tt.name, len(links))

			continue
		}
		if links[0].PageTitle != tt.want || links[0].TopicName != topicNameNotes {
			t.Errorf("%sのリンクが %s/%s であることを期待したが %s/%s だった",
				tt.name, topicNameNotes, tt.want, links[0].TopicName, links[0].PageTitle)
		}
	}
}

func TestLinkHubBodyHasNoHeadings(t *testing.T) {
	t.Parallel()

	// The page screen puts the page title above the body as an H1, and the
	// link listing below it under a heading of its own. A heading in the body
	// naming either of them would put the same words on the screen twice, once
	// above the body and once below it.
	//
	// [Ja] ページ画面は本文の上にページタイトルを H1 として置き、本文の下に
	// リンク一覧をそれ自身の見出しで描画する。そのどちらかを名指す見出しが本文に
	// あると、同じ文字列が本文の上と下に 2 回並ぶことになる。
	bodyHTML := markup.RenderMarkdown(linkHubBody(3))
	headingElement := regexp.MustCompile(`<h[1-6](?:[ >])`)
	if headingElement.MatchString(bodyHTML) {
		t.Errorf("ハブの本文に見出しがあった: %s", bodyHTML)
	}
}

func TestLinkHubAmountsLeavePartialLastListingPage(t *testing.T) {
	t.Parallel()

	// The three listings under a page body paginate independently and at three
	// different sizes, so each needs a count of its own to reach a last page
	// holding a remainder rather than a full one. The limits are read from the
	// view model instead of being written out here, so that raising one of them
	// fails this test and the counts get re-picked, rather than the seed
	// quietly stopping to produce the partial page.
	//
	// [Ja] ページ本文の下の 3 つの一覧はそれぞれ独立に、しかも異なる件数で
	// ページングするため、最終ページが 1 画面分ではなく端数になるには、それぞれに
	// 件数が要る。件数上限をここに書き写さず ViewModel から読むのは、上限を引き
	// 上げたときにこのテストが落ちて件数を選び直せるようにするため。書き写して
	// おくと、シードが端数のページを作らなくなったことに気づけない。
	for _, tt := range []struct {
		name          string
		count         int
		limit         int
		wantPages     int
		wantRemainder int
	}{
		{
			name:          "ページのリンク一覧",
			count:         defaultAmounts.linkHubTargets,
			limit:         int(viewmodel.LinkLimit),
			wantPages:     4,
			wantRemainder: 5,
		},
		{
			name:          "ページのバックリンク一覧",
			count:         defaultAmounts.linkHubBacklinks,
			limit:         int(viewmodel.PageBacklinkLimit),
			wantPages:     4,
			wantRemainder: 3,
		},
		{
			name:          "ネストしたバックリンク一覧",
			count:         defaultAmounts.nestedBacklinks,
			limit:         int(viewmodel.BacklinkLimit),
			wantPages:     2,
			wantRemainder: 7,
		},
	} {
		gotPages := (tt.count + tt.limit - 1) / tt.limit
		if gotPages != tt.wantPages {
			t.Errorf("%sが %d ページになることを期待したが %d ページだった", tt.name, tt.wantPages, gotPages)
		}

		gotRemainder := tt.count % tt.limit
		if gotRemainder != tt.wantRemainder {
			t.Errorf("%sの最終ページが %d 件になることを期待したが %d 件だった", tt.name, tt.wantRemainder, gotRemainder)
		}
	}
}

// firstLinkedPageTitle returns the title of the page a link list shows first,
// ordered the way FindLinkedPagesPaginated orders it.
//
// [Ja] firstLinkedPageTitle は、リンク一覧が最初に見せるページのタイトルを返す。
// 並び順は FindLinkedPagesPaginated に合わせている。
func firstLinkedPageTitle(
	ctx context.Context,
	t *testing.T,
	tx *sql.Tx,
	spaceID model.SpaceID,
	linkedPageIDs []model.PageID,
) string {
	t.Helper()

	ids := make([]string, 0, len(linkedPageIDs))
	for _, id := range linkedPageIDs {
		ids = append(ids, string(id))
	}

	var title string
	err := tx.QueryRowContext(
		ctx,
		`SELECT title FROM pages
         WHERE space_id = $1 AND id = ANY($2)
         ORDER BY modified_at DESC, id DESC
         LIMIT 1`,
		string(spaceID), pq.Array(ids),
	).Scan(&title)
	if err != nil {
		t.Fatalf("リンク一覧の先頭ページの取得に失敗: %v", err)
	}

	return title
}

// countPagesLinkingTo counts the pages whose body links to the given page,
// which is what its backlink list is built from.
//
// [Ja] countPagesLinkingTo は、本文が指定ページへリンクしているページを数える。
// バックリンク一覧はこれを元に組み立てられる。
func countPagesLinkingTo(
	ctx context.Context,
	t *testing.T,
	tx *sql.Tx,
	spaceID model.SpaceID,
	pageID model.PageID,
) int {
	t.Helper()

	var count int
	err := tx.QueryRowContext(
		ctx,
		`SELECT count(*) FROM pages WHERE space_id = $1 AND $2 = ANY(linked_page_ids)`,
		string(spaceID), string(pageID),
	).Scan(&count)
	if err != nil {
		t.Fatalf("バックリンク数の取得に失敗: %v", err)
	}

	return count
}
