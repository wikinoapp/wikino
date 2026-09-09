package seed

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGenerateBulkPages(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	spaces := buildSeedSpaces(t, tx, "seed-bulk")
	topics, err := generateTopics(ctx, tx, io.Discard, spaces)
	if err != nil {
		t.Fatalf("トピック生成に失敗: %v", err)
	}

	// Small amounts, but enough of them to run past the body variations and to
	// reach a topic only one role writes into.
	//
	// [Ja] 件数は小さくしつつ、本文のバリエーションを一周し、1 つの役割だけが書く
	// トピックまで届く程度にする。
	amt := amounts{handbookPages: 5, privateNotesPages: 2, secretPages: 2}

	if err := generateBulkPages(ctx, tx, io.Discard, amt, spaces, topics); err != nil {
		t.Fatalf("ページネーション用ページの生成に失敗: %v", err)
	}

	for _, tt := range []struct {
		topic *seededTopic
		want  int
	}{
		{topic: topics.handbook, want: amt.handbookPages},
		{topic: topics.privateNotes, want: amt.privateNotesPages},
		{topic: topics.secret, want: amt.secretPages},
	} {
		if got := countPagesInTopic(ctx, t, tx, spaces.wiki.id, tt.topic.id); got != tt.want {
			t.Errorf("トピック %s のページが %d 件であることを期待したが %d 件だった", tt.topic.name, tt.want, got)
		}
	}

	// The generated pages are the only pages in the space. A body carrying a
	// wiki link would have the resolver create pages the amounts do not
	// account for, and the listings would no longer hold the counts they were
	// chosen for.
	//
	// [Ja] 生成したページがスペース内のすべてのページになる。本文に Wiki リンクが
	// あると、resolver が件数の設定に無いページを作成し、一覧が選んだ件数を
	// 保たなくなる。
	wantTotal := amt.handbookPages + amt.privateNotesPages + amt.secretPages
	if got := countPagesInSpace(ctx, t, tx, spaces.wiki.id); got != wantTotal {
		t.Errorf("スペース全体のページが %d 件であることを期待したが %d 件だった", wantTotal, got)
	}

	// Every page has to be published: the space and topic listings count only
	// published pages, so an unpublished one fills nothing.
	//
	// [Ja] すべてのページが公開済みである必要がある。スペース・トピックの一覧は
	// 公開済みのページだけを数えるため、未公開のページは一覧を埋めない。
	first := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.handbook.id, topicNameHandbook+" 001")
	last := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.handbook.id, topicNameHandbook+" 005")
	if !first.published || !last.published {
		t.Error("生成したページが公開済みであることを期待したが未公開だった")
	}

	// Titles are numbered within their topic, so the numbering restarts in the
	// next topic rather than continuing across the space.
	//
	// [Ja] タイトルはトピック内で採番するため、番号はスペースを跨いで続かず、
	// 次のトピックで振り直される。
	findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.privateNotes.id, topicNamePrivateNotes+" 001")
	findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.secret.id, topicNameSecret+" 001")

	// The pages of a topic more than one role writes into are handed round those
	// roles. Pages of a topic only roleOwner writes into stay with roleOwner,
	// even when their number is even.
	//
	// [Ja] 複数の役割が書くトピックのページは、その役割へ順に回される。roleOwner
	// だけが書くトピックのページは、偶数番でも roleOwner のものになる。
	for _, tt := range []struct {
		topic  *seededTopic
		title  string
		editor *seededSpaceMember
	}{
		{topic: topics.handbook, title: topicNameHandbook + " 001", editor: spaces.wiki.member(roleOwner)},
		{topic: topics.handbook, title: topicNameHandbook + " 002", editor: spaces.wiki.member(roleCollaborator)},
		{topic: topics.privateNotes, title: topicNamePrivateNotes + " 001", editor: spaces.wiki.member(roleOwner)},
		{topic: topics.privateNotes, title: topicNamePrivateNotes + " 002", editor: spaces.wiki.member(roleCollaborator)},
		{topic: topics.secret, title: topicNameSecret + " 001", editor: spaces.wiki.member(roleOwner)},
		{topic: topics.secret, title: topicNameSecret + " 002", editor: spaces.wiki.member(roleOwner)},
	} {
		page := findPageByTitle(ctx, t, tx, spaces.wiki.id, tt.topic.id, tt.title)
		assertPageEditor(ctx, t, tx, spaces.wiki, tt.editor, page.id)
	}

	// The first four pages use every body variation once, and the fifth page
	// starts the cycle again. Each body also carries its page title, so a page
	// opened from the listing shows which one it is.
	//
	// [Ja] 最初の 4 ページで本文の全バリエーションを 1 回ずつ使用し、5 ページ目で
	// 周回を始める。各本文にはページタイトルも含み、一覧から開いたページがどの
	// ページなのかを表示できるようにする。
	bodyMarkers := []string{
		"ここに読むべきことはありません。",
		"## 概要",
		"> シードが作成するページはドキュメントではありません。",
		"| リンク | 件数を保つため、張っていない",
	}
	if len(bulkPageBodies) != len(bodyMarkers) {
		t.Fatalf("本文のバリエーションが %d 種類であることを期待したが %d 種類だった", len(bodyMarkers), len(bulkPageBodies))
	}

	for number := 1; number <= 5; number++ {
		title := fmt.Sprintf("%s %03d", topicNameHandbook, number)
		page := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.handbook.id, title)
		row := readPage(ctx, t, tx, spaces.wiki.id, page.id)
		wantMarker := bodyMarkers[(number-1)%len(bodyMarkers)]
		if !strings.Contains(row.body, title) {
			t.Errorf("%s の本文に自身のタイトルが含まれていない", title)
		}
		if !strings.Contains(row.body, wantMarker) {
			t.Errorf("%s の本文にバリエーションを識別する %q が含まれていない", title, wantMarker)
		}
	}
}

