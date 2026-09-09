package seed

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGenerateDraftPages(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	spaces := buildSeedSpaces(t, tx, "seed-draft")
	topics, err := generateTopics(ctx, tx, io.Discard, spaces)
	if err != nil {
		t.Fatalf("トピック生成に失敗: %v", err)
	}

	amt := amounts{ownerDraftPages: 6, collaboratorDraftPages: 2, draftRevisions: 5}
	publishPagesForDrafts(ctx, t, tx, spaces.wiki, topics)

	if err := generateDraftPages(ctx, tx, io.Discard, amt, spaces, topics, newDraftStamps(time.Now())); err != nil {
		t.Fatalf("下書きページの生成に失敗: %v", err)
	}

	// A draft belongs to the member who wrote it, so each account's drafts are
	// counted on their own: the listings are filled per account.
	//
	// [Ja] 下書きは書いたメンバーのものであるため、件数はアカウントごとに数える。
	// 一覧がアカウントごとに埋まるため。
	ownerDrafts := listDraftPages(ctx, t, tx, spaces.wiki.id, spaces.wiki.member(roleOwner).id)
	if len(ownerDrafts) != amt.ownerDraftPages {
		t.Fatalf("ownerの下書きが %d 件であることを期待したが %d 件だった", amt.ownerDraftPages, len(ownerDrafts))
	}
	collaboratorDrafts := listDraftPages(ctx, t, tx, spaces.wiki.id, spaces.wiki.member(roleCollaborator).id)
	if len(collaboratorDrafts) != amt.collaboratorDraftPages {
		t.Fatalf("collaboratorの下書きが %d 件であることを期待したが %d 件だった", amt.collaboratorDraftPages, len(collaboratorDrafts))
	}

	// The listings that show drafts order them by modified_at descending, and
	// the home screen keeps only the newest few. The drafts on unpublished
	// pages have to be at that end: they are the ones written to be met there.
	//
	// [Ja] 下書きを見せる一覧は modified_at の降順で並べ、ホーム画面は新しいものを
	// 数件しか残さない。未公開のページに付く下書きはその側に無ければならない。
	// そこで出会うために書かれた下書きであるため。
	specs := newPageDraftSpecs()
	for i, spec := range specs {
		draft := ownerDrafts[i]
		assertDraftTitle(t, draft, spec.title)

		if page := readPageBehindDraft(ctx, t, tx, spaces.wiki.id, draft.pageID); page.published {
			t.Errorf("%s のページが未公開であることを期待したが公開済みだった", draftLabelOf(draft))
		}
		if draft.topicID != topics.notes.id {
			t.Errorf("%s が「ノート」に置かれていない", draftLabelOf(draft))
		}

		wantRevisions := ordinaryDraftRevisions
		if spec.longHistory {
			wantRevisions = amt.draftRevisions
		}
		if got := len(listDraftPageRevisions(ctx, t, tx, spaces.wiki.id, draft.id)); got != wantRevisions {
			t.Errorf("%s のリビジョンが %d 件であることを期待したが %d 件だった", draftLabelOf(draft), wantRevisions, got)
		}
	}

	// The rest of roleOwner's drafts, and all of roleCollaborator's, are written
	// against pages that are already published. Such a draft keeps the page's
	// title, and the page keeps whatever was published last.
	//
	// [Ja] roleOwner の残りの下書きと roleCollaborator のすべての下書きは、既に公開
	// されているページに対して書かれている。この種の下書きはページのタイトルを
	// そのまま持ち、ページ側は最後に公開された内容を保ったままになる。
	onPublishedPages := make([]draftPageRow, 0, len(ownerDrafts)-len(specs)+len(collaboratorDrafts))
	onPublishedPages = append(onPublishedPages, ownerDrafts[len(specs):]...)
	onPublishedPages = append(onPublishedPages, collaboratorDrafts...)

	for _, draft := range onPublishedPages {
		page := readPageBehindDraft(ctx, t, tx, spaces.wiki.id, draft.pageID)
		if !page.published {
			t.Errorf("%s のページが公開済みであることを期待したが未公開だった", draftLabelOf(draft))
		}
		if page.title == nil {
			t.Fatalf("%s のページにタイトルが無い", draftLabelOf(draft))
		}
		if draft.title == nil || *draft.title != *page.title {
			t.Errorf("%s がページのタイトル %q を持つことを期待した", draftLabelOf(draft), *page.title)
		}
		if draft.body == page.body {
			t.Errorf("%s の本文が、公開されているページの本文と同じだった", draftLabelOf(draft))
		}
	}

	// The drafts have to reach more than one topic: the draft listing groups by
	// space and topic, and a single topic would leave it with one group.
	//
	// [Ja] 下書きは複数のトピックに行き渡る必要がある。下書き一覧画面はスペースと
	// トピックでグループ分けするため、1 つのトピックに寄ると 1 グループになる。
	topicIDs := make(map[model.TopicID]bool)
	for _, draft := range ownerDrafts {
		topicIDs[draft.topicID] = true
	}
	for _, topic := range []*seededTopic{topics.handbook, topics.notes, topics.privateNotes} {
		if !topicIDs[topic.id] {
			t.Errorf("トピック %s にownerの下書きが 1 件も無い", topic.name)
		}
	}

	allDrafts := make([]draftPageRow, 0, len(ownerDrafts)+len(collaboratorDrafts))
	allDrafts = append(allDrafts, ownerDrafts...)
	allDrafts = append(allDrafts, collaboratorDrafts...)
	assertDraftsCarryWhatTheLastSaveWrote(ctx, t, tx, spaces.wiki.id, allDrafts)
}

