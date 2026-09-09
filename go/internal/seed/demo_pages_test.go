package seed

import (
	"context"
	"database/sql"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"
	"golang.org/x/text/unicode/norm"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGenerateDemoPages(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	_, spaces := buildSeedUsersAndSpaces(t, tx, "seed-demo-pages")

	topics, err := generateTopics(ctx, tx, io.Discard, spaces)
	if err != nil {
		t.Fatalf("トピック生成に失敗: %v", err)
	}

	if err := generateDemoPages(ctx, tx, io.Discard, spaces, topics); err != nil {
		t.Fatalf("デモスペースのページの生成に失敗: %v", err)
	}

	pages, err := loadDemoPages()
	if err != nil {
		t.Fatalf("デモページ本文の読み込みに失敗: %v", err)
	}

	// The pages of the space are the bodies and nothing else. A wiki link
	// naming a title no file holds — or a title whose normalization form
	// differs from the one the file name gave — would have the resolver create
	// an unpublished page for it, and that page would be counted here.
	//
	// [Ja] スペースのページは本文の分だけで、他には無い。どのファイルも持たない
	// タイトルを名指しする Wiki リンク、あるいはファイル名が与えた正規化形と異なる
	// 形のタイトルがあれば、resolver がそのための未公開ページを作成し、それがここで
	// 数えられる。
	if got := countPagesInSpace(ctx, t, tx, spaces.demo.id); got != len(pages) {
		t.Errorf("デモスペースのページが %d 件であることを期待したが %d 件だった", len(pages), got)
	}
	if got := countPagesInTopic(ctx, t, tx, spaces.demo.id, topics.demoMemo.id); got != len(pages) {
		t.Errorf("デモトピックのページが %d 件であることを期待したが %d 件だった", len(pages), got)
	}

	rows := readDemoPages(ctx, t, tx, spaces.demo.id)
	if len(rows) != len(pages) {
		t.Fatalf("読み戻したページが %d 件であることを期待したが %d 件だった", len(pages), len(rows))
	}

	numbers := make(map[model.PageID]model.PageNumber, len(rows))
	for _, row := range rows {
		numbers[row.id] = row.number
	}

	// A page is linked to by at least one other page, which is what the demo
	// space is written to show. The links are read from the linking side, so
	// this set is what the backlink listing of each page will hold.
	//
	// [Ja] どのページも 1 枚以上の他のページからリンクされている。デモスペースが
	// 見せたいのはそれである。リンクはリンクする側から読み取るため、この集合が
	// 各ページのバックリンク一覧の中身になる。
	backlinked := make(map[model.PageID]bool, len(rows))

	for i, row := range rows {
		// The rows are read in page number order, and the numbers were handed
		// out in the order the file names sort in, so a row and a body stand at
		// the same position.
		//
		// [Ja] 行はページ番号の順で読んでおり、番号はファイル名の並び順に振られて
		// いるため、行と本文は同じ位置に並ぶ。
		if row.title != pages[i].title {
			t.Errorf("%d 番目のページのタイトルが %q であることを期待したが %q だった", i+1, pages[i].title, row.title)
		}
		if !row.publishedAt.Valid {
			t.Errorf("ページ %s が公開済みであることを期待したが未公開だった", row.title)

			continue
		}

		// The page was written once and left alone, so the three stamps of the
		// row say the same instant.
		//
		// [Ja] ページは一度書かれてそのままになっているため、行の 3 つの打刻は同じ
		// 時刻を告げる。
		if !row.publishedAt.Time.Equal(row.modifiedAt) {
			t.Errorf("ページ %s の公開日時 %v が更新日時 %v と異なる", row.title, row.publishedAt.Time, row.modifiedAt)
		}
		if !row.createdAt.Equal(row.modifiedAt) {
			t.Errorf("ページ %s の作成日時 %v が更新日時 %v と異なる", row.title, row.createdAt, row.modifiedAt)
		}

		// The listings order by modified_at, so two pages sharing a stamp would
		// stand in an order nothing decides. The earlier a page's file name
		// sorts, the more recently it was written.
		//
		// [Ja] 一覧は modified_at で並ぶため、打刻を共有する 2 枚は何も決めていない
		// 順序で並ぶことになる。ファイル名が先に並ぶページほど、新しく書かれている。
		if i > 0 && !row.modifiedAt.Before(rows[i-1].modifiedAt) {
			t.Errorf(
				"ページ %s の更新日時 %v が、1 つ前のページ %s の %v より新しい",
				row.title, row.modifiedAt, rows[i-1].title, rows[i-1].modifiedAt,
			)
		}

		if len(row.linkedPageIDs) == 0 {
			t.Errorf("ページ %s の本文の Wiki リンクが 1 つも解決していない", row.title)
		}
		for _, linked := range row.linkedPageIDs {
			number, ok := numbers[linked]
			if !ok {
				t.Errorf("ページ %s がデモスペースの外のページ %s へリンクしている", row.title, linked)

				continue
			}
			assertContains(t, row.bodyHTML, hrefOf(spaces.demo, number))
			backlinked[linked] = true
		}
	}

	for _, row := range rows {
		if !backlinked[row.id] {
			t.Errorf("ページ %s へのバックリンクが 1 つも張られていない", row.title)
		}
	}

	// Publishing leaves a revision and an editor entry per page. They are
	// counted for the space as a whole rather than read back one page at a
	// time; the first page is then checked in full, which is what says the rows
	// hold the page's own snapshot rather than merely being there.
	//
	// [Ja] 公開はページごとにリビジョンと編集者エントリを残す。これらは 1 ページ
	// ずつ読み戻すのではなくスペース全体で数え、そのうえで最初のページだけを詳しく
	// 確認する。行が「あるだけ」ではなくそのページ自身のスナップショットを持って
	// いることを述べるのは後者である。
	owner := spaces.demo.member(roleOwner)
	if got := countRowsInSpace(ctx, t, tx, "page_revisions", spaces.demo.id); got != len(pages) {
		t.Errorf("リビジョンが %d 件であることを期待したが %d 件だった", len(pages), got)
	}
	if got := countRowsInSpace(ctx, t, tx, "page_editors", spaces.demo.id); got != len(pages) {
		t.Errorf("ページ編集者が %d 件であることを期待したが %d 件だった", len(pages), got)
	}
	assertPageRevision(ctx, t, tx, spaces.demo, owner, rows[0].id, readPage(ctx, t, tx, spaces.demo.id, rows[0].id))
	assertPageEditor(ctx, t, tx, spaces.demo, owner, rows[0].id)
}

func TestLoadDemoPages(t *testing.T) {
	t.Parallel()

	pages, err := loadDemoPages()
	if err != nil {
		t.Fatalf("デモページ本文の読み込みに失敗: %v", err)
	}
	if len(pages) == 0 {
		t.Fatal("デモページ本文が 1 件も読み込まれなかった")
	}

	seen := make(map[string]bool, len(pages))
	for _, page := range pages {
		if strings.HasSuffix(page.title, demoBodyExtension) {
			t.Errorf("タイトル %q が拡張子を残している", page.title)
		}

		// A title that is not in NFC would resolve none of the wiki links that
		// name it, and the pages that name it would each create an empty page
		// of their own instead.
		//
		// [Ja] NFC でないタイトルは、それを名指しする Wiki リンクをどれも解決
		// できず、名指しした側のページがそれぞれ空のページを作ることになる。
		if !norm.NFC.IsNormalString(page.title) {
			t.Errorf("タイトル %q が NFC に正規化されていない", page.title)
		}

		// Two files whose names differ only outside the title would be two
		// pages of one title, which the unique index over (topic_id, title)
		// refuses.
		//
		// [Ja] タイトル以外のところだけが違う名前の 2 ファイルは、1 つのタイトルを
		// 持つ 2 ページになり、(topic_id, title) の一意インデックスがそれを拒む。
		if seen[page.title] {
			t.Errorf("タイトル %q が重複している", page.title)
		}
		seen[page.title] = true

		if strings.TrimSpace(page.body) == "" {
			t.Errorf("ページ %q の本文が空である", page.title)
		}
	}
}

// TestDemoPageTextsCarryNoSeedWording checks that no demo page puts the word
// the development data is called by on the screen. The titles and the bodies
// reach the help pages as the image of a wiki someone keeps, and a page named
// after the seed would say what that wiki really is. The same rule is held over
// the names of the space and the topic by
// TestDemoSpaceNamesCarryNoSeedWording.
//
// [Ja] TestDemoPageTextsCarryNoSeedWording は、開発用データの呼び名をデモページが
// 画面に出していないことを確認する。タイトルと本文は、誰かが書きためている Wiki の
// 画像としてヘルプページへ届くものであり、シードの名を持つページはその Wiki の正体を
// 明かしてしまう。スペースとトピックの名前に対する同じ規則は
// TestDemoSpaceNamesCarryNoSeedWording が受け持っている。
func TestDemoPageTextsCarryNoSeedWording(t *testing.T) {
	t.Parallel()

	pages, err := loadDemoPages()
	if err != nil {
		t.Fatalf("デモページ本文の読み込みに失敗: %v", err)
	}

	for _, page := range pages {
		for _, tt := range []struct {
			label string
			text  string
		}{
			{label: "タイトル", text: page.title},
			{label: "本文", text: page.body},
		} {
			if strings.Contains(strings.ToLower(tt.text), "seed") {
				t.Errorf("デモページ %q の%sに seed が含まれている", page.title, tt.label)
			}
			if strings.Contains(tt.text, "シード") {
				t.Errorf("デモページ %q の%sに シード が含まれている", page.title, tt.label)
			}
		}
	}
}

// demoPageRow is a stored demo page, read back with the stamps and the links
// the generator wrote.
//
// [Ja] demoPageRow は保存されたデモページ。生成器が書いた打刻とリンクを伴って
// 読み戻したもの。
type demoPageRow struct {
	id            model.PageID
	number        model.PageNumber
	title         string
	bodyHTML      string
	linkedPageIDs []model.PageID
	createdAt     time.Time
	modifiedAt    time.Time
	publishedAt   sql.NullTime
}

// readDemoPages reads back every page of the space in page number order, which
// is the order the bodies were read in.
//
// [Ja] readDemoPages はスペースのすべてのページを、ページ番号の順で読み戻す。この順
// が本文を読んだ順にあたる。
func readDemoPages(ctx context.Context, t *testing.T, tx *sql.Tx, spaceID model.SpaceID) []demoPageRow {
	t.Helper()

	rows, err := tx.QueryContext(
		ctx,
		`SELECT id, number, title, body_html, linked_page_ids, created_at, modified_at, published_at
         FROM pages WHERE space_id = $1 ORDER BY number`,
		string(spaceID),
	)
	if err != nil {
		t.Fatalf("デモページの取得に失敗: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var stored []demoPageRow
	for rows.Next() {
		var (
			row    demoPageRow
			id     string
			number int32
			linked []string
		)
		err := rows.Scan(
			&id, &number, &row.title, &row.bodyHTML, pq.Array(&linked),
			&row.createdAt, &row.modifiedAt, &row.publishedAt,
		)
		if err != nil {
			t.Fatalf("デモページの読み取りに失敗: %v", err)
		}

		row.id = model.PageID(id)
		row.number = model.PageNumber(number)
		for _, linkedID := range linked {
			row.linkedPageIDs = append(row.linkedPageIDs, model.PageID(linkedID))
		}
		stored = append(stored, row)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("デモページの走査に失敗: %v", err)
	}

	return stored
}

// countRowsInSpace counts the rows a table holds for a space.
//
// [Ja] countRowsInSpace は、テーブルがスペースについて持つ行を数える。
func countRowsInSpace(ctx context.Context, t *testing.T, tx *sql.Tx, table string, spaceID model.SpaceID) int {
	t.Helper()

	// The table name is written by the caller as a literal and cannot be a
	// placeholder, so it is interpolated. No value from the database or from a
	// body reaches this.
	//
	// [Ja] テーブル名は呼び出し側がリテラルとして書くもので、プレースホルダーには
	// できないため文字列として埋め込む。データベースや本文から来た値がここへ届く
	// ことはない。
	var count int
	err := tx.QueryRowContext(
		ctx,
		`SELECT count(*) FROM `+table+` WHERE space_id = $1`,
		string(spaceID),
	).Scan(&count)
	if err != nil {
		t.Fatalf("%s の行数の取得に失敗: %v", table, err)
	}

	return count
}