func TestBulkPageBodiesCarryNoWikilinks(t *testing.T) {
	t.Parallel()

	// A wiki link in a filler body would make the resolver create the page it
	// names, so the counts the listings are checked against would drift with
	// every body edit. Catching it here says why, which a count mismatch in the
	// generation test would not.
	//
	// [Ja] 埋め草の本文に Wiki リンクがあると、resolver がその名前のページを作成
	// するため、一覧を確認する基準の件数が本文を編集するたびにずれていく。ここで
	// 捕まえれば理由まで示せる。生成のテストで件数が合わないだけでは示せない。
	for i, body := range bulkPageBodies {
		if got := len(markup.ScanWikilinks(fmt.Sprintf(body, "埋め草ページ"), topicNameHandbook)); got != 0 {
			t.Errorf("本文 %d にWikiリンクが %d 件含まれている", i, got)
		}
	}
}

func TestDefaultAmountsLeavePartialLastListingPage(t *testing.T) {
	t.Parallel()

	// The counts exist to make every part of a listing reachable: a first page,
	// a middle one, and a last one holding a remainder rather than a full
	// screen. A count that divides evenly hides the partial page, and one that
	// fits on a single screen hides pagination altogether.
	//
	// The per-page count mirrors spaceShowPageLimit and topicShowPageLimit,
	// which are unexported in their handler packages.
	//
	// [Ja] 件数は、一覧のどの部分にも到達できるようにするために選んでいる。最初の
	// ページ、途中のページ、そして 1 画面分ではなく端数になる最終ページ。割り切れる
	// 件数では端数のページが確認できず、1 画面に収まる件数ではページネーション自体が
	// 確認できない。
	//
	// 1 ページあたりの件数は、それぞれのハンドラーパッケージで非公開になっている
	// spaceShowPageLimit と topicShowPageLimit に合わせている。
	const listingPageLimit = 100

	// The space-wide listing counts every published page of the space, not only
	// the ones the filler generator makes. Two more are written a page at a
	// time: the Markdown guide and the link hub. Later generators add published
	// pages of their own too, so what has to hold is that the total stays
	// between 301 and 400, which is what keeps the listing at 4 pages with a
	// remainder on the last.
	//
	// [Ja] スペース全体の一覧は、埋め草の生成器が作るページだけでなく、スペースの
	// 公開済みページをすべて数える。1 ページずつ書かれるものが 2 つあり、Markdown
	// 記法紹介ページとリンク集中ページがそれにあたる。後続の生成器も公開済みページを
	// 追加するため、成り立たせるべきは合計が 301〜400 に収まること。それが一覧を
	// 4 ページに保ち、最終ページを端数にする。
	const singlePages = 2

	// The state variations are left out of the count. Pinned pages are listed
	// above the listing rather than in it, and trashed and unpublished pages
	// are not listed at all, so none of them moves the number of listing pages.
	//
	// [Ja] 状態バリエーションのページは数に入れない。ピン留めされたページは一覧の
	// 中ではなく上に並び、ゴミ箱のページと未公開のページはそもそも並ばないため、
	// いずれも一覧のページ数を動かさない。

	// The pages of seed-solo are left out too. They are in the other space, and
	// a listing pages through one space at a time.
	//
	// [Ja] seed-solo のページも数に入れない。もう一方のスペースにあるページであり、
	// 一覧が一度にページングするのは 1 つのスペースの中だけであるため。

	// The suggestions are left out as well. A suggestion proposes a change to a
	// page that has already been published rather than adding one of its own,
	// so however many of them are created, the counts a listing pages through
	// stay where they are.
	//
	// [Ja] 編集提案も数に入れない。編集提案は、既に公開されているページへの変更を
	// 提案するものであり、自前のページを追加しない。そのため何件作られても、一覧が
	// ページングする件数は動かない。
	wantAmounts := amounts{
		handbookPages:          250,
		privateNotesPages:      60,
		secretPages:            10,
		linkHubTargets:         50,
		linkHubBacklinks:       45,
		nestedBacklinks:        20,
		pinnedPages:            3,
		trashedPages:           3,
		soloNotesPages:         5,
		soloSecretPages:        3,
		ownerDraftPages:        22,
		collaboratorDraftPages: 6,
		draftRevisions:         24,
		openSuggestions:        20,
		appliedSuggestions:     4,
		closedSuggestions:      4,
		suggestionComments:     6,
	}
	if defaultAmounts != wantAmounts {
		t.Errorf("既定件数が %+v であることを期待したが %+v だった", wantAmounts, defaultAmounts)
	}

	for _, tt := range []struct {
		name          string
		count         int
		wantPages     int
		wantRemainder int
	}{
		{
			name:          "「ハンドブック」のページ一覧",
			count:         defaultAmounts.handbookPages,
			wantPages:     3,
			wantRemainder: 50,
		},
		{
			name: "スペースのページ一覧",
			count: defaultAmounts.handbookPages + defaultAmounts.privateNotesPages + defaultAmounts.secretPages +
				defaultAmounts.linkHubBacklinks + defaultAmounts.nestedBacklinks + singlePages +
				len(sandboxPageSpecs()),
			wantPages:     4,
			wantRemainder: 91,
		},
	} {
		gotPages := (tt.count + listingPageLimit - 1) / listingPageLimit
		if gotPages != tt.wantPages {
			t.Errorf("%sが %d ページになることを期待したが %d ページだった", tt.name, tt.wantPages, gotPages)
		}

		gotRemainder := tt.count % listingPageLimit
		if gotRemainder != tt.wantRemainder {
			t.Errorf("%sの最終ページが %d 件になることを期待したが %d 件だった", tt.name, tt.wantRemainder, gotRemainder)
		}
	}
}