// assertDraftsCarryWhatTheLastSaveWrote checks that every draft holds the body
// of its newest revision, and that no two drafts share a modified_at.
//
// A draft is what the editor opens and its newest revision is what the last
// save stored, so the two differing would be a state no save can produce.
// Sharing a modified_at would leave the order of the listings to the ids, and
// which drafts the home screen keeps would change from run to run.
//
// [Ja] assertDraftsCarryWhatTheLastSaveWrote は、各下書きが自身の最新リビジョンの
// 本文を保持していること、そして modified_at が下書き間で重複していないことを
// 確認する。
//
// 下書きは編集画面が開くもので、その最新リビジョンは最後の保存が保存したもので
// あるため、両者が食い違う状態はどの保存でも作れない。modified_at が重複すると
// 一覧の並び順が ID 任せになり、ホーム画面が残す下書きが実行ごとに変わってしまう。
func assertDraftsCarryWhatTheLastSaveWrote(
	ctx context.Context,
	t *testing.T,
	tx *sql.Tx,
	spaceID model.SpaceID,
	drafts []draftPageRow,
) {
	t.Helper()

	seen := make(map[time.Time]bool, len(drafts))
	for _, draft := range drafts {
		if seen[draft.modifiedAt] {
			t.Errorf("%s の最終更新時刻が他の下書きと重複している", draftLabelOf(draft))
		}
		seen[draft.modifiedAt] = true

		revisions := listDraftPageRevisions(ctx, t, tx, spaceID, draft.id)
		if len(revisions) == 0 {
			t.Fatalf("%s にリビジョンが 1 件も無い", draftLabelOf(draft))
		}

		// A revision stores the title the save was made under. A draft that has
		// never been given a title is saved under none, and the column that
		// holds it takes an empty string rather than a NULL.
		//
		// [Ja] リビジョンは、その保存が行われた時点のタイトルを保存する。一度も
		// タイトルを付けられていない下書きはタイトル無しで保存され、それを保持する
		// 列は NULL ではなく空文字列を取る。
		wantTitle := ""
		if draft.title != nil {
			wantTitle = *draft.title
		}
		for _, revision := range revisions {
			if revision.title != wantTitle {
				t.Errorf(
					"%s のリビジョンのタイトルが %q であることを期待したが %q だった",
					draftLabelOf(draft), wantTitle, revision.title,
				)
			}
		}

		if draft.body != revisions[0].body {
			t.Errorf("%s の本文が最新リビジョンの本文と一致しない", draftLabelOf(draft))
		}
		if draft.bodyHTML != revisions[0].bodyHTML {
			t.Errorf("%s の本文HTMLが最新リビジョンの本文HTMLと一致しない", draftLabelOf(draft))
		}
		if draft.bodyHTML == "" {
			t.Errorf("%s の本文HTMLが空だった", draftLabelOf(draft))
		}
	}
}

