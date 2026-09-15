package exportfile

import (
	"strconv"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// Deduperはアーカイブの1ディレクトリ分の名前を配り、同じディレクトリのエントリが名前を
// 共有しないようにする。大文字小文字だけが違うタイトルや、文字の合成の仕方だけが違うタイトルも、
// 展開すると衝突する。大文字小文字を区別しないファイルシステムは "API" と "api" の片方しか
// 保持せず、macOSは分解した形で名前を保存する一方でアーカイブは合成した形で持つためである。
//
// 1つのDeduperが担当するのは1ディレクトリである。トピックのディレクトリ名は1つのDeduper
// で、各トピックのページ名はトピックごとのDeduperで決める。並行して呼び出すことはできない。
type Deduper struct {
	taken map[string]bool
}

// NewDeduperは1ディレクトリ分のDeduperを返す。
func NewDeduper() *Deduper {
	return &Deduper{taken: map[string]bool{}}
}

// Uniqueはこのディレクトリでのtitleの名前を、Nameと同じ形で返す。すでに配った名前の
// ときは拡張子の前に連番を付け ("api-2.md")、このディレクトリにまだ無い名前になるまで数を
// 増やす。
func (d *Deduper) Unique(title, ext string) string {
	name := Name(title, ext)

	for counter := 2; d.taken[dedupeKey(name)]; counter++ {
		suffix := "-" + strconv.Itoa(counter)
		safeExt := safeExtension(ext, len(suffix))
		name = sanitize(title, len(suffix)+len(safeExt)) + suffix + safeExt
	}

	d.taken[dedupeKey(name)] = true

	return name
}

// dedupeKeyは、2つの名前が同じものかをファイルシステムが判断するときに無視する差を
// 吸収する。
func dedupeKey(name string) string {
	return strings.ToLower(norm.NFC.String(name))
}
