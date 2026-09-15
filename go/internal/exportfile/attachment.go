package exportfile

import (
	"net/url"
	"strings"

	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// AttachmentDirNameは、トピックがページの添付ファイルを置くディレクトリ。参照する
// Markdownファイルと同じ階層に置かれる。
const AttachmentDirName = "attachments"

// RewriteAttachmentLinksはbodyの添付ファイルへの参照のうちnamesにあるものを、
// アーカイブがページの隣に持つ複製を指す形にして返す。namesは添付ファイルからトピックの
// attachmentsディレクトリでのファイル名への対応である。
//
// 本文のどこが参照かはWikiリンクと同じくinternal/markupが決めるため、本文がMarkdownの
// コードとして見せている参照はそのまま残る。書き手が見せているものは書かれたままになる。
//
// namesのどの添付ファイルも指していない参照はそのまま残す。エクスポートがその複製を持たない
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

// escapeAttachmentNameはnameをリンク先が持てる形にして返す。添付ファイルはアップロード
// 元のファイル名を保っており、そのファイル名は制限していないため、周囲のMarkdownとHTMLの
// どちらでも壊れない形にする必要がある。
//
// まずパーセントエンコードする。ファイル名は空白・"#"・"?"・括弧を含むことがあり、これらを含む
// リンク先を持つMarkdownのリンクは、そのファイルへのリンクとして読まれない。
//
// HTMLの属性値の中のリンク先は、続けてHTML用のエスケープも行う。url.PathEscapeは "&" を
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
