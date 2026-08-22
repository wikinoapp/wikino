package seed

import (
	"context"
	_ "embed"
	"io"

	"github.com/wikinoapp/wikino/go/internal/query"
)

// markdownGuideBody contains the Markdown notation guide rendered by the seed.
// Attachment-backed image examples are intentionally excluded because the
// seed creates no attachments.
//
// It is embedded from a file rather than written as a string literal: the body
// is mostly code fences, which rules out a raw string literal, and an
// interpreted one would turn into concatenation and escapes that nobody can
// read as Markdown any more.
//
// [Ja] markdownGuideBody は、シードがレンダリングする Markdown 記法の紹介本文。
// シードは添付ファイルを作らないため、添付画像を使う例は意図的に除外している。
//
// 文字列リテラルではなくファイルから埋め込むのは、本文の大半がコードフェンスであり
// raw string literal が使えないため。通常の文字列リテラルにすると連結と
// エスケープになり、もはや Markdown として読めなくなる。
//
//go:embed bodies/markdown-guide.md
var markdownGuideBody string

// markdownGuideTitle is the title of the page the body above is written to.
//
// [Ja] markdownGuideTitle は、上の本文を書き込むページのタイトル。
const markdownGuideTitle = "Markdown 記法"

// generateMarkdownGuide creates the Markdown notation page in topics.notes.
//
// It is the first page the seed writes through the normal rendering path, so
// it is also what shows that the path works: the body carries wiki links to a
// page in its own topic, to a page in another topic, and to a topic that does
// not exist, which is the whole range of what a link can resolve to.
//
// [Ja] generateMarkdownGuide は「ノート」トピックに Markdown 記法のページを作成する。
//
// シードが通常のレンダリング経路を通して書く最初のページであり、その経路が
// 動いていることを示すページでもある。本文は、同じトピックのページ・別トピックの
// ページ・存在しないトピックへの Wiki リンクを持っており、これはリンクの解決先が
// 取りうる範囲そのものになっている。
func generateMarkdownGuide(
	ctx context.Context,
	dbtx query.DBTX,
	out io.Writer,
	spaces *seededSpaces,
	topics *seededTopics,
) error {
	bar := newProgress(out, "Markdown記法紹介ページ", 1)
	defer bar.finish()

	owner, err := spaces.wiki.requireMember(roleOwner)
	if err != nil {
		return err
	}

	writer := newPageWriter(dbtx, spaces.wiki)

	// The error is returned as it is: createPage names the page in every error
	// it returns, and Runner.Run names the generator, so wrapping here would
	// only repeat text the message already carries.
	//
	// [Ja] エラーはそのまま返す。createPage は返すエラーのいずれにもページ名を入れ、
	// Runner.Run は生成器名を入れるため、ここでラップしても同じ文言が重なるだけに
	// なる。
	if _, err := writer.createPage(ctx, createPageInput{
		topic:  topics.notes,
		author: owner,
		title:  markdownGuideTitle,
		body:   markdownGuideBody,
	}); err != nil {
		return err
	}
	bar.advance()

	return nil
}
