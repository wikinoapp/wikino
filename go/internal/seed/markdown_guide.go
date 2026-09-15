package seed

import (
	"context"
	_ "embed"
	"io"

	"github.com/wikinoapp/wikino/go/internal/query"
)

// markdownGuideBodyは、シードがレンダリングするMarkdown記法の紹介本文。
// シードは添付ファイルを作らないため、添付画像を使う例は意図的に除外している。
//
// 文字列リテラルではなくファイルから埋め込むのは、本文の大半がコードフェンスであり
// raw string literalが使えないため。通常の文字列リテラルにすると連結と
// エスケープになり、もはやMarkdownとして読めなくなる。
//
// 中身はMarkdownだがファイル名を .mdではなく .md.txtとしている。このファイルは
// リポジトリについて書かれたドキュメントではなくページの中身であり、一方で共通の
// Markdownリンタは .mdを一律に走査して除外の手段を持たない。Lintを通すために本文を
// 句点改行へ書き換えると、ページのレンダラーはハードラップを有効にしているため、その
// 改行は画面上の <br> になる。
//
//go:embed bodies/markdown-guide.md.txt
var markdownGuideBody string

// markdownGuideTitleは、上の本文を書き込むページのタイトル。
const markdownGuideTitle = "Markdown 記法"

// generateMarkdownGuideは「ノート」トピックにMarkdown記法のページを作成する。
//
// シードが通常のレンダリング経路を通して書く最初のページであり、その経路が
// 動いていることを示すページでもある。本文は、同じトピックのページ・別トピックの
// ページ・存在しないトピックへのWikiリンクを持っており、これはリンクの解決先が
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

	// エラーはそのまま返す。createPageは返すエラーのいずれにもページ名を入れ、
	// Runner.Runは生成器名を入れるため、ここでラップしても同じ文言が重なるだけに
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
