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

// The display names carry what a body must not let through: a wiki link, an
// emphasis marker, a heading marker, a newline, and characters HTML escapes.
// Both tests of the bodies read them from here, so that a character added to
// the set is seen by each of them.
//
// [Ja] 表示名には、本文が通してはならないものを載せる。Wiki リンク、強調記号、
// 見出し記号、改行、HTML がエスケープする文字。本文を対象とする 2 つのテストは
// どちらもここから読むため、集合へ文字を足せばその両方が見ることになる。
const (
	markupHeavyGuestName = "閲覧者 [[余分なページ]] *強調*"
	markupHeavyOwnerName = "管理者\n# 見出し & <タグ>"
)

func TestGenerateSoloPages(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	users, spaces := buildSeedUsersAndSpaces(t, tx, "seed-solo-pages")
	users.user(roleGuest).Name = markupHeavyGuestName
	users.user(roleOwner).Name = markupHeavyOwnerName
	spaces.solo.member(roleOwner).name = markupHeavyOwnerName

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

	if err := generateSoloPages(ctx, tx, io.Discard, amt, users, spaces, topics); err != nil {
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
			assertLinkedPageIDs(t, row.linkedPageIDs, nil)
		}
	}

	assertSoloPagesReachableFromOutside(ctx, t, tx, spaces.solo, topics)
	assertSoloPageBodiesNameTheAccountThatWalksThem(ctx, t, tx, users, spaces.solo, topics)
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
		topicNameSoloNotes: soloNotesPageBody(
			topicNameSoloNotes+" 01",
			markupHeavyGuestName,
		),
		topicNameSoloSecret: soloSecretPageBody(
			topicNameSoloSecret+" 01",
			markupHeavyGuestName,
			markupHeavyOwnerName,
		),
	} {
		if got := len(markup.ScanWikilinks(body, name)); got != 0 {
			t.Errorf("%s の本文にWikiリンクが %d 件含まれている", name, got)
		}
	}
}

