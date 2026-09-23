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

	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGenerateDemoDrafts(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	_, spaces := buildSeedUsersAndSpaces(t, tx, "seed-demo-drafts")

	topics, err := generateTopics(ctx, tx, io.Discard, spaces)
	if err != nil {
		t.Fatalf("トピック生成に失敗: %v", err)
	}
	if err := generateDemoPages(ctx, tx, io.Discard, spaces, topics); err != nil {
		t.Fatalf("デモスペースのページの生成に失敗: %v", err)
	}

	pageCount := countPagesInSpace(ctx, t, tx, spaces.demo.id)

	stamps := newDraftStamps(time.Now())
	if err := generateDemoDrafts(ctx, tx, io.Discard, spaces, topics, stamps); err != nil {
		t.Fatalf("デモスペースの下書きの生成に失敗: %v", err)
	}

	drafts, err := loadDemoDrafts()
	if err != nil {
		t.Fatalf("デモ下書き本文の読み込みに失敗: %v", err)
	}

	// 下書きの本文が名指しするページはすべてデモページの中にある。そうでなければ
	// resolverが未公開のページを作り、ここで数えられる。
	if got := countPagesInSpace(ctx, t, tx, spaces.demo.id); got != pageCount {
		t.Errorf("下書きの作成後もページが%d件であることを期待したが%d件だった", pageCount, got)
	}
	if got := countRowsInSpace(ctx, t, tx, "draft_pages", spaces.demo.id); got != len(drafts) {
		t.Errorf("下書きが%d件であることを期待したが%d件だった", len(drafts), got)
	}

	owner := spaces.demo.member(roleOwner)
	var previousModifiedAt time.Time

	for i, draft := range drafts {
		var (
			pageID      string
			pinnedAt    *time.Time
			publishedAt time.Time
			draftID     string
			memberID    string
			body        string
			linked      []string
			modifiedAt  time.Time
		)
		err := tx.QueryRowContext(
			ctx,
			`SELECT p.id, p.pinned_at, p.published_at, d.id, d.space_member_id, d.body, d.linked_page_ids, d.modified_at
             FROM draft_pages d
             INNER JOIN pages p ON p.id = d.page_id AND p.space_id = d.space_id
             WHERE d.space_id = $1 AND p.title = $2`,
			string(spaces.demo.id), draft.pageTitle,
		).Scan(&pageID, &pinnedAt, &publishedAt, &draftID, &memberID, &body, pq.Array(&linked), &modifiedAt)
		if err != nil {
			t.Fatalf("ページ%sの下書きの取得に失敗: %v", draft.pageTitle, err)
		}

		if model.SpaceMemberID(memberID) != owner.id {
			t.Errorf("ページ%sの下書きの書き手がownerではない", draft.pageTitle)
		}
		if body != draft.bodies[len(draft.bodies)-1] {
			t.Errorf("ページ%sの下書きが最後の保存の本文を持っていない", draft.pageTitle)
		}
		if len(linked) == 0 {
			t.Errorf("ページ%sの下書きの本文のWikiリンクが1つも解決していない", draft.pageTitle)
		}

		// ピン留めするのはスクリーンショットに写すページだけ。
		if featured := draft.pageTitle == demoFeaturedPageTitle; featured != (pinnedAt != nil) {
			t.Errorf("ページ%sのピン留めが%tであることを期待したが%tだった", draft.pageTitle, featured, pinnedAt != nil)
		}

		// 先に作った下書きほど新しく、スクリーンショットに写す下書きが一覧の
		// 先頭に並ぶ。
		if i > 0 && !modifiedAt.Before(previousModifiedAt) {
			t.Errorf("下書き%sの更新日時%vが、1つ前の下書きの%vより新しい", draft.pageTitle, modifiedAt, previousModifiedAt)
		}
		previousModifiedAt = modifiedAt

		revisions := readDemoDraftRevisions(ctx, t, tx, spaces.demo.id, draftID)
		if len(revisions) != len(draft.bodies) {
			t.Fatalf("下書き%sのリビジョンが%d件であることを期待したが%d件だった", draft.pageTitle, len(draft.bodies), len(revisions))
		}
		for j, revision := range revisions {
			if revision.body != draft.bodies[j] {
				t.Errorf("下書き%sの%d回目の保存の本文が異なる", draft.pageTitle, j+1)
			}
			if !revision.createdAt.After(publishedAt) {
				t.Errorf("下書き%sの%d回目の保存%vが、ページの公開%vより前にある", draft.pageTitle, j+1, revision.createdAt, publishedAt)
			}
			if j > 0 && !revision.createdAt.After(revisions[j-1].createdAt) {
				t.Errorf("下書き%sの%d回目の保存が、その前の保存より新しくない", draft.pageTitle, j+1)
			}
		}
		if last := revisions[len(revisions)-1]; !last.createdAt.Equal(modifiedAt) {
			t.Errorf("下書き%sの最後の保存%vが、下書きの更新日時%vと異なる", draft.pageTitle, last.createdAt, modifiedAt)
		}
	}
}

