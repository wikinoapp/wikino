package seed

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/wikinoapp/wikino/go/internal/query"
)

// リンクが集中するページ群を組み立てるためのタイトル。定数にしているのは、
// これらのページが本文で互いを名指しするため。Wikiリンクは対象のタイトルを
// そのとおりに書く必要があり、書き損じても失敗にはならず、意図した相手へ
// リンクする代わりに別のページを作ってしまう。
const (
	linkHubTitle              = "リンクハブ"
	linkTargetTitleFormat     = "リンク先 %02d"
	hubBacklinkTitleFormat    = "ハブ被リンク %02d"
	nestedBacklinkTitleFormat = "ネスト被リンク %02d"
)

// linkSourceSpecは、どこかへリンクするために存在するページ群1組の内容。
// 2つの組は件数・名前・リンク先しか違わないため、この仕様を回す1つのループで
// 生成する。
type linkSourceSpec struct {
	count       int
	titleFormat string
	// bodyは自身のタイトルからページ本文を組み立てる。組ごとにリンク先が
	// 異なるため、リンクはリンク先のタイトルを受け取るのではなく、その組自身の
	// ビルダーが書く。
	body func(title string) string
}

// generateLinkHubは、リンク一覧・バックリンク一覧・ネストしたバックリンク
// 一覧のいずれもが1ページに収まらない件数を持つページ群を作成する。
//
// ページ本文の下に並ぶ3つの一覧はそれぞれ独立に、しかも異なる件数でページングする。
// あらゆる向きのリンクを持つページを1つ置くことで、3つすべてを1画面から
// 辿れるようになる。
func generateLinkHub(
	ctx context.Context,
	dbtx query.DBTX,
	out io.Writer,
	amt amounts,
	spaces *seededSpaces,
	topics *seededTopics,
) error {
	// リンク先ページはここでは数えない。ハブの本文をレンダリングする過程で
	// resolverが作成するため、その全体がハブを作成する1ステップの仕事に
	// あたる。
	bar := newProgress(out, "リンク集中ページ", 1+amt.linkHubBacklinks+amt.nestedBacklinks)
	defer bar.finish()

	owner, err := spaces.wiki.requireMember(roleOwner)
	if err != nil {
		return err
	}

	writer := newPageWriter(dbtx, spaces.wiki)

	if _, err := writer.createPage(ctx, createPageInput{
		topic:  topics.notes,
		author: owner,
		title:  linkHubTitle,
		body:   linkHubBody(amt.linkHubTargets),
	}); err != nil {
		return err
	}
	bar.advance()

	// ネストしたバックリンクは、リンク一覧が最初に見せるリンク先に付ける。
	// リンク一覧をページ送りしてそこへ辿り着かなくても、ネストした一覧の
	// ページ送りを確認できるようにするため。そのリンク先はハブの本文が最後に
	// 名指しするページになる。リンク一覧の並び順はmodified_atの降順であり、
	// resolverはリンク先を書かれた順に作成するため。
	nestedTarget := fmt.Sprintf(linkTargetTitleFormat, amt.linkHubTargets)

	specs := []linkSourceSpec{
		{
			count:       amt.linkHubBacklinks,
			titleFormat: hubBacklinkTitleFormat,
			body:        hubBacklinkBody,
		},
		{
			count:       amt.nestedBacklinks,
			titleFormat: nestedBacklinkTitleFormat,
			body:        func(title string) string { return nestedBacklinkBody(title, nestedTarget) },
		},
	}

	for _, spec := range specs {
		for number := 1; number <= spec.count; number++ {
			title := fmt.Sprintf(spec.titleFormat, number)

			// アカウントは順に担当する。一覧を埋めるページと同じで、自分の
			// ページが並ぶホーム画面を1つのアカウントだけが持つ状態にしないため。
			author, err := spaces.wiki.memberInTurn(contentAuthorRoles, number)
			if err != nil {
				return err
			}

			if _, err := writer.createPage(ctx, createPageInput{
				topic:  topics.notes,
				author: author,
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

// linkHubBodyはハブの本文を組み立てる。このページが何のためにあるのかの
// 短い説明に続けて、リンク先1件につき1つのWikiリンクを並べる。
//
// 本文は見出しを持たない。ページ画面は本文の上にページタイトルをH1として置き、
// 本文の下にリンク一覧を「リンク」の見出しで描画する。このページ自身やその
// リンクを名指す見出しをここに置くと、同じ文字列が本文の上と下に2回並ぶことになる。
func linkHubBody(targets int) string {
	var b strings.Builder

	b.WriteString(`下に並ぶページはすべてこのページからリンクしており、それがこのページのリンク一覧に並んでいる理由です。一覧が一度に見せる件数より多く用意してあるため、ページ送りを試せます。

これらのページはまだ書かれていません。誰も書いていないタイトルへの Wiki リンクは、その名前のページを未公開のまま作成します。ここに並ぶページはすべてそうして作られたものです。

最後に並ぶページが、リンク一覧が最初に見せるページです。一覧の並び順が最終更新の新しい順であるためです。そのページにはさらにそこへリンクするページが付いており、その下にネストして並ぶ一覧もページ送りが必要な件数になっています。

`)

	for number := 1; number <= targets; number++ {
		b.WriteString("- [[" + fmt.Sprintf(linkTargetTitleFormat, number) + "]]\n")
	}

	return b.String()
}

// hubBacklinkBodyは、ハブへリンクするページの本文を組み立てる。
func hubBacklinkBody(title string) string {
	return fmt.Sprintf(`%s は [[%s]] へリンクするために存在するページです。

このようなページを、ハブのバックリンク一覧が一度に見せる件数より多く用意しています。
`, title, linkHubTitle)
}

// nestedBacklinkBodyは、ハブのリンク先の1つへリンクするページの本文を
// 組み立てる。ハブはWikiリンクではなくプレーンテキストで名指しする。リンクに
// すると、このページがハブ自身のバックリンク一覧にも入ってしまい、あの一覧は
// 件数を数えているため。
func nestedBacklinkBody(title string, target string) string {
	return fmt.Sprintf(`%s は、リンクハブがリンクしているページの 1 つである [[%s]] へリンクするために存在するページです。

リンクハブはそのページを一覧に並べ、その下にそこへリンクするページを見せます。このようなページが、ネストした一覧を一度に見せる件数より多くしています。
`, title, target)
}