func TestDraftRevisionsGrowByOneLinePerSave(t *testing.T) {
	t.Parallel()

	// The edit history is read through the diff of one revision against the one
	// before it. Bodies that repeat unchanged would leave every diff empty, and
	// bodies rewritten wholesale would leave every diff showing everything.
	//
	// [Ja] 編集履歴は、あるリビジョンと 1 つ前との差分を通して読まれる。同じ本文の
	// 繰り返しではどの差分も空になり、全面的に書き直した本文ではどの差分も全体を
	// 見せてしまう。
	const intro = "書き出し。"

	previous := draftRevisionBody(intro, 1)
	for version := 2; version <= 4; version++ {
		body := draftRevisionBody(intro, version)

		want := previous + fmt.Sprintf("- %d 回目の保存がこの行を足しました。\n", version)
		if body != want {
			t.Errorf("バージョン %d の本文が 1 行の追加になっていない: %q", version, body)
		}
		previous = body
	}
}

func TestDraftPageCopyUsesJapanese(t *testing.T) {
	t.Parallel()

	// These expectations intentionally stay independent of the production
	// titles and prose. Deriving them from newPageDraftSpecs would let the test
	// follow the implementation back to English instead of catching the
	// regression. None of this copy has a reason to retain English words, so an
	// ASCII letter anywhere in it is also a failure.
	//
	// [Ja] 期待値は、実装側のタイトルや本文から意図的に独立させる。
	// newPageDraftSpecs から導出すると、実装が英語へ戻ったときにテストも追従し、
	// 退行を検出できないため。ここに英単語を残す理由は無いため、ASCII の英字が
	// 1 文字でも含まれている場合も失敗とする。
	for _, tt := range []struct {
		name string
		got  string
		want string
	}{
		{name: "未公開ページの下書きのタイトル", got: unpublishedDraftTitle, want: "未公開ページの下書き"},
		{name: "履歴の長い下書きのタイトル", got: longHistoryDraftTitle, want: "履歴の長い下書き"},
	} {
		if tt.got != tt.want {
			t.Errorf("%sが %q であることを期待したが %q だった", tt.name, tt.want, tt.got)
		}
	}

	specs := newPageDraftSpecs()
	// The draft with a history carries a second marker: the edit history column
	// numbers its entries but never reports a total, so the body has to point at
	// the entry numbers as what shows the list was cut short.
	//
	// [Ja] 履歴を持つ下書きには 2 つ目のマーカーを置く。編集履歴カラムはエントリに
	// 番号を振るだけで総件数を出さないため、一覧が切られていることを示すものとして、
	// 本文はエントリの番号を指す必要がある。
	wantIntroMarkers := [][]string{
		{"一度も公開されていないページに対して書かれています"},
		{
			"編集履歴が一度に見せる件数より多く保存されています",
			"一覧の末尾が 1 番で終わっていない",
		},
		{"一度もタイトルが付けられておらず"},
	}
	if len(specs) != len(wantIntroMarkers) {
		t.Fatalf("未公開ページに付く下書きが %d 件であることを期待したが %d 件だった", len(wantIntroMarkers), len(specs))
	}
	for i, markers := range wantIntroMarkers {
		for _, marker := range markers {
			if !strings.Contains(specs[i].intro, marker) {
				t.Errorf("%d 件目の下書きの書き出しに日本語のマーカー %q が含まれていない: %q", i+1, marker, specs[i].intro)
			}
		}
		if containsASCIILetter(specs[i].intro) {
			t.Errorf("%d 件目の下書きの書き出しにASCIIの英字が含まれている: %q", i+1, specs[i].intro)
		}
	}

	publishedIntro := publishedPageDraftIntro("ハンドブック 001")
	for _, marker := range []string{"これはハンドブック 001 の未公開の編集です", "最後に公開された内容を見せたまま"} {
		if !strings.Contains(publishedIntro, marker) {
			t.Errorf("公開済みページに付く下書きの書き出しに日本語のマーカー %q が含まれていない: %q", marker, publishedIntro)
		}
	}
	if containsASCIILetter(publishedIntro) {
		t.Errorf("公開済みページに付く下書きの書き出しにASCIIの英字が含まれている: %q", publishedIntro)
	}

	revisionBody := draftRevisionBody("書き出し。", 2)
	if marker := "- 2 回目の保存がこの行を足しました。\n"; !strings.Contains(revisionBody, marker) {
		t.Errorf("下書きのリビジョン本文に日本語の追加行 %q が含まれていない: %q", marker, revisionBody)
	}
	if containsASCIILetter(revisionBody) {
		t.Errorf("下書きのリビジョン本文にASCIIの英字が含まれている: %q", revisionBody)
	}
}

