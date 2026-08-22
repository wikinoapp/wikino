package seed

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

	"github.com/wikinoapp/wikino/go/internal/query"
)

// markdownASCIIPunctuation are the characters CommonMark defines as ASCII
// punctuation, which is the set a backslash escapes and therefore the set a
// display name has to be encoded out of before it is put into a body.
//
// [Ja] markdownASCIIPunctuation は、CommonMark が ASCII 句読点として定義している
// 文字。バックスラッシュがエスケープする集合であり、表示名を本文へ入れる前に
// エンコードして外しておくべき集合はこれになる。
const markdownASCIIPunctuation = "!\"#$%&'()*+,-./:;<=>?@[\\]^_\x60{|}~"

// soloPageSpec describes one topic of seed-solo to fill with pages. The specs
// are built at call time rather than held as a package-level list because each
// one needs the topic that was just created, the amount to create in it, and
// the names the body it writes carries.
//
// [Ja] soloPageSpec は、ページで埋める seed-solo のトピック 1 件の内容。仕様を
// パッケージ変数の一覧ではなく呼び出し時に組み立てるのは、各仕様が「直前に作成した
// トピック」「そこへ作る件数」「書き込む本文が載せる名前」を必要とするため。
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
//
// Of the two accounts that have not joined the space, the bodies name
// roleGuest, because it is the one that walks the path they describe. It holds
// the feature flags, so the screens it opens here are answered by the Go
// version; roleCollaborator is equally a non-member, but Rails answers for it.
//
// Which account a body names is decided here, while what that account is called
// comes from the roster. A body that spelled the name out would go on saying it
// in a development environment whose roster calls the account something else.
//
// [Ja] 本文が名指しするのは、このスペースに参加していない 2 つのアカウントのうち
// roleGuest である。本文が説明している経路を通るのがそちらであるため。roleGuest は
// フィーチャーフラグを持ち、ここで開く画面には Go 版が応答する。roleCollaborator も
// 同じく非メンバーだが、そちらには Rails が応答する。
//
// 本文がどのアカウントを名指しするかはここで決まり、そのアカウントが何と呼ばれるか
// は名簿から来る。名前を書き下した本文は、名簿がそのアカウントを別の名前で呼んで
// いる開発環境でも、その名前を言い続けることになる。
func generateSoloPages(
	ctx context.Context,
	dbtx query.DBTX,
	out io.Writer,
	amt amounts,
	users *seededUsers,
	spaces *seededSpaces,
	topics *seededTopics,
) error {
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

	// The account the bodies point the reader at is asked for by role, and it is
	// asked of the accounts rather than of the space: not having joined this
	// space is the whole of what the bodies say about it.
	//
	// [Ja] 本文が読む人へ案内するアカウントは役割で求め、その求め先はスペースでは
	// なくアカウントのほうになる。このスペースに参加していないことこそが、本文が
	// そのアカウントについて述べていることのすべてであるため。
	nonMemberName, err := users.requireName(roleGuest)
	if err != nil {
		return err
	}

	specs := []soloPageSpec{
		{
			topic: topics.soloNotes,
			count: amt.soloNotesPages,
			body:  func(title string) string { return soloNotesPageBody(title, nonMemberName) },
		},
		{
			topic: topics.soloSecret,
			count: amt.soloSecretPages,
			body:  func(title string) string { return soloSecretPageBody(title, nonMemberName, owner.name) },
		},
	}

	total := 0
	for _, spec := range specs {
		total += spec.count
	}

	bar := newProgress(out, "個人スペースのページ", total)
	defer bar.finish()

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
// nonMemberName is what the account that has not joined the space is called.
//
// [Ja] soloNotesPageBody は seed-solo の公開トピックのページの本文を組み立てる。
// nonMemberName は、このスペースに参加していないアカウントの呼び名。
func soloNotesPageBody(title string, nonMemberName string) string {
	return fmt.Sprintf(`%s は、スペースに参加していない人でも開ける公開トピックのページです。

%s はこのスペースのメンバーではありません。それでもこのページを開いて本文を読めます。公開トピックのページを読むのに、スペースへの参加もサインインも要らないためです。

読めることと書けることは別です。メンバーではない画面には、編集の導線も新しいページを作る導線も出ません。
`, title, markdownPlainText(nonMemberName))
}

// soloSecretPageBody builds the body of a page of the private topic of
// seed-solo. nonMemberName is what the account that has not joined the space is
// called, and memberName the one that has.
//
// [Ja] soloSecretPageBody は seed-solo の非公開トピックのページの本文を組み立てる。
// nonMemberName は、このスペースに参加していないアカウントの呼び名。memberName は
// 参加しているアカウントの呼び名。
func soloSecretPageBody(title string, nonMemberName string, memberName string) string {
	return fmt.Sprintf(`%s は、スペースに参加していない人には開けない非公開トピックのページです。

%s でこのページの URL を開くと、ページが見つからなかったときと同じ応答が返ります。隠していることと存在しないことを応答の上で区別すると、URL を試すだけでページの有無が分かってしまうためです。

このページを開けるのは、スペースに参加しトピックにも参加している %s だけです。
`, title, markdownPlainText(nonMemberName), markdownPlainText(memberName))
}

// markdownPlainText encodes characters that Markdown or the wiki-link scanner
// could interpret, while preserving the display name rendered for the reader.
// Non-space whitespace is encoded too, so a newline in a name cannot start a
// new block in the surrounding body.
//
// It is for text that goes through the Markdown renderer, and only for that.
// What it produces are numeric character references, which the renderer turns
// back into the characters they stand for; a place that draws its text as it is
// — a topic description reaching the screen through templ, say — would show the
// references themselves. wikiTopicSpecs says the same thing from that side.
//
// It sits here, next to the two functions that build the bodies, because they
// are its only callers. A generator that comes to name an account in a body of
// its own would call it from here rather than encode the name again.
//
// [Ja] markdownPlainText は、Markdown または Wiki リンク走査が解釈し得る文字を、
// 読む人に表示される名前を保ったままエンコードする。半角スペース以外の空白も
// エンコードし、名前に含まれる改行が周囲の本文で新しいブロックを始めないようにする。
//
// 対象は Markdown レンダラーを通るテキストであり、それだけになる。ここが作るのは
// 数値文字参照で、レンダラーがそれを元の文字へ戻す。テキストをそのまま描画する箇所
// (templ を通って画面へ出るトピックの説明文など) では、参照そのものが表示される。
// wikiTopicSpecs が同じことを反対側から述べている。
//
// 本文を組み立てる 2 つの関数の隣に置いているのは、呼び出し元がその 2 つだけである
// ため。自身の本文でアカウントを名指しする生成器が現れたら、エンコードを書き直すの
// ではなくここから呼ぶことになる。
func markdownPlainText(text string) string {
	var encoded strings.Builder
	encoded.Grow(len(text))

	for _, r := range text {
		if strings.ContainsRune(markdownASCIIPunctuation, r) ||
			unicode.IsControl(r) || (unicode.IsSpace(r) && r != ' ') {
			encoded.WriteString("&#")
			encoded.WriteString(strconv.Itoa(int(r)))
			encoded.WriteByte(';')
			continue
		}

		encoded.WriteRune(r)
	}

	return encoded.String()
}
