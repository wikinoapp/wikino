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

	// 件数は小さくしつつ、各組で両方のアカウントが1回ずつ担当し、リンク先の
	// 並び順を確認できる程度にする。
	amt := amounts{linkHubTargets: 4, linkHubBacklinks: 3, nestedBacklinks: 2}

	if err := generateLinkHub(ctx, tx, io.Discard, amt, spaces, topics); err != nil {
		t.Fatalf("リンク集中ページの生成に失敗: %v", err)
	}

	hub := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.notes.id, linkHubTitle)
	if !hub.published {
		t.Errorf("%sが公開済みであることを期待したが未公開だった", linkHubTitle)
	}

	// リンク一覧こそがハブの存在理由であるため、本文が名指しするリンク先は
	// すべてページになり、ハブの行に記録されている必要がある。一覧は
	// linked_page_idsから引かれるため。
	targetIDs := make([]model.PageID, 0, amt.linkHubTargets)
	for number := 1; number <= amt.linkHubTargets; number++ {
		target := findPageByTitle(
			ctx, t, tx, spaces.wiki.id, topics.notes.id, fmt.Sprintf(linkTargetTitleFormat, number),
		)

		// リンク先は、そこを指すリンクによって作成されるため未公開のままに
		// なる。これにより、件数を別の場所で決めているスペース・トピックの一覧にも
		// 入らない。
		if target.published {
			t.Errorf("リンク先%dが未公開であることを期待したが公開済みだった", number)
		}
		targetIDs = append(targetIDs, target.id)
	}
	assertLinkedPageIDs(t, readPage(ctx, t, tx, spaces.wiki.id, hub.id).linkedPageIDs, targetIDs)

	// ネストしたバックリンクは、リンク一覧が最初に見せるリンク先に付いている
	// ときだけ、リンク一覧をページ送りせずに辿り着ける。この並び順を決めるのは
	// シードではなくデータベースであるため、前提にせず確認する。
	nestedTarget := fmt.Sprintf(linkTargetTitleFormat, amt.linkHubTargets)
	if got := firstLinkedPageTitle(ctx, t, tx, spaces.wiki.id, targetIDs); got != nestedTarget {
		t.Errorf("リンク一覧の先頭が%qであることを期待したが%qだった", nestedTarget, got)
	}

	// リンク元ページの各組はそれぞれ1つの一覧を養うため、重要なのはハブへ
	// リンクするページ数と、ネストした一覧が付くリンク先へリンクするページ数。
	if got := countPagesLinkingTo(ctx, t, tx, spaces.wiki.id, hub.id); got != amt.linkHubBacklinks {
		t.Errorf("%sへのバックリンクが%d件であることを期待したが%d件だった",
			linkHubTitle, amt.linkHubBacklinks, got)
	}
	// ハブもそのリンク先へリンクしているため、そこへリンクするページは、
	// ネストした一覧が見せる件数より1件多くなる。この一覧はハブ自身の画面に
	// 描画されるもので、バックリンク一覧は、それが描画されているページを除くため。
	// 端数の最終ページを作るために選んだ件数は、ハブを含まないほうの件数。
	nestedTargetID := targetIDs[len(targetIDs)-1]
	wantNestedLinks := amt.nestedBacklinks + 1
	if got := countPagesLinkingTo(ctx, t, tx, spaces.wiki.id, nestedTargetID); got != wantNestedLinks {
		t.Errorf("%sへのバックリンクが%d件であることを期待したが%d件だった", nestedTarget, wantNestedLinks, got)
	}

	// リンク元ページはシードが書いた本文を持つため、公開済みである必要がある。
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

	// 途中で他のページは作られていない。余分なページがあれば、本文が意図
	// しない先へリンクしたということであり、シードが基準にしている一覧の件数が
	// 成り立たなくなる。
	wantTotal := 1 + amt.linkHubTargets + amt.linkHubBacklinks + amt.nestedBacklinks
	if got := countPagesInSpace(ctx, t, tx, spaces.wiki.id); got != wantTotal {
		t.Errorf("スペース全体のページが%d件であることを期待したが%d件だった", wantTotal, got)
	}
}