func TestPublishedPageDraftIntroSpacesTheTitleByItsEdges(t *testing.T) {
	t.Parallel()

	// A space belongs where an ASCII character meets Japanese text, so
	// the same title takes one on one side and none on the other. The titles
	// here are the ones the seed publishes into the topics drafts are written
	// against.
	//
	// [Ja] 空白が要るのは ASCII 文字と日本語が接する箇所であるため、同じタイトルでも
	// 片側には空白が付き、もう片側には付かない。ここに並べたタイトルは、下書きが
	// 対象とするトピックへシードが公開するものになる。
	for _, tt := range []struct {
		title string
		want  string
	}{
		{title: "ハンドブック 001", want: "これはハンドブック 001 の未公開の編集です。"},
		{title: "リンクハブ", want: "これはリンクハブの未公開の編集です。"},
		{title: "Markdown 記法", want: "これは Markdown 記法の未公開の編集です。"},
	} {
		if got := publishedPageDraftIntro(tt.title); !strings.HasPrefix(got, tt.want) {
			t.Errorf("タイトル %q の下書きの書き出しが %q で始まることを期待したが %q だった", tt.title, tt.want, got)
		}
	}
}

func TestDraftPageBodiesCarryNoWikilinks(t *testing.T) {
	t.Parallel()

	// A wiki link in a draft body would have the resolver create the page it
	// names, adding a page to a topic whose listing counts are decided
	// elsewhere. A draft exists to show what an unpublished edit looks like,
	// not to add pages.
	//
	// [Ja] 下書きの本文に Wiki リンクがあると、resolver がその名前のページを作成し、
	// 一覧の件数を別の場所で決めているトピックにページが増える。下書きは、公開前の
	// 編集がどう見えるかを示すために存在するのであって、ページを増やすためではない。
	intros := []string{publishedPageDraftIntro(topicNameHandbook + " 001")}
	for _, spec := range newPageDraftSpecs() {
		intros = append(intros, spec.intro)
	}

	for _, intro := range intros {
		body := draftRevisionBody(intro, ordinaryDraftRevisions)
		if got := len(markup.ScanWikilinks(body, topicNameNotes)); got != 0 {
			t.Errorf("下書きの本文にWikiリンクが %d 件含まれている: %q", got, body)
		}
	}
}

func TestDefaultAmountsClearTheDraftListingLimits(t *testing.T) {
	t.Parallel()

	// The limits mirror homeDraftPagesLimit, editDraftPagesLimit and
	// editDraftRevisionsLimit, which are unexported in the usecase package. A
	// count that only reaches a limit fills its listing but leaves it unknown
	// whether anything was dropped, which is the part of the screen the seed is
	// there to make visible.
	//
	// [Ja] 上限は usecase パッケージで非公開になっている homeDraftPagesLimit・
	// editDraftPagesLimit・editDraftRevisionsLimit に合わせている。上限にちょうど
	// 届くだけの件数では一覧は埋まるが、何かが落ちたかどうかは分からないままになる。
	// そこはまさに、シードが見えるようにするために存在する部分である。
	const (
		homeDraftPagesLimit     = 5
		editDraftPagesLimit     = 20
		editDraftRevisionsLimit = 20
	)

	for _, tt := range []struct {
		name  string
		count int
		limit int
	}{
		{name: "ownerの下書きとホーム画面の上限", count: defaultAmounts.ownerDraftPages, limit: homeDraftPagesLimit},
		{name: "collaboratorの下書きとホーム画面の上限", count: defaultAmounts.collaboratorDraftPages, limit: homeDraftPagesLimit},
		{name: "ownerの下書きと編集画面の下書きカラムの上限", count: defaultAmounts.ownerDraftPages, limit: editDraftPagesLimit},
		{name: "履歴のある下書きのリビジョンと編集履歴の上限", count: defaultAmounts.draftRevisions, limit: editDraftRevisionsLimit},
	} {
		if tt.count <= tt.limit {
			t.Errorf("%sについて %d 件が上限 %d 件を超えることを期待した", tt.name, tt.count, tt.limit)
		}
	}
}