func TestLoadDemoDrafts(t *testing.T) {
	t.Parallel()

	drafts, err := loadDemoDrafts()
	if err != nil {
		t.Fatalf("デモ下書き本文の読み込みに失敗: %v", err)
	}

	pages, err := loadDemoPages()
	if err != nil {
		t.Fatalf("デモページ本文の読み込みに失敗: %v", err)
	}
	titles := make(map[string]bool, len(pages))
	for _, page := range pages {
		titles[page.title] = true
	}

	if drafts[0].pageTitle != demoFeaturedPageTitle {
		t.Errorf("先頭の下書きが%sであることを期待したが%sだった", demoFeaturedPageTitle, drafts[0].pageTitle)
	}

	for _, draft := range drafts {
		if !norm.NFC.IsNormalString(draft.pageTitle) {
			t.Errorf("タイトル%qがNFCに正規化されていない", draft.pageTitle)
		}
		if !titles[draft.pageTitle] {
			t.Errorf("下書き%qの対象とするデモページが無い", draft.pageTitle)
		}

		// 編集履歴の差分が空にならないよう、どの保存も前の保存から本文を変えている。
		for i, body := range draft.bodies {
			if strings.TrimSpace(body) == "" {
				t.Errorf("下書き%qの%d回目の保存の本文が空である", draft.pageTitle, i+1)
			}
			if strings.Contains(strings.ToLower(body), "seed") || strings.Contains(body, "シード") {
				t.Errorf("下書き%qの%d回目の保存の本文に開発用データの呼び名が含まれている", draft.pageTitle, i+1)
			}
			if i > 0 && body == draft.bodies[i-1] {
				t.Errorf("下書き%qの%d回目の保存が、前の保存と同じ本文である", draft.pageTitle, i+1)
			}
		}
	}
}

// TestDemoFeaturedDraftKeepsRelatedLinksShortは、スクリーンショットに写す
// 下書きのリンク先が2〜3件で、そのうち関連リンクを持つのが1件だけであることを
// 確認する。デモページの本文を書き足して、このページのリンク先へ新たにリンクすると、
// 関連リンクの段が増えて画面が縦に伸びる。それをここで検出する。
//
// 関連リンクは、リンク先ページへのバックリンクから、このページ自身とリンク先の
// ページを除いたもの。
func TestDemoFeaturedDraftKeepsRelatedLinksShort(t *testing.T) {
	t.Parallel()

	drafts, err := loadDemoDrafts()
	if err != nil {
		t.Fatalf("デモ下書き本文の読み込みに失敗: %v", err)
	}
	pages, err := loadDemoPages()
	if err != nil {
		t.Fatalf("デモページ本文の読み込みに失敗: %v", err)
	}

	featured := drafts[0]
	linked := make(map[string]bool)
	for _, key := range markup.ScanWikilinks(featured.bodies[len(featured.bodies)-1], topicNameDemoMemo) {
		linked[key.PageTitle] = true
	}
	if len(linked) < 2 || len(linked) > 3 {
		t.Fatalf("%sの下書きのリンク先が2〜3件であることを期待したが%d件だった", featured.pageTitle, len(linked))
	}

	var withRelated []string
	for title := range linked {
		for _, page := range pages {
			if page.title == featured.pageTitle || linked[page.title] {
				continue
			}
			if linksTo(page.body, title) {
				withRelated = append(withRelated, title)

				break
			}
		}
	}
	if len(withRelated) != 1 {
		t.Errorf("関連リンクを持つリンク先が1件であることを期待したが%v だった", withRelated)
	}
}

// linksToは、本文がtitleのページへWikiリンクしているかを返す。
func linksTo(body string, title string) bool {
	for _, key := range markup.ScanWikilinks(body, topicNameDemoMemo) {
		if key.PageTitle == title {
			return true
		}
	}

	return false
}

// demoDraftRevisionRowは保存された下書きのリビジョン。
type demoDraftRevisionRow struct {
	body      string
	createdAt time.Time
}

// readDemoDraftRevisionsは下書きのリビジョンを古い保存から順に読み戻す。
func readDemoDraftRevisions(ctx context.Context, t *testing.T, tx *sql.Tx, spaceID model.SpaceID, draftID string) []demoDraftRevisionRow {
	t.Helper()

	rows, err := tx.QueryContext(
		ctx,
		`SELECT body, created_at FROM draft_page_revisions
         WHERE space_id = $1 AND draft_page_id = $2
         ORDER BY created_at`,
		string(spaceID), draftID,
	)
	if err != nil {
		t.Fatalf("下書きのリビジョンの取得に失敗: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var revisions []demoDraftRevisionRow
	for rows.Next() {
		var row demoDraftRevisionRow
		if err := rows.Scan(&row.body, &row.createdAt); err != nil {
			t.Fatalf("下書きのリビジョンの読み取りに失敗: %v", err)
		}
		revisions = append(revisions, row)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("下書きのリビジョンの走査に失敗: %v", err)
	}

	return revisions
}