// assertSoloPageBodiesNameTheAccountThatWalksThem checks that the account the
// bodies point the reader at is the one whose path they describe. Both
// roleCollaborator and roleGuest are outside seed-solo, but only roleGuest holds
// the feature flags, so it alone reaches these screens as the Go version answers
// them; roleCollaborator is sent to Rails, which is not what these pages were
// written to show.
//
// The bodies are read back from the database rather than built here, because
// what is under test is the account the generator reached for. The bodies
// themselves are handed the names they carry, and building one here with the
// name of roleGuest would say nothing more than that the name reached the text.
//
// [Ja] assertSoloPageBodiesNameTheAccountThatWalksThem は、本文が読む人へ案内する
// アカウントが、本文の説明している経路を通るアカウントであることを確認する。
// roleCollaborator と roleGuest はどちらも seed-solo の外にいるが、フィーチャー
// フラグを持つのは roleGuest だけであり、これらの画面へ Go 版が応答する形で辿り着ける
// のもそちらだけになる。roleCollaborator は Rails へ送られ、それはこれらのページが
// 見せようとしているものではない。
//
// 本文をここで組み立てず、データベースから読み直すのは、検査対象が「生成器がどの
// アカウントへ手を伸ばしたか」であるため。本文は自身が載せる名前を渡される側であり、
// ここで roleGuest の名前を渡して組み立てても、その名前がテキストへ届いたこと以上の
// ことは言えない。
func assertSoloPageBodiesNameTheAccountThatWalksThem(
	ctx context.Context,
	t *testing.T,
	tx *sql.Tx,
	users *seededUsers,
	solo *seededSpace,
	topics *seededTopics,
) {
	t.Helper()

	guestName := users.user(roleGuest).Name
	collaboratorName := users.user(roleCollaborator).Name
	ownerName := users.user(roleOwner).Name

	for _, tt := range []struct {
		topic *seededTopic
		// wantMemberNamed is set for the body that says who may open the page.
		// Only the private topic has an answer to that: a page of the public one
		// is open to anybody, member or not.
		//
		// [Ja] wantMemberNamed は、誰がそのページを開けるのかを述べる本文に立てる。
		// その答えを持つのは非公開トピックだけになる。公開トピックのページは、
		// メンバーかどうかを問わず誰にでも開くため。
		wantMemberNamed bool
	}{
		{topic: topics.soloNotes},
		{topic: topics.soloSecret, wantMemberNamed: true},
	} {
		title := fmt.Sprintf("%s %02d", tt.topic.name, 1)
		page := findPageByTitle(ctx, t, tx, solo.id, tt.topic.id, title)
		bodyText := markup.PlainText(readPage(ctx, t, tx, solo.id, page.id).bodyHTML, 0)
		guestText := strings.Join(strings.Fields(guestName), " ")
		ownerText := strings.Join(strings.Fields(ownerName), " ")

		if !strings.Contains(bodyText, guestText) {
			t.Errorf("%s の本文が %s (%s) を案内することを期待したが名指ししていない", title, guestName, roleGuest)
		}
		if strings.Contains(bodyText, collaboratorName) {
			t.Errorf("%s の本文が %s (%s) を名指ししている", title, collaboratorName, roleCollaborator)
		}
		if tt.wantMemberNamed && !strings.Contains(bodyText, ownerText) {
			t.Errorf("%s の本文がページを開けるアカウント %s (%s) を名指ししていない", title, ownerName, roleOwner)
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

// TestMarkdownPlainText states what the encoding leaves alone and what it takes
// out. The bodies are checked elsewhere for the two things that would break if
// this stopped working — a wiki link the resolver would follow, and a rendered
// text that no longer reads as the roster's name — while what is written down
// here is the rule those checks rest on.
//
// [Ja] TestMarkdownPlainText は、エンコードが何を残し何を外すのかを述べる。これが
// 働かなくなったときに壊れるもの、すなわち resolver が辿る Wiki リンクと、名簿の
// 名前として読めなくなったレンダリング後のテキストは別の場所で確認しており、ここに
// 書くのは、それらの確認が拠って立つ規則になる。
func TestMarkdownPlainText(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name string
		text string
		want string
	}{
		{
			// The names a roster actually holds pass through untouched, so a
			// body reads as it was written for every name but an odd one.
			//
			// [Ja] 名簿が実際に持つ名前はそのまま通る。変わった名前でない限り、
			// 本文は書かれたとおりに読める。
			name: "句読点を含まない名前は1文字も変換しない",
			text: "シードユーザー 1",
			want: "シードユーザー 1",
		},
		{
			name: "Wikiリンクの角括弧をエンコードする",
			text: "[[ページ]]",
			want: "&#91;&#91;ページ&#93;&#93;",
		},
		{
			// The ampersand is encoded before anything else can use it, so a
			// name that already looks like a reference is shown as itself
			// rather than decoded a second time.
			//
			// [Ja] アンパサンドは他の何かに使われる前にエンコードされる。既に参照の
			// 形をしている名前も、二度目のデコードを受けずにそのまま表示される。
			name: "アンパサンド自身をエンコードして二重デコードを防ぐ",
			text: "&#91;",
			want: "&#38;&#35;91&#59;",
		},
		{
			name: "改行をエンコードして本文のブロックを断ち切らせない",
			text: "上\n下",
			want: "上&#10;下",
		},
		{
			// A tab is whitespace but not the space that separates words, so it
			// is encoded while the space beside it stays.
			//
			// [Ja] タブは空白だが、語を区切る半角スペースではない。そのためタブは
			// エンコードされ、隣の半角スペースは残る。
			name: "半角スペースは残しタブはエンコードする",
			text: "姓 名\t敬称",
			want: "姓 名&#9;敬称",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := markdownPlainText(tt.text); got != tt.want {
				t.Errorf("%q のエンコード結果が %q であることを期待したが %q だった", tt.text, tt.want, got)
			}
		})
	}
}
