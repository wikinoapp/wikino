package seed

import (
	"context"
	"fmt"
	"io"

	"github.com/wikinoapp/wikino/go/internal/query"
)

// bulkPageSpecは、ページで埋めるトピック1件の内容。仕様をパッケージ変数の
// 一覧ではなく呼び出し時に組み立てるのは、各仕様が「直前に作成したトピック」と
// 「そこへ作る件数」の両方を必要とするため。
type bulkPageSpec struct {
	topic *seededTopic
	count int
	// authorRolesは、そのトピックのページを順に担当するアカウント。複数の
	// 役割が加わるトピックでは1件おきに次の役割が書き手になり、自分のページが
	// 並ぶホーム画面を1つのアカウントだけが持つ状態にならないようにする。
	authorRoles []seedRole
}

// bulkPageBodiesは、生成するページが位置に応じて順に使う本文。意図的に
// 退屈な内容にしている。これらのページは一覧に数えられるために存在しており、
// ここに読ませたい内容を書いても、読ませるために存在するページと競合するだけで
// あるため。
//
// 本文はWikiリンクを含まない。まだ書かれていないタイトルへのリンクがあると
// resolverがそのページを作成し、件数の設定が数えていないページが増えて、一覧が
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

// generateBulkPagesは、一覧が1画面に収まらない件数を必要とするトピックを
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
		// roleCollaboratorは「シークレット」に参加していないため、ここのページは
		// roleOwnerだけのものになる。roleCollaboratorで書くと、開けないページが
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
			// タイトルはトピック内で採番する。一覧のあるページと次のページを
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
