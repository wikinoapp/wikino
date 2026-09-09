package seed

import (
	"context"
	"fmt"
	"io"

	"github.com/wikinoapp/wikino/go/internal/query"
)

// bulkPageSpec describes one topic to fill with pages. The specs are built at
// call time rather than held as a package-level list because each one needs
// both the topic that was just created and the amount to create in it.
//
// [Ja] bulkPageSpec は、ページで埋めるトピック 1 件の内容。仕様をパッケージ変数の
// 一覧ではなく呼び出し時に組み立てるのは、各仕様が「直前に作成したトピック」と
// 「そこへ作る件数」の両方を必要とするため。
type bulkPageSpec struct {
	topic *seededTopic
	count int
	// authorRoles are the accounts the pages of the topic are handed round to,
	// in order. Where more than one role takes part, every other page goes to
	// the next of them, so that a home screen with pages of its own is not
	// something only one account has.
	//
	// [Ja] authorRoles は、そのトピックのページを順に担当するアカウント。複数の
	// 役割が加わるトピックでは 1 件おきに次の役割が書き手になり、自分のページが
	// 並ぶホーム画面を 1 つのアカウントだけが持つ状態にならないようにする。
	authorRoles []seedRole
}

// bulkPageBodies are the bodies the generated pages cycle through, keyed to
// nothing but the page's position. They are deliberately dull: these pages
// exist to be counted by a listing, and anything interesting written here
// would only compete with the pages that exist to be read.
//
// The bodies carry no wiki links. A link to a title nothing has written yet
// makes the resolver create that page, which would add pages the amounts do
// not account for and move the listings off the counts they were chosen for.
// Link data is the subject of its own generator.
//
// [Ja] bulkPageBodies は、生成するページが位置に応じて順に使う本文。意図的に
// 退屈な内容にしている。これらのページは一覧に数えられるために存在しており、
// ここに読ませたい内容を書いても、読ませるために存在するページと競合するだけで
// あるため。
//
// 本文は Wiki リンクを含まない。まだ書かれていないタイトルへのリンクがあると
// resolver がそのページを作成し、件数の設定が数えていないページが増えて、一覧が
// 選んだ件数からずれてしまうため。リンクのデータは専用の生成器が受け持つ。
var bulkPageBodies = []string{
	`%s は、このトピックの一覧を埋めるためにシードが作成したページの 1 つです。

ここに読むべきことはありません。大事なのは、一覧が 1 画面に収まらない件数のページを持っていて、ページ送りを試せる状態になっていることです。
`,
	`## 概要

%s は、それが並ぶ一覧に見せるものがある状態を作るために存在します。

- ページは順番に作成され、タイトルには連番が付きます。
- 連番があることで、一覧のあるページと次のページを見分けられます。
- 本文は数ページごとに繰り返します。
`,
	`%s は、段落 1 つより少しだけ長い埋め草のページです。一覧から開いたときに、空の画面に行き当たらないようにしています。

> シードが作成するページはドキュメントではありません。ページネーションのように、量が無いと確認できない画面のための量です。

このページに書いてあることは以上です。
`,
	`## 補足

%s は、シードが順に使う本文の組を締めくくるページです。

| 項目   | 内容                         |
| ------ | ---------------------------- |
| 目的   | 一覧を埋めること             |
| 内容   | 数ページごとに繰り返す       |
| リンク | 件数を保つため、張っていない |
`,
}

// generateBulkPages fills the topics whose listings need more pages than fit on
// one screen.
//
// [Ja] generateBulkPages は、一覧が 1 画面に収まらない件数を必要とするトピックを
// ページで埋める。
func generateBulkPages(
	ctx context.Context,
	dbtx query.DBTX,
	out io.Writer,
	amt amounts,
	spaces *seededSpaces,
	topics *seededTopics,
) error {
	specs := []bulkPageSpec{
		{topic: topics.handbook, count: amt.handbookPages, authorRoles: contentAuthorRoles},
		{topic: topics.privateNotes, count: amt.privateNotesPages, authorRoles: contentAuthorRoles},
		// roleCollaborator has not joined topics.secret, so pages there are
		// roleOwner's alone. Writing them as roleCollaborator would put pages on
		// the home screen of an account that cannot open them.
		//
		// [Ja] roleCollaborator は「シークレット」に参加していないため、ここのページは
		// roleOwner だけのものになる。roleCollaborator で書くと、開けないページが
		// そのアカウントのホーム画面に並んでしまう。
		{topic: topics.secret, count: amt.secretPages, authorRoles: []seedRole{roleOwner}},
	}

	total := 0
	for _, spec := range specs {
		total += spec.count
	}

	bar := newProgress(out, "ページネーション用ページ", total)
	defer bar.finish()

	writer := newPageWriter(dbtx, spaces.wiki)

	for _, spec := range specs {
		for number := 1; number <= spec.count; number++ {
			// The title is numbered within its topic, so that a page of the
			// listing can be told apart from the next one at a glance, and it
			// carries the topic name because the space-wide listing shows the
			// pages of every topic together.
			//
			// [Ja] タイトルはトピック内で採番する。一覧のあるページと次のページを
			// 一目で見分けられるようにするため。トピック名を含めるのは、スペース
			// 全体の一覧が全トピックのページを混ぜて表示するため。
			title := fmt.Sprintf("%s %03d", spec.topic.name, number)

			author, err := spaces.wiki.memberInTurn(spec.authorRoles, number)
			if err != nil {
				return err
			}

			if _, err := writer.createPage(ctx, createPageInput{
				topic:  spec.topic,
				author: author,
				title:  title,
				body:   fmt.Sprintf(bulkPageBodies[(number-1)%len(bulkPageBodies)], title),
			}); err != nil {
				return err
			}
			bar.advance()
		}
	}

	return nil
}
