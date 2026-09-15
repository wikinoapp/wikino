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

	// 件数は小さくしつつ、本文のバリエーションを一周し、1つの役割だけが書く
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
			t.Errorf("トピック%sのページが%d件であることを期待したが%d件だった", tt.topic.name, tt.want, got)
		}
	}

	// 生成したページがスペース内のすべてのページになる。本文にWikiリンクが
	// あると、resolverが件数の設定に無いページを作成し、一覧が選んだ件数を
	// 保たなくなる。
	wantTotal := amt.handbookPages + amt.privateNotesPages + amt.secretPages
	if got := countPagesInSpace(ctx, t, tx, spaces.wiki.id); got != wantTotal {
		t.Errorf("スペース全体のページが%d件であることを期待したが%d件だった", wantTotal, got)
	}

	// すべてのページが公開済みである必要がある。スペース・トピックの一覧は
	// 公開済みのページだけを数えるため、未公開のページは一覧を埋めない。
	first := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.handbook.id, topicNameHandbook+" 001")
	last := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.handbook.id, topicNameHandbook+" 005")
	if !first.published || !last.published {
		t.Error("生成したページが公開済みであることを期待したが未公開だった")
	}

	// タイトルはトピック内で採番するため、番号はスペースを跨いで続かず、
	// 次のトピックで振り直される。
	findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.privateNotes.id, topicNamePrivateNotes+" 001")
	findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.secret.id, topicNameSecret+" 001")

	// 複数の役割が書くトピックのページは、その役割へ順に回される。roleOwner
	// だけが書くトピックのページは、偶数番でもroleOwnerのものになる。
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

	// 最初の4ページで本文の全バリエーションを1回ずつ使用し、5ページ目で
	// 周回を始める。各本文にはページタイトルも含み、一覧から開いたページがどの
	// ページなのかを表示できるようにする。
	bodyMarkers := []string{
		"ここに読むべきことはありません。",
		"## 概要",
		"> シードが作成するページはドキュメントではありません。",
		"| リンク | 件数を保つため、張っていない",
	}
	if len(bulkPageBodies) != len(bodyMarkers) {
		t.Fatalf("本文のバリエーションが%d種類であることを期待したが%d種類だった", len(bodyMarkers), len(bulkPageBodies))
	}

	for number := 1; number <= 5; number++ {
		title := fmt.Sprintf("%s %03d", topicNameHandbook, number)
		page := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.handbook.id, title)
		row := readPage(ctx, t, tx, spaces.wiki.id, page.id)
		wantMarker := bodyMarkers[(number-1)%len(bodyMarkers)]
		if !strings.Contains(row.body, title) {
			t.Errorf("%sの本文に自身のタイトルが含まれていない", title)
		}
		if !strings.Contains(row.body, wantMarker) {
			t.Errorf("%sの本文にバリエーションを識別する%qが含まれていない", title, wantMarker)
		}
	}
}

func TestBulkPageBodiesCarryNoWikilinks(t *testing.T) {
	t.Parallel()

	// 埋め草の本文にWikiリンクがあると、resolverがその名前のページを作成
	// するため、一覧を確認する基準の件数が本文を編集するたびにずれていく。ここで
	// 捕まえれば理由まで示せる。生成のテストで件数が合わないだけでは示せない。
	for i, body := range bulkPageBodies {
		if got := len(markup.ScanWikilinks(fmt.Sprintf(body, "埋め草ページ"), topicNameHandbook)); got != 0 {
			t.Errorf("本文%dにWikiリンクが%d件含まれている", i, got)
		}
	}
}

func TestDefaultAmountsLeavePartialLastListingPage(t *testing.T) {
	t.Parallel()

	// 件数は、一覧のどの部分にも到達できるようにするために選んでいる。最初の
	// ページ、途中のページ、そして1画面分ではなく端数になる最終ページ。割り切れる
	// 件数では端数のページが確認できず、1画面に収まる件数ではページネーション自体が
	// 確認できない。
	//
	// 1ページあたりの件数は、それぞれのハンドラーパッケージで非公開になっている
	// spaceShowPageLimitとtopicShowPageLimitに合わせている。
	const listingPageLimit = 100

	// スペース全体の一覧は、埋め草の生成器が作るページだけでなく、スペースの
	// 公開済みページをすべて数える。1ページずつ書かれるものが2つあり、Markdown
	// 記法紹介ページとリンク集中ページがそれにあたる。後続の生成器も公開済みページを
	// 追加するため、成り立たせるべきは合計が301〜400に収まること。それが一覧を
	// 4ページに保ち、最終ページを端数にする。合計は現在399件であり、このスペースへ
	// 公開済みページを足す生成器は、どこかで1件減らす必要がある。
	//
	// エクスポート確認用ページのWikiリンクが作る未公開のページは数に入れない。
	// 未公開のページは一覧に出ないため、一覧のページ数も最終ページの端数も動かさない。
	const singlePages = 2

	// 状態バリエーションのページは数に入れない。ピン留めされたページは一覧の
	// 中ではなく上に並び、ゴミ箱のページと未公開のページはそもそも並ばないため、
	// いずれも一覧のページ数を動かさない。

	// seed-soloのページも数に入れない。もう一方のスペースにあるページであり、
	// 一覧が一度にページングするのは1つのスペースの中だけであるため。

	// 編集提案も数に入れない。編集提案は、既に公開されているページへの変更を
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
		t.Errorf("既定件数が%+vであることを期待したが%+vだった", wantAmounts, defaultAmounts)
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
				len(sandboxPageSpecs()) + len(exportPageSpecs()),
			wantPages:     4,
			wantRemainder: 99,
		},
	} {
		gotPages := (tt.count + listingPageLimit - 1) / listingPageLimit
		if gotPages != tt.wantPages {
			t.Errorf("%sが%dページになることを期待したが%dページだった", tt.name, tt.wantPages, gotPages)
		}

		gotRemainder := tt.count % listingPageLimit
		if gotRemainder != tt.wantRemainder {
			t.Errorf("%sの最終ページが%d件になることを期待したが%d件だった", tt.name, tt.wantRemainder, gotRemainder)
		}
	}
}

// countPagesInTopicは1つのトピックのページを数える。
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

// countPagesInSpaceは、トピックを問わずスペースのすべてのページを数える。
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
