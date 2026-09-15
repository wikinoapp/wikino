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

// 表示名には、本文が通してはならないものを載せる。Wikiリンク、強調記号、
// 見出し記号、改行、HTMLがエスケープする文字。本文を対象とする2つのテストは
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

	// 件数は小さくし、2つを互いに異なる値にする。取り違えたトピックから
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
			t.Errorf("トピック%sのページが%d件であることを期待したが%d件だった", tt.topic.name, tt.want, got)
		}
	}

	// 生成したページがseed-soloのすべてのページになる。本文にWikiリンクが
	// あるとresolverが件数の設定を超えるページを作成し、上の件数が本文を編集する
	// たびにずれていく。
	wantTotal := amt.soloNotesPages + amt.soloSecretPages
	if got := countPagesInSpace(ctx, t, tx, spaces.solo.id); got != wantTotal {
		t.Errorf("スペース全体のページが%d件であることを期待したが%d件だった", wantTotal, got)
	}

	// seed-wikiのページは、あのスペースの一覧の件数を選ぶ基準になっている
	// ため、この生成器はそこへ何も書いてはならない。
	if got := countPagesInSpace(ctx, t, tx, spaces.wiki.id); got != 0 {
		t.Errorf("seed-wikiのページが0件であることを期待したが%d件だった", got)
	}

	// すべてのページがroleOwnerのもので、公開済みである。seed-soloにはページの
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
				t.Errorf("ページ%sが公開済みであることを期待したが未公開だった", title)
			}
			assertPageEditor(ctx, t, tx, spaces.solo, spaces.solo.member(roleOwner), page.id)

			row := readPage(ctx, t, tx, spaces.solo.id, page.id)
			if !strings.Contains(row.body, title) {
				t.Errorf("%sの本文に自身のタイトルが含まれていない", title)
			}
			assertLinkedPageIDs(t, row.linkedPageIDs, nil)
		}
	}

	assertSoloPagesReachableFromOutside(ctx, t, tx, spaces.solo, topics)
	assertSoloPageBodiesNameTheAccountThatWalksThem(ctx, t, tx, users, spaces.solo, topics)
}

func TestSoloPageBodiesCarryNoWikilinks(t *testing.T) {
	t.Parallel()

	// Wikiリンクがあるとresolverがその名前のページを作成するため、生成の
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
			t.Errorf("%sの本文にWikiリンクが%d件含まれている", name, got)
		}
	}
}

// assertSoloPageBodiesNameTheAccountThatWalksThemは、本文が読む人へ案内する
// アカウントが、本文の説明している経路を通るアカウントであることを確認する。
// roleCollaboratorとroleGuestはどちらもseed-soloの外にいるが、フィーチャー
// フラグを持つのはroleGuestだけであり、これらの画面へGo版が応答する形で辿り着ける
// のもそちらだけになる。roleCollaboratorはRailsへ送られ、それはこれらのページが
// 見せようとしているものではない。
//
// 本文をここで組み立てず、データベースから読み直すのは、検査対象が「生成器がどの
// アカウントへ手を伸ばしたか」であるため。本文は自身が載せる名前を渡される側であり、
// ここでroleGuestの名前を渡して組み立てても、その名前がテキストへ届いたこと以上の
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
		// wantMemberNamedは、誰がそのページを開けるのかを述べる本文に立てる。
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
			t.Errorf("%sの本文が%s (%s) を案内することを期待したが名指ししていない", title, guestName, roleGuest)
		}
		if strings.Contains(bodyText, collaboratorName) {
			t.Errorf("%sの本文が%s (%s) を名指ししている", title, collaboratorName, roleCollaborator)
		}
		if tt.wantMemberNamed && !strings.Contains(bodyText, ownerText) {
			t.Errorf("%sの本文がページを開けるアカウント%s (%s) を名指ししていない", title, ownerName, roleOwner)
		}
	}
}

// assertSoloPagesReachableFromOutsideは、これらのページが作り出そうとしている
// 2つの答えを確認する。非メンバーは公開トピックのページを開けて、非公開トピックの
// ページは非メンバーには見つからない。
//
// 確認は、ページ詳細画面がゲストに対して行う判定に揃える。まず
// GuestPolicy.CanShowTopicでトピックを通し、GuestPolicy.CanShowTrashがfalseを返す
// ためゴミ箱のページを拒否する。公開状態は一覧のために上で別途確認しており、ページ
// 詳細自体はpublished_atで絞り込まない。
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
			t.Fatalf("トピック%sの公開範囲の取得に失敗: %v", tt.topic.name, err)
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
			t.Fatalf("トピック%sの開けるページ数の取得に失敗: %v", tt.topic.name, err)
		}

		if got := canShowTopic && openable > 0; got != tt.wantOpen {
			t.Errorf(
				"非メンバーがトピック%sのページを開けるかが%tであることを期待したが%tだった",
				tt.topic.name, tt.wantOpen, got,
			)
		}
	}
}

// TestMarkdownPlainTextは、エンコードが何を残し何を外すのかを述べる。これが
// 働かなくなったときに壊れるもの、すなわちresolverが辿るWikiリンクと、名簿の
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
			// 名簿が実際に持つ名前はそのまま通る。変わった名前でない限り、
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
			// アンパサンドは他の何かに使われる前にエンコードされる。既に参照の
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
			// タブは空白だが、語を区切る半角スペースではない。そのためタブは
			// エンコードされ、隣の半角スペースは残る。
			name: "半角スペースは残しタブはエンコードする",
			text: "姓 名\t敬称",
			want: "姓 名&#9;敬称",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := markdownPlainText(tt.text); got != tt.want {
				t.Errorf("%qのエンコード結果が%qであることを期待したが%qだった", tt.text, tt.want, got)
			}
		})
	}
}