// countPagesInTopic counts the pages of one topic.
//
// [Ja] countPagesInTopic は 1 つのトピックのページを数える。
func countPagesInTopic(
	ctx context.Context,
	t *testing.T,
	tx *sql.Tx,
	spaceID model.SpaceID,
	topicID model.TopicID,
) int {
	t.Helper()

	var count int
	err := tx.QueryRowContext(
		ctx,
		`SELECT count(*) FROM pages WHERE space_id = $1 AND topic_id = $2`,
		string(spaceID), string(topicID),
	).Scan(&count)
	if err != nil {
		t.Fatalf("トピックのページ数の取得に失敗: %v", err)
	}

	return count
}

// countPagesInSpace counts every page of a space, whatever topic it is in.
//
// [Ja] countPagesInSpace は、トピックを問わずスペースのすべてのページを数える。
func countPagesInSpace(ctx context.Context, t *testing.T, tx *sql.Tx, spaceID model.SpaceID) int {
	t.Helper()

	var count int
	err := tx.QueryRowContext(
		ctx,
		`SELECT count(*) FROM pages WHERE space_id = $1`,
		string(spaceID),
	).Scan(&count)
	if err != nil {
		t.Fatalf("スペースのページ数の取得に失敗: %v", err)
	}

	return count
}
