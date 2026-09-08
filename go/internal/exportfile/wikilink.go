package exportfile

import (
	"strings"

	"github.com/wikinoapp/wikino/go/internal/markup"
)

// PageRef names a page the way a wiki link does: by the name of its topic and by its own title.
// It is the key of the table an export builds while it decides the entry names of the archive.
//
// [Ja] PageRef は Wiki リンクと同じ形でページを指す。トピックの名前と、ページ自身のタイトルの
// 組である。エクスポートがアーカイブのエントリ名を決めながら組み立てる対応表のキーになる。
type PageRef struct {
	TopicName string
	PageTitle string
}

// RewriteWikilinks returns body with every wiki link that names a page of paths pointing at the
// entry that page has in the archive. paths maps a page to its path from the root of the archive
// ("topic directory/page file name"), which is what the extracted directory holds.
//
// The pair of names inside a wiki link is the one the space uses, while an entry of the archive
// carries a name that replacement, deduplication and truncation have changed. Which is why the
// path is looked up rather than converted a second time from the title.
//
// A link written as [[page title]] means a page of the topic the body belongs to, given by
// currentTopicName, and comes out carrying that topic as well: a vault resolves a path from its
// root, while it has no documented rule for choosing between the notes that share a name.
//
// A link that names no page of paths is left as it is. It points at a page that does not exist or
// that the export does not carry, and there is no entry to point it at.
//
// [Ja] RewriteWikilinks は body の Wiki リンクのうち paths にあるページを指すものを、そのページが
// アーカイブで持つエントリを指す形にして返す。paths はページからアーカイブのルートを起点とした
// パス ("トピックのディレクトリ名/ページのファイル名") への対応で、これが展開したディレクトリの
// 中身になる。
//
// Wiki リンクの中にある名前の組はスペース上のものだが、アーカイブのエントリは置換・重複の解決・
// 切り詰めを経た名前を持つ。タイトルからもう一度変換するのではなくパスを引くのはこのためである。
//
// [[ページタイトル]] と書かれたリンクは、本文が属するトピック (currentTopicName) のページを
// 指しており、そのトピックも含んだ形になって出てくる。vault はルートを起点にパスを解決する一方、
// 同じ名前を持つノートのどれを選ぶかの規則は文書化されていないためである。
//
// paths のどのページも指していないリンクはそのまま残す。存在しないページか、エクスポートが
// 持たないページを指しており、向ける先のエントリが無いためである。
func RewriteWikilinks(body string, currentTopicName string, paths map[PageRef]string) string {
	matches := markup.ScanWikilinkMatches(body, currentTopicName)
	if len(matches) == 0 {
		return body
	}

	var rewritten strings.Builder
	rewritten.Grow(len(body))

	written := 0

	for _, match := range matches {
		path, ok := paths[PageRef{TopicName: match.Key.TopicName, PageTitle: match.Key.PageTitle}]
		if !ok {
			continue
		}

		rewritten.WriteString(body[written:match.Start])
		rewritten.WriteString("[[" + path + "]]")

		written = match.Stop
	}

	if written == 0 {
		return body
	}

	rewritten.WriteString(body[written:])

	return rewritten.String()
}
