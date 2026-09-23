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

func TestGeneratePageVariations(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	spaces := buildSeedSpaces(t, tx, "seed-variations")
	topics, err := generateTopics(ctx, tx, io.Discard, spaces)
	if err != nil {
		t.Fatalf("トピック生成に失敗: %v", err)
	}

	// 各状態2件は、ピン留めの区画に順序があることと、両方のアカウントが
	// 1回ずつ担当することを確認できる最小の件数。
	amt := amounts{pinnedPages: 2, trashedPages: 2}

	if err := generatePageVariations(ctx, tx, io.Discard, amt, spaces, topics); err != nil {
		t.Fatalf("状態バリエーションページの生成に失敗: %v", err)
	}

	// ピン留めされたページは公開済みのままである必要がある。ピン留めは
	// ページを一覧から上の区画へ移すものであり、未公開のページはそのどちらにも
	// 現れない。
	pinnedAts := make([]time.Time, 0, amt.pinnedPages)
	for number := 1; number <= amt.pinnedPages; number++ {
		title := fmt.Sprintf(pinnedPageTitleFormat, number)
		row := readPageState(ctx, t, tx, spaces.wiki.id, topics.notes.id, title)

		if row.publishedAt == nil {
			t.Errorf("%sが公開済みであることを期待したが未公開だった", title)
		}
		if row.pinnedAt == nil {
			t.Fatalf("%sがピン留めされていることを期待したがピン留めされていなかった", title)
		}
		if row.trashedAt != nil {
			t.Errorf("%sがゴミ箱に入っていないことを期待したが入っていた", title)
		}
		pinnedAts = append(pinnedAts, *row.pinnedAt)
	}

	// ピン留めの区画はpinned_atの降順で並ぶため、タイトルが先に読まれる
	// ページが、最後にピン留めされたページである必要がある。同じ時刻で打刻すると
	// 区画はページID順に並び、タイトルの番号はその順序について何も語らなくなる。
	for i := 1; i < len(pinnedAts); i++ {
		if !pinnedAts[i-1].After(pinnedAts[i]) {
			t.Errorf(
				"ピン留めの%d件目が%d件目より後にピン留めされていることを期待したが、そうではなかった (%v, %v)",
				i, i+1, pinnedAts[i-1], pinnedAts[i],
			)
		}
	}

	// ゴミ箱に入ったページは、ゴミ箱から復元できるようにタイトルと本文を
	// 保持する。どちらかを空にすると、復元するものが無くなってしまう。
	for number := 1; number <= amt.trashedPages; number++ {
		title := fmt.Sprintf(trashedPageTitleFormat, number)
		row := readPageState(ctx, t, tx, spaces.wiki.id, topics.notes.id, title)

		if row.trashedAt == nil {
			t.Errorf("%sがゴミ箱に入っていることを期待したが入っていなかった", title)
		}
		if row.publishedAt == nil {
			t.Errorf("%sが公開済みであることを期待したが未公開だった", title)
		}
		if row.body == "" {
			t.Errorf("%sの本文が残っていることを期待したが空だった", title)
		}
	}

	// 両方のアカウントが交互に担当し、どちらのホーム画面もこれらの状態の
	// ページを持たないままにならないようにする。
	for _, tt := range []struct {
		title  string
		editor *seededSpaceMember
	}{
		{title: fmt.Sprintf(pinnedPageTitleFormat, 1), editor: spaces.wiki.member(roleOwner)},
		{title: fmt.Sprintf(pinnedPageTitleFormat, 2), editor: spaces.wiki.member(roleCollaborator)},
		{title: fmt.Sprintf(trashedPageTitleFormat, 1), editor: spaces.wiki.member(roleOwner)},
		{title: fmt.Sprintf(trashedPageTitleFormat, 2), editor: spaces.wiki.member(roleCollaborator)},
	} {
		page := findPageByTitle(ctx, t, tx, spaces.wiki.id, topics.notes.id, tt.title)
		assertPageEditor(ctx, t, tx, spaces.wiki, tt.editor, page.id)
	}

	// 何も書かれていないページは、Wikiリンクが残す形をしている。タイトルが
	// あり、本文は無く、公開もされていない。どこからもリンクされていない点が、
	// リンク集中ページが名指しするページとの違いになる。
	unwritten := readPageState(ctx, t, tx, spaces.wiki.id, topics.notes.id, unwrittenPageTitle)
	if unwritten.publishedAt != nil {
		t.Errorf("%sが未公開であることを期待したが公開済みだった", unwrittenPageTitle)
	}
	if unwritten.body != "" {
		t.Errorf("%sの本文が空であることを期待したが%qだった", unwrittenPageTitle, unwritten.body)
	}
	if got := countPagesLinkingTo(ctx, t, tx, spaces.wiki.id, unwritten.id); got != 0 {
		t.Errorf("%sへのリンクが0件であることを期待したが%d件だった", unwrittenPageTitle, got)
	}

	// 空のページは、シードがタイトルを付けずに書く唯一のページであり、画面が
	// フォールバックする「無題」の表示は、このNULLだけを見て決まる。
	blank := readBlankPage(ctx, t, tx, spaces.wiki.id, topics.notes.id)
	if blank.publishedAt != nil {
		t.Error("タイトル未設定のページが未公開であることを期待したが公開済みだった")
	}
	if blank.body != "" {
		t.Errorf("タイトル未設定のページの本文が空であることを期待したが%qだった", blank.body)
	}
	assertPageEditor(ctx, t, tx, spaces.wiki, spaces.wiki.member(roleOwner), blank.id)

	// この生成器のページはすべて「ノート」に置く。見せたいページを集めている
	// トピックであるため。件数を厳密に確認しているので、誤って別のトピックへ
	// 書き込んだページは、閲覧して気づくのではなくここで捕まる。
	wantTotal := amt.pinnedPages + amt.trashedPages + 2
	if got := countPagesInTopic(ctx, t, tx, spaces.wiki.id, topics.notes.id); got != wantTotal {
		t.Errorf("「ノート」のページが%d件であることを期待したが%d件だった", wantTotal, got)
	}
	if got := countPagesInSpace(ctx, t, tx, spaces.wiki.id); got != wantTotal {
		t.Errorf("スペース全体のページが%d件であることを期待したが%d件だった", wantTotal, got)
	}
}

