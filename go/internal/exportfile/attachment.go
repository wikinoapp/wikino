package exportfile

import (
	"net/url"
	"strings"

	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// AttachmentDirName is the directory a topic keeps the attachments of its pages in, next to the
// Markdown files that reference them.
//
// [Ja] AttachmentDirName は、トピックがページの添付ファイルを置くディレクトリ。参照する
// Markdown ファイルと同じ階層に置かれる。
const AttachmentDirName = "attachments"

// RewriteAttachmentLinks returns body with every reference to an attachment of names pointing at
// the copy the archive carries beside the page. names maps an attachment to the file name it has
// in the attachments directory of the topic.
//
// Which references the body holds is decided by internal/markup, the way it is for wiki links, so
// a reference the body shows as Markdown code is left alone: what the author is showing stays as
// written.
//
// A reference to an attachment that names none of names is left as it is, since the export does
// not carry a copy of it.
//
// [Ja] RewriteAttachmentLinks は body の添付ファイルへの参照のうち names にあるものを、
// アーカイブがページの隣に持つ複製を指す形にして返す。names は添付ファイルからトピックの
// attachments ディレクトリでのファイル名への対応である。
//
// 本文のどこが参照かは Wiki リンクと同じく internal/markup が決めるため、本文が Markdown の
// コードとして見せている参照はそのまま残る。書き手が見せているものは書かれたままになる。
//
// names のどの添付ファイルも指していない参照はそのまま残す。エクスポートがその複製を持たない
// ためである。
func RewriteAttachmentLinks(body string, names map[model.AttachmentID]string) string {
	matches := markup.ScanAttachmentRefMatches(body)
	if len(matches) == 0 {
		return body
	}

	var rewritten strings.Builder
	rewritten.Grow(len(body))

	written := 0

	for _, match := range matches {
		name, ok := names[match.AttachmentID]
		if !ok {
			continue
		}

		rewritten.WriteString(body[written:match.Start])
		rewritten.WriteString(AttachmentDirName + "/" + escapeAttachmentName(name, match.InHTMLAttribute))

		written = match.Stop
	}

	if written == 0 {
		return body
	}

	rewritten.WriteString(body[written:])

	return rewritten.String()
}

// escapeAttachmentName returns name in the form a destination can carry it. An attachment keeps
// the name of the file it was uploaded from, which is not restricted, so the name has to be made
// to survive both the Markdown and the HTML around it.
//
// Percent encoding comes first: a name may hold a space, a "#", a "?" or a parenthesis, and a
// Markdown link whose destination holds one of those does not read as a link to the file.
//
// A destination inside an HTML attribute is then escaped for HTML as well. url.PathEscape leaves
// "&" as it is, and a name such as "A&times.png" would otherwise reach the reader as "A×.png",
// which names no file of the archive. It also leaves "=", which must be encoded for an unquoted
// attribute value. The same encoding is valid in quoted attributes, so no quote-style change is needed.
//
// [Ja] escapeAttachmentName は name をリンク先が持てる形にして返す。添付ファイルはアップロード
// 元のファイル名を保っており、そのファイル名は制限していないため、周囲の Markdown と HTML の
// どちらでも壊れない形にする必要がある。
//
// まずパーセントエンコードする。ファイル名は空白・"#"・"?"・括弧を含むことがあり、これらを含む
// リンク先を持つ Markdown のリンクは、そのファイルへのリンクとして読まれない。
//
// HTML の属性値の中のリンク先は、続けて HTML 用のエスケープも行う。url.PathEscape は "&" を
// そのまま残すため、"A&times.png" のような名前は読み手に "A×.png" として届き、アーカイブの
// どのファイルも指さなくなる。残される "=" も、引用符のない属性値ではエンコードが必要である。
// 引用符のある属性でも同じエンコードが有効なので、引用符の形式を変更する必要はない。
func escapeAttachmentName(name string, inHTMLAttribute bool) string {
	escaped := url.PathEscape(name)
	if !inHTMLAttribute {
		return escaped
	}

	escaped = strings.ReplaceAll(escaped, "=", "%3D")
	return strings.ReplaceAll(escaped, "&", "&amp;")
}