func TestLinkHubBodiesLinkOnlyWhereIntended(t *testing.T) {
	t.Parallel()

	// これらの本文のリンクはいずれもページを作るか既存のページを掴むもので、
	// 一覧は正確な件数で確認している。意図していない場所に書かれたリンクは、生成
	// 時には何も失敗させずに件数だけを動かすため、本文がどこへリンクするのかを
	// レンダリングより前に読んで確認する。
	const targets = 3

	hubLinks := markup.ScanWikilinks(linkHubBody(targets), topicNameNotes)
	if len(hubLinks) != targets {
		t.Fatalf("ハブの本文のWikiリンクが%d件であることを期待したが%d件だった", targets, len(hubLinks))
	}
	for i, link := range hubLinks {
		want := fmt.Sprintf(linkTargetTitleFormat, i+1)
		if link.PageTitle != want || link.TopicName != topicNameNotes {
			t.Errorf("ハブの%d番目のリンクが%s/%sであることを期待したが%s/%sだった",
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
			// こちらは本文で「リンクハブ」にも言及している。リンクにしてよいのは
			// リンク先だけ。ハブへのリンクにすると、このページが件数を数えている
			// ハブのバックリンク一覧へ入ってしまう。
			name: "リンク先へのリンク元",
			body: nestedBacklinkBody(fmt.Sprintf(nestedBacklinkTitleFormat, 1), nestedTarget),
			want: nestedTarget,
		},
	} {
		links := markup.ScanWikilinks(tt.body, topicNameNotes)
		if len(links) != 1 {
			t.Errorf("%sの本文のWikiリンクが1件であることを期待したが%d件だった", tt.name, len(links))

			continue
		}
		if links[0].PageTitle != tt.want || links[0].TopicName != topicNameNotes {
			t.Errorf("%sのリンクが%s/%sであることを期待したが%s/%sだった",
				tt.name, topicNameNotes, tt.want, links[0].TopicName, links[0].PageTitle)
		}
	}
}

func TestLinkHubBodyHasNoHeadings(t *testing.T) {
	t.Parallel()

	// ページ画面は本文の上にページタイトルをH1として置き、本文の下に
	// リンク一覧をそれ自身の見出しで描画する。そのどちらかを名指す見出しが本文に
	// あると、同じ文字列が本文の上と下に2回並ぶことになる。
	bodyHTML := markup.RenderMarkdown(linkHubBody(3))
	headingElement := regexp.MustCompile(`<h[1-6](?:[ >])`)
	if headingElement.MatchString(bodyHTML) {
		t.Errorf("ハブの本文に見出しがあった: %s", bodyHTML)
	}
}

func TestLinkHubAmountsLeavePartialLastListingPage(t *testing.T) {
	t.Parallel()

	// ページ本文の下の3つの一覧はそれぞれ独立にページングするため、最終ページが
	// 1画面分ではなく端数になるには、それぞれに件数が要る。各ページの件数は一定では
	// なく、1ページ目は初回件数、後続ページはそれより1件多く持つ。そのため、カードが
	// どのページに載るかはここで計算し直さず、一覧自身がページングに使うViewModelの
	// 関数から読む。こうすると、どちらの件数を変えてもこのテストが落ちて件数を選び直せる
	// ようになり、シードが端数のページを作らなくなったことを見逃さずに済む。
	for _, tt := range []struct {
		name          string
		count         int
		limit         int32
		wantPages     int
		wantRemainder int
	}{
		{
			name:          "ページのリンク一覧",
			count:         defaultAmounts.linkHubTargets,
			limit:         viewmodel.LinkLimit,
			wantPages:     4,
			wantRemainder: 6,
		},
		{
			name:          "ページのバックリンク一覧",
			count:         defaultAmounts.linkHubBacklinks,
			limit:         viewmodel.PageBacklinkLimit,
			wantPages:     4,
			wantRemainder: 1,
		},
		{
			name:          "ネストしたバックリンク一覧",
			count:         defaultAmounts.nestedBacklinks,
			limit:         viewmodel.BacklinkLimit,
			wantPages:     2,
			wantRemainder: 6,
		},
	} {
		gotPages := viewmodel.RelatedPageTotalPages(int64(tt.count), tt.limit)
		if gotPages != tt.wantPages {
			t.Errorf("%sが%dページになることを期待したが%dページだった", tt.name, tt.wantPages, gotPages)
		}

		// 最終ページの件数は、各カードがどのページに載るかを問い合わせて数える。
		// こうすると端数が、一覧が描画に使うのと同じ対応付けに従う。
		gotRemainder := 0
		for index := range tt.count {
			if int(viewmodel.RelatedPageNumberForIndex(index, tt.limit)) == gotPages {
				gotRemainder++
			}
		}
		if gotRemainder != tt.wantRemainder {
			t.Errorf("%sの最終ページが%d件になることを期待したが%d件だった", tt.name, tt.wantRemainder, gotRemainder)
		}
	}
}

// firstLinkedPageTitleは、リンク一覧が最初に見せるページのタイトルを返す。
// 並び順はFindLinkedPagesPaginatedに合わせている。
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

// countPagesLinkingToは、本文が指定ページへリンクしているページを数える。
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