func TestPageVariationCopyUsesJapanese(t *testing.T) {
	t.Parallel()

	// ここでは、実装側の定数から期待値を組み立てず、利用者に見える文言を意図的に
	// 重複させる。定数の変更へ追従するテストでは、シードデータが英語へ戻ったことを
	// 検出できないため。
	wantPinnedTitle := "ピン留めページ 01"
	if got := fmt.Sprintf(pinnedPageTitleFormat, 1); got != wantPinnedTitle {
		t.Errorf("ピン留めページのタイトルが%qであることを期待したが%qだった", wantPinnedTitle, got)
	}
	if got := pinnedPageBody(wantPinnedTitle); !strings.Contains(got, wantPinnedTitle+" はピン留めされているため") {
		t.Errorf("ピン留めページの本文に日本語の説明が含まれていない: %q", got)
	}

	wantTrashedTitle := "ゴミ箱のページ 01"
	if got := fmt.Sprintf(trashedPageTitleFormat, 1); got != wantTrashedTitle {
		t.Errorf("ゴミ箱のページのタイトルが%qであることを期待したが%qだった", wantTrashedTitle, got)
	}
	if got := trashedPageBody(wantTrashedTitle); !strings.Contains(got, wantTrashedTitle+" はゴミ箱へ移されたページです") {
		t.Errorf("ゴミ箱のページの本文に日本語の説明が含まれていない: %q", got)
	}

	wantUnwrittenTitle := "未公開のページ"
	if unwrittenPageTitle != wantUnwrittenTitle {
		t.Errorf("未公開ページのタイトルが%qであることを期待したが%qだった", wantUnwrittenTitle, unwrittenPageTitle)
	}
}

func TestPageVariationBodiesCarryNoWikilinks(t *testing.T) {
	t.Parallel()

	// これらの本文にWikiリンクがあると、resolverがその名前のページを作成し、
	// トピックの一覧にページが増え、名指しした相手のバックリンク一覧にも入る。
	// これらのページはそれぞれ1つの状態を見せるためにあり、リンクはそこへ持ち込む
	// 2つ目の主題になる。
	for _, tt := range []struct {
		name string
		body string
	}{
		{name: "ピン留めページ", body: pinnedPageBody(fmt.Sprintf(pinnedPageTitleFormat, 1))},
		{name: "ゴミ箱のページ", body: trashedPageBody(fmt.Sprintf(trashedPageTitleFormat, 1))},
	} {
		if got := len(markup.ScanWikilinks(tt.body, topicNameNotes)); got != 0 {
			t.Errorf("%sの本文にWikiリンクが%d件含まれている", tt.name, got)
		}
	}
}

// pageStateRowは、ページがどの状態にあるかを語る部分の行。
type pageStateRow struct {
	id          model.PageID
	body        string
	publishedAt *time.Time
	pinnedAt    *time.Time
	trashedAt   *time.Time
}

// readPageStateは、指定タイトルのページの状態を読み戻す。
func readPageState(
	ctx context.Context,
	t *testing.T,
	tx *sql.Tx,
	spaceID model.SpaceID,
	topicID model.TopicID,
	title string,
) pageStateRow {
	t.Helper()

	var (
		row pageStateRow
		id  string
	)
	err := tx.QueryRowContext(
		ctx,
		`SELECT id, body, published_at, pinned_at, trashed_at
         FROM pages WHERE space_id = $1 AND topic_id = $2 AND title = $3`,
		string(spaceID), string(topicID), title,
	).Scan(&id, &row.body, &row.publishedAt, &row.pinnedAt, &row.trashedAt)
	if err != nil {
		t.Fatalf("ページ%sの取得に失敗: %v", title, err)
	}

	row.id = model.PageID(id)

	return row
}

// readBlankPageは、トピック内のタイトルを持たないページを読み戻す。ここの
// 他の検索はいずれもタイトルを頼りにしているため、タイトルを持たないページは
// 「そういうページが1つしか無いこと」を頼りに見つける。
func readBlankPage(
	ctx context.Context,
	t *testing.T,
	tx *sql.Tx,
	spaceID model.SpaceID,
	topicID model.TopicID,
) pageStateRow {
	t.Helper()

	var (
		row pageStateRow
		id  string
	)
	err := tx.QueryRowContext(
		ctx,
		`SELECT id, body, published_at, pinned_at, trashed_at
         FROM pages WHERE space_id = $1 AND topic_id = $2 AND title IS NULL`,
		string(spaceID), string(topicID),
	).Scan(&id, &row.body, &row.publishedAt, &row.pinnedAt, &row.trashedAt)
	if err != nil {
		t.Fatalf("タイトル未設定のページの取得に失敗: %v", err)
	}

	row.id = model.PageID(id)

	return row
}
