package seed

import (
	"context"
	"fmt"
	"io"

	"github.com/wikinoapp/wikino/go/internal/query"
)

// soloPageSpec describes one topic of seed-solo to fill with pages. The specs
// are built at call time rather than held as a package-level list because each
// one needs both the topic that was just created and the amount to create in
// it.
//
// [Ja] soloPageSpec は、ページで埋める seed-solo のトピック 1 件の内容。仕様を
// パッケージ変数の一覧ではなく呼び出し時に組み立てるのは、各仕様が「直前に作成した
// トピック」と「そこへ作る件数」の両方を必要とするため。
type soloPageSpec struct {
	topic *seededTopic
	count int
	// body builds the body of the page with the given title. Each topic gets a
	// body of its own because what the two topics show a non-member is the
	// opposite of each other, and the body is where that is written down.
	//
	// [Ja] body は、指定のタイトルを持つページの本文を組み立てる。トピックごとに
	// 別の本文を持たせるのは、2 つのトピックが非メンバーに見せるものが互いに正反対で
	// あり、それが書かれる場所が本文であるため。
	body func(title string) string
}

// generateSoloPages fills the two topics of seed-solo.
//
// The space exists to be looked at from an account that has not joined it.
// Without pages, the only thing that can be checked from there is which topics
// are listed: the public topic opens on an empty state, and the page detail
// screen a non-member reaches through GuestPolicy is never opened at all. These
// pages are what puts both answers on the screen — a page of the public topic
// that a non-member may read, and one of the private topic that is not found for
// them.
//
// [Ja] generateSoloPages は seed-solo の 2 つのトピックをページで埋める。
//
// このスペースは、参加していないアカウントから眺めるために存在する。ページが
// 無いと、そこから確認できるのはどのトピックが一覧に出るかだけになる。公開
// トピックは空の状態で開き、非メンバーが GuestPolicy を通って辿り着くページ詳細
// 画面はそもそも開かれない。ここのページが、その両方の答えを画面に出す。非メンバー
// が読める公開トピックのページと、非メンバーには見つからない非公開トピックのページ。
func generateSoloPages(
	ctx context.Context,
	dbtx query.DBTX,
	out io.Writer,
	amt amounts,
	spaces *seededSpaces,
	topics *seededTopics,
) error {
	specs := []soloPageSpec{
		{topic: topics.soloNotes, count: amt.soloNotesPages, body: soloNotesPageBody},
		{topic: topics.soloSecret, count: amt.soloSecretPages, body: soloSecretPageBody},
	}

	total := 0
	for _, spec := range specs {
		total += spec.count
	}

	bar := newProgress(out, "個人スペースのページ", total)
	defer bar.finish()

	// Every page here belongs to roleOwner: no other role has joined the space,
	// and spaces.solo therefore holds no other membership to attribute a page to.
	//
	// [Ja] ここのページはすべて roleOwner のものになる。他の役割はこのスペースに
	// 参加しておらず、spaces.solo にはページの書き手にできる他のメンバーシップが
	// 無いため。
	owner, err := spaces.solo.requireMember(roleOwner)
	if err != nil {
		return err
	}

	writer := newPageWriter(dbtx, spaces.solo)

	for _, spec := range specs {
		for number := 1; number <= spec.count; number++ {
			// The title carries the topic name for the same reason the filler
			// pages of seed-wiki do: the space-wide listing shows the pages of
			// every topic together, and the number tells one page of a listing
			// apart from the next.
			//
			// [Ja] タイトルにトピック名を含めるのは、seed-wiki の埋め草ページと
			// 同じ理由による。スペース全体の一覧が全トピックのページを混ぜて表示し、
			// 番号が一覧のあるページと次のページを見分けさせる。
			title := fmt.Sprintf("%s %02d", spec.topic.name, number)

			if _, err := writer.createPage(ctx, createPageInput{
				topic:  spec.topic,
				author: owner,
				title:  title,
				body:   spec.body(title),
			}); err != nil {
				return err
			}
			bar.advance()
		}
	}

	return nil
}

// soloNotesPageBody builds the body of a page of the public topic of seed-solo.
//
// [Ja] soloNotesPageBody は seed-solo の公開トピックのページの本文を組み立てる。
func soloNotesPageBody(title string) string {
	return fmt.Sprintf(`%s は、スペースに参加していない人でも開ける公開トピックのページです。

シードユーザー 2 はこのスペースのメンバーではありません。それでもこのページを開いて本文を読めます。公開トピックのページを読むのに、スペースへの参加もサインインも要らないためです。

読めることと書けることは別です。メンバーではない画面には、編集の導線も新しいページを作る導線も出ません。
`, title)
}

// soloSecretPageBody builds the body of a page of the private topic of
// seed-solo.
//
// [Ja] soloSecretPageBody は seed-solo の非公開トピックのページの本文を組み立てる。
func soloSecretPageBody(title string) string {
	return fmt.Sprintf(`%s は、スペースに参加していない人には開けない非公開トピックのページです。

シードユーザー 2 でこのページの URL を開くと、ページが見つからなかったときと同じ応答が返ります。隠していることと存在しないことを応答の上で区別すると、URL を試すだけでページの有無が分かってしまうためです。

このページを開けるのは、スペースに参加しトピックにも参加しているシードユーザー 1 だけです。
`, title)
}
