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
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGenerateSoloPages(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	spaces := buildSeedSpaces(t, tx, "seed-solo-pages")
	topics, err := generateTopics(ctx, tx, io.Discard, spaces)
	if err != nil {
		t.Fatalf("トピック生成に失敗: %v", err)
	}

	// Small amounts, kept different from each other so that a count read from
	// the wrong topic shows up as a mismatch rather than passing by chance.
	//
	// [Ja] 件数は小さくし、2 つを互いに異なる値にする。取り違えたトピックから
	// 数えた件数が、偶然一致して通り抜けるのではなく不一致として現れるようにする
	// ため。
	amt := amounts{soloNotesPages: 3, soloSecretPages: 2}

	if err := generateSoloPages(ctx, tx, io.Discard, amt, spaces, topics); err != nil {
		t.Fatalf("個人スペースのページの生成に失敗: %v", err)
	}

	for _, tt := range []struct {
		topic *seededTopic
		want  int
	}{
		{topic: topics.soloNotes, want: amt.soloNotesPages},
		{topic: topics.soloSecret, want: amt.soloSecretPages},
	} {
		if got := countPagesInTopic(ctx, t, tx, spaces.solo.id, tt.topic.id); got != tt.want {
			t.Errorf("トピック %s のページが %d 件であることを期待したが %d 件だった", tt.topic.name, tt.want, got)
		}
	}

	// The generated pages are the only pages of seed-solo. A body carrying a
	// wiki link would have the resolver create pages beyond the amounts, and
	// the counts above would drift with every body edit.
	//
	// [Ja] 生成したページが seed-solo のすべてのページになる。本文に Wiki リンクが
	// あると resolver が件数の設定を超えるページを作成し、上の件数が本文を編集する
	// たびにずれていく。
	wantTotal := amt.soloNotesPages + amt.soloSecretPages
	if got := countPagesInSpace(ctx, t, tx, spaces.solo.id); got != wantTotal {
		t.Errorf("スペース全体のページが %d 件であることを期待したが %d 件だった", wantTotal, got)
	}

	// The pages of seed-wiki are what the listing counts of that space are
	// chosen against, so this generator must not write anything into it.
	//
	// [Ja] seed-wiki のページは、あのスペースの一覧の件数を選ぶ基準になっている
	// ため、この生成器はそこへ何も書いてはならない。
	if got := countPagesInSpace(ctx, t, tx, spaces.wiki.id); got != 0 {
		t.Errorf("seed-wikiのページが 0 件であることを期待したが %d 件だった", got)
	}

	// Every page belongs to roleOwner and is published. seed-solo has no other
	// membership to attribute a page to, and an unpublished page is not listed
	// at all, so it would show a non-member nothing.
	//
	// [Ja] すべてのページが roleOwner のもので、公開済みである。seed-solo にはページの
	// 書き手にできる他のメンバーシップが無く、未公開のページはそもそも一覧に
	// 並ばないため、非メンバーには何も見せられない。
	for _, tt := range []struct {
		topic *seededTopic
		count int
	}{
		{topic: topics.soloNotes, count: amt.soloNotesPages},
		{topic: topics.soloSecret, count: amt.soloSecretPages},
	} {
		for number := 1; number <= tt.count; number++ {
			title := fmt.Sprintf("%s %02d", tt.topic.name, number)
			page := findPageByTitle(ctx, t, tx, spaces.solo.id, tt.topic.id, title)
			if !page.published {
				t.Errorf("ページ %s が公開済みであることを期待したが未公開だった", title)
			}
			assertPageEditor(ctx, t, tx, spaces.solo, spaces.solo.member(roleOwner), page.id)

			row := readPage(ctx, t, tx, spaces.solo.id, page.id)
			if !strings.Contains(row.body, title) {
				t.Errorf("%s の本文に自身のタイトルが含まれていない", title)
			}
		}
	}

	assertSoloPagesReachableFromOutside(ctx, t, tx, spaces.solo, topics)
}

func TestSoloPageBodiesCarryNoWikilinks(t *testing.T) {
	t.Parallel()

	// A wiki link would make the resolver create the page it names, so the
	// counts the generation test checks would drift with every body edit.
	// Catching it here says why, which a count mismatch would not.
	//
	// [Ja] Wiki リンクがあると resolver がその名前のページを作成するため、生成の
	// テストが確認する件数が本文を編集するたびにずれていく。ここで捕まえれば理由まで
	// 示せる。件数が合わないだけでは示せない。
	for name, body := range map[string]string{
		topicNameSoloNotes:  soloNotesPageBody(topicNameSoloNotes + " 01"),
		topicNameSoloSecret: soloSecretPageBody(topicNameSoloSecret + " 01"),
	} {
		if got := len(markup.ScanWikilinks(body, name)); got != 0 {
			t.Errorf("%s の本文にWikiリンクが %d 件含まれている", name, got)
		}
	}
}

// assertSoloPagesReachableFromOutside checks the pair of answers these pages
// exist to produce: a non-member may open a page of the public topic, and a
// page of the private topic is not found for them.
//
// The check mirrors what the page detail screen decides for a guest: it admits
// the topic through GuestPolicy.CanShowTopic and then refuses a trashed page
// because GuestPolicy.CanShowTrash returns false. Publication is checked
// separately above for the listings; the page detail itself does not filter on
// published_at.
//
// [Ja] assertSoloPagesReachableFromOutside は、これらのページが作り出そうとしている
// 2 つの答えを確認する。非メンバーは公開トピックのページを開けて、非公開トピックの
// ページは非メンバーには見つからない。
//
// 確認は、ページ詳細画面がゲストに対して行う判定に揃える。まず
// GuestPolicy.CanShowTopic でトピックを通し、GuestPolicy.CanShowTrash が false を返す
// ためゴミ箱のページを拒否する。公開状態は一覧のために上で別途確認しており、ページ
// 詳細自体は published_at で絞り込まない。
func assertSoloPagesReachableFromOutside(
	ctx context.Context,
	t *testing.T,
	tx *sql.Tx,
	solo *seededSpace,
	topics *seededTopics,
) {
	t.Helper()

	for _, tt := range []struct {
		topic    *seededTopic
		wantOpen bool
	}{
		{topic: topics.soloNotes, wantOpen: true},
		{topic: topics.soloSecret, wantOpen: false},
	} {
		var visibility int32
		err := tx.QueryRowContext(
			ctx,
			`SELECT visibility FROM topics WHERE id = $1 AND space_id = $2`,
			string(tt.topic.id), string(solo.id),
		).Scan(&visibility)
		if err != nil {
			t.Fatalf("トピック %s の公開範囲の取得に失敗: %v", tt.topic.name, err)
		}

		canShowTopic := policy.NewGuestPolicy().CanShowTopic(
			&model.Topic{Visibility: model.TopicVisibility(visibility)},
		)

		var openable int
		err = tx.QueryRowContext(
			ctx,
			`SELECT count(*) FROM pages
             WHERE space_id = $1 AND topic_id = $2 AND trashed_at IS NULL`,
			string(solo.id), string(tt.topic.id),
		).Scan(&openable)
		if err != nil {
			t.Fatalf("トピック %s の開けるページ数の取得に失敗: %v", tt.topic.name, err)
		}

		if got := canShowTopic && openable > 0; got != tt.wantOpen {
			t.Errorf(
				"非メンバーがトピック %s のページを開けるかが %t であることを期待したが %t だった",
				tt.topic.name, tt.wantOpen, got,
			)
		}
	}
}