func TestGenerateDraftPagesRejectsTooFewDrafts(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	spaces := buildSeedSpaces(t, tx, "seed-draft-few")
	topics, err := generateTopics(ctx, tx, io.Discard, spaces)
	if err != nil {
		t.Fatalf("トピック生成に失敗: %v", err)
	}

	// The drafts on unpublished pages are a fixed list, so a count below its
	// length cannot be satisfied. Creating fewer of them would silently drop
	// one of the states they exist to show.
	//
	// [Ja] 未公開のページに付く下書きは固定の一覧であるため、その件数を下回る
	// 指定は満たせない。黙って減らすと、それらが見せるために存在する状態のどれかが
	// 失われる。
	amt := amounts{ownerDraftPages: len(newPageDraftSpecs()) - 1, collaboratorDraftPages: 0, draftRevisions: 2}
	if err := generateDraftPages(ctx, tx, io.Discard, amt, spaces, topics, newDraftStamps(time.Now())); err == nil {
		t.Error("下書きの件数が足りない場合にエラーになることを期待した")
	}
}

// publishPagesForDrafts publishes the pages the drafts are written against.
// The generator takes them from what earlier generators published rather than
// creating them, so a test has to leave it something to take.
//
// [Ja] publishPagesForDrafts は、下書きが対象とするページを公開する。生成器は
// これらを自分で作らず、先行する生成器が公開したものから取るため、テスト側で
// 取れるものを用意しておく必要がある。
func publishPagesForDrafts(
	ctx context.Context,
	t *testing.T,
	tx *sql.Tx,
	space *seededSpace,
	topics *seededTopics,
) {
	t.Helper()

	writer := newPageWriter(tx, space)
	for _, topic := range []*seededTopic{topics.handbook, topics.notes, topics.privateNotes} {
		for number := 1; number <= 3; number++ {
			title := fmt.Sprintf("%s %03d", topic.name, number)
			if _, err := writer.createPage(ctx, createPageInput{
				topic:  topic,
				author: space.member(roleOwner),
				title:  title,
				body:   title + " は、下書きの対象にできる公開済みのページです。\n",
			}); err != nil {
				t.Fatalf("ページ %s の作成に失敗: %v", title, err)
			}
		}
	}
}

// pageBehindDraft is the page a draft is written against. Its title is read as
// a pointer because a draft can hang on a page that has never been given one.
//
// [Ja] pageBehindDraft は、下書きが対象とするページ。タイトルをポインタで読むのは、
// 一度もタイトルを付けられていないページに下書きが付きうるため。
type pageBehindDraft struct {
	title     *string
	body      string
	published bool
}

// readPageBehindDraft reads back the page a draft is written against.
//
// [Ja] readPageBehindDraft は、下書きが対象とするページを読み戻す。
func readPageBehindDraft(
	ctx context.Context,
	t *testing.T,
	tx *sql.Tx,
	spaceID model.SpaceID,
	pageID model.PageID,
) pageBehindDraft {
	t.Helper()

	var page pageBehindDraft
	err := tx.QueryRowContext(
		ctx,
		`SELECT title, body, published_at IS NOT NULL
         FROM pages WHERE id = $1 AND space_id = $2`,
		string(pageID), string(spaceID),
	).Scan(&page.title, &page.body, &page.published)
	if err != nil {
		t.Fatalf("下書きの対象ページの取得に失敗: %v", err)
	}

	return page
}

// draftPageRow is a stored draft.
//
// [Ja] draftPageRow は保存された下書き。
type draftPageRow struct {
	id         model.DraftPageID
	pageID     model.PageID
	topicID    model.TopicID
	title      *string
	body       string
	bodyHTML   string
	modifiedAt time.Time
}

