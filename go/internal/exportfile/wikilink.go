package exportfile

import (
	"strings"

	"github.com/wikinoapp/wikino/go/internal/markup"
)

// PageRefはWikiリンクと同じ形でページを指す。トピックの名前と、ページ自身のタイトルの
// 組である。エクスポートがアーカイブのエントリ名を決めながら組み立てる対応表のキーになる。
type PageRef struct {
	TopicName string
	PageTitle string
}

// RewriteWikilinksはbodyのWikiリンクのうちpathsにあるページを指すものを、そのページが
// アーカイブで持つエントリを指す形にして返す。pathsはページからアーカイブのルートを起点とした
// パス ("トピックのディレクトリ名/ページのファイル名") への対応で、これが展開したディレクトリの
// 中身になる。
//
// Wikiリンクの中にある名前の組はスペース上のものだが、アーカイブのエントリは置換・重複の解決・
// 切り詰めを経た名前を持つ。タイトルからもう一度変換するのではなくパスを引くのはこのためである。
//
// [[ページタイトル]] と書かれたリンクは、本文が属するトピック (currentTopicName) のページを
// 指しており、そのトピックも含んだ形になって出てくる。vaultはルートを起点にパスを解決する一方、
// 同じ名前を持つノートのどれを選ぶかの規則は文書化されていないためである。
//
// pathsのどのページも指していないリンクはそのまま残す。存在しないページか、エクスポートが
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
