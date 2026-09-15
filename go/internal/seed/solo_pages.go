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

// markdownASCIIPunctuationは、CommonMarkがASCII句読点として定義している
// 文字。バックスラッシュがエスケープする集合であり、表示名を本文へ入れる前に
// エンコードして外しておくべき集合はこれになる。
const markdownASCIIPunctuation = "!\"#$%&'()*+,-./:;<=>?@[\\]^_\x60{|}~"

// soloPageSpecは、ページで埋めるseed-soloのトピック1件の内容。仕様を
// パッケージ変数の一覧ではなく呼び出し時に組み立てるのは、各仕様が「直前に作成した
// トピック」「そこへ作る件数」「書き込む本文が載せる名前」を必要とするため。
type soloPageSpec struct {
	topic *seededTopic
	count int
	// bodyは、指定のタイトルを持つページの本文を組み立てる。トピックごとに
	// 別の本文を持たせるのは、2つのトピックが非メンバーに見せるものが互いに正反対で
	// あり、それが書かれる場所が本文であるため。
	body func(title string) string
}

// generateSoloPagesはseed-soloの2つのトピックをページで埋める。
//
// このスペースは、参加していないアカウントから眺めるために存在する。ページが
// 無いと、そこから確認できるのはどのトピックが一覧に出るかだけになる。公開
// トピックは空の状態で開き、非メンバーがGuestPolicyを通って辿り着くページ詳細
// 画面はそもそも開かれない。ここのページが、その両方の答えを画面に出す。非メンバー
// が読める公開トピックのページと、非メンバーには見つからない非公開トピックのページ。
//
// 本文が名指しするのは、このスペースに参加していない2つのアカウントのうち
// roleGuestである。本文が説明している経路を通るのがそちらであるため。roleGuestは
// フィーチャーフラグを持ち、ここで開く画面にはGo版が応答する。roleCollaboratorも
// 同じく非メンバーだが、そちらにはRailsが応答する。
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
	// ここのページはすべてroleOwnerのものになる。他の役割はこのスペースに
	// 参加しておらず、spaces.soloにはページの書き手にできる他のメンバーシップが
	// 無いため。
	owner, err := spaces.solo.requireMember(roleOwner)
	if err != nil {
		return err
	}

	// 本文が読む人へ案内するアカウントは役割で求め、その求め先はスペースでは
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
			// タイトルにトピック名を含めるのは、seed-wikiの埋め草ページと
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

// soloNotesPageBodyはseed-soloの公開トピックのページの本文を組み立てる。
// nonMemberNameは、このスペースに参加していないアカウントの呼び名。
func soloNotesPageBody(title string, nonMemberName string) string {
	return fmt.Sprintf(`%s は、スペースに参加していない人でも開ける公開トピックのページです。

%s はこのスペースのメンバーではありません。それでもこのページを開いて本文を読めます。公開トピックのページを読むのに、スペースへの参加もサインインも要らないためです。

読めることと書けることは別です。メンバーではない画面には、編集の導線も新しいページを作る導線も出ません。
`, title, markdownPlainText(nonMemberName))
}

// soloSecretPageBodyはseed-soloの非公開トピックのページの本文を組み立てる。
// nonMemberNameは、このスペースに参加していないアカウントの呼び名。memberNameは
// 参加しているアカウントの呼び名。
func soloSecretPageBody(title string, nonMemberName string, memberName string) string {
	return fmt.Sprintf(`%s は、スペースに参加していない人には開けない非公開トピックのページです。

%s でこのページの URL を開くと、ページが見つからなかったときと同じ応答が返ります。隠していることと存在しないことを応答の上で区別すると、URL を試すだけでページの有無が分かってしまうためです。

このページを開けるのは、スペースに参加しトピックにも参加している %s だけです。
`, title, markdownPlainText(nonMemberName), markdownPlainText(memberName))
}

// markdownPlainTextは、MarkdownまたはWikiリンク走査が解釈し得る文字を、
// 読む人に表示される名前を保ったままエンコードする。半角スペース以外の空白も
// エンコードし、名前に含まれる改行が周囲の本文で新しいブロックを始めないようにする。
//
// 対象はMarkdownレンダラーを通るテキストであり、それだけになる。ここが作るのは
// 数値文字参照で、レンダラーがそれを元の文字へ戻す。テキストをそのまま描画する箇所
// (templを通って画面へ出るトピックの説明文など) では、参照そのものが表示される。
// wikiTopicSpecsが同じことを反対側から述べている。
//
// 本文を組み立てる2つの関数の隣に置いているのは、呼び出し元がその2つだけである
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