// draftPageRevisionRow is one stored revision of a draft.
//
// [Ja] draftPageRevisionRow は保存された下書きのリビジョン 1 件。
type draftPageRevisionRow struct {
	title    string
	body     string
	bodyHTML string
}

// listDraftPages reads back one member's drafts in the order the listings show
// them.
//
// [Ja] listDraftPages は、あるメンバーの下書きを、一覧が見せる順序で読み戻す。
func listDraftPages(
	ctx context.Context,
	t *testing.T,
	tx *sql.Tx,
	spaceID model.SpaceID,
	spaceMemberID model.SpaceMemberID,
) []draftPageRow {
	t.Helper()

	rows, err := tx.QueryContext(
		ctx,
		`SELECT id, page_id, topic_id, title, body, body_html, modified_at
         FROM draft_pages
         WHERE space_id = $1 AND space_member_id = $2
         ORDER BY modified_at DESC`,
		string(spaceID), string(spaceMemberID),
	)
	if err != nil {
		t.Fatalf("下書きの取得に失敗: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var drafts []draftPageRow
	for rows.Next() {
		var (
			draft   draftPageRow
			id      string
			pageID  string
			topicID string
		)
		if err := rows.Scan(
			&id, &pageID, &topicID, &draft.title, &draft.body, &draft.bodyHTML, &draft.modifiedAt,
		); err != nil {
			t.Fatalf("下書きの読み取りに失敗: %v", err)
		}

		draft.id = model.DraftPageID(id)
		draft.pageID = model.PageID(pageID)
		draft.topicID = model.TopicID(topicID)
		drafts = append(drafts, draft)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("下書きの読み取りに失敗: %v", err)
	}

	return drafts
}

// listDraftPageRevisions reads back a draft's revisions newest first, which is
// the order the edit history shows them in.
//
// [Ja] listDraftPageRevisions は、下書きのリビジョンを新しい順に読み戻す。これは
// 編集履歴が見せる順序にあたる。
func listDraftPageRevisions(
	ctx context.Context,
	t *testing.T,
	tx *sql.Tx,
	spaceID model.SpaceID,
	draftPageID model.DraftPageID,
) []draftPageRevisionRow {
	t.Helper()

	rows, err := tx.QueryContext(
		ctx,
		`SELECT title, body, body_html
         FROM draft_page_revisions
         WHERE space_id = $1 AND draft_page_id = $2
         ORDER BY created_at DESC, id DESC`,
		string(spaceID), string(draftPageID),
	)
	if err != nil {
		t.Fatalf("下書きリビジョンの取得に失敗: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var revisions []draftPageRevisionRow
	for rows.Next() {
		var revision draftPageRevisionRow
		if err := rows.Scan(&revision.title, &revision.body, &revision.bodyHTML); err != nil {
			t.Fatalf("下書きリビジョンの読み取りに失敗: %v", err)
		}
		revisions = append(revisions, revision)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("下書きリビジョンの読み取りに失敗: %v", err)
	}

	return revisions
}

// assertDraftTitle compares a stored draft title with what was asked for,
// including the absence of one.
//
// [Ja] assertDraftTitle は、保存された下書きのタイトルを、求めたものと比べる。
// タイトルが無いことも比較の対象に含める。
func assertDraftTitle(t *testing.T, draft draftPageRow, want *string) {
	t.Helper()

	switch {
	case want == nil && draft.title != nil:
		t.Errorf("タイトルの無い下書きを期待したが %q だった", *draft.title)
	case want != nil && draft.title == nil:
		t.Errorf("下書きのタイトルが %q であることを期待したが未設定だった", *want)
	case want != nil && *draft.title != *want:
		t.Errorf("下書きのタイトルが %q であることを期待したが %q だった", *want, *draft.title)
	}
}

// draftLabelOf names a draft in a failure message.
//
// [Ja] draftLabelOf は、失敗メッセージ内で下書きを名指しする。
func draftLabelOf(draft draftPageRow) string {
	if draft.title != nil {
		return fmt.Sprintf("下書き %q", *draft.title)
	}

	return "タイトル未設定の下書き"
}
