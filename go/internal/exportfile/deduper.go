package exportfile

import (
	"strconv"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// Deduper hands out the names of one directory of the archive, so that no two entries in it share
// a name. Two titles that differ only in case, or in how their characters are composed, still
// collide once extracted: a file system that ignores case keeps only one of "API" and "api", and
// macOS stores a name decomposed while the archive carries it composed.
//
// A Deduper covers a single directory. The topic directories are named through one Deduper, and
// the pages of each topic through one of its own. Its calls are not safe to make concurrently.
//
// [Ja] Deduper はアーカイブの 1 ディレクトリ分の名前を配り、同じディレクトリのエントリが名前を
// 共有しないようにする。大文字小文字だけが違うタイトルや、文字の合成の仕方だけが違うタイトルも、
// 展開すると衝突する。大文字小文字を区別しないファイルシステムは "API" と "api" の片方しか
// 保持せず、macOS は分解した形で名前を保存する一方でアーカイブは合成した形で持つためである。
//
// 1 つの Deduper が担当するのは 1 ディレクトリである。トピックのディレクトリ名は 1 つの Deduper
// で、各トピックのページ名はトピックごとの Deduper で決める。並行して呼び出すことはできない。
type Deduper struct {
	taken map[string]bool
}

// NewDeduper returns a Deduper for one directory.
//
// [Ja] NewDeduper は 1 ディレクトリ分の Deduper を返す。
func NewDeduper() *Deduper {
	return &Deduper{taken: map[string]bool{}}
}

// Unique returns the name of title inside this directory, in the form Name gives it. A name
// already handed out grows a counter before its extension ("api-2.md"), counting up until the
// result is one this directory does not hold yet.
//
// [Ja] Unique はこのディレクトリでの title の名前を、Name と同じ形で返す。すでに配った名前の
// ときは拡張子の前に連番を付け ("api-2.md")、このディレクトリにまだ無い名前になるまで数を
// 増やす。
func (d *Deduper) Unique(title, ext string) string {
	name := Name(title, ext)

	for counter := 2; d.taken[dedupeKey(name)]; counter++ {
		suffix := "-" + strconv.Itoa(counter)
		name = sanitize(title, len(suffix)+len(ext)) + suffix + ext
	}

	d.taken[dedupeKey(name)] = true

	return name
}

// dedupeKey folds the differences that a file system ignores when it decides whether two names are
// the same one.
//
// [Ja] dedupeKey は、2 つの名前が同じものかをファイルシステムが判断するときに無視する差を
// 吸収する。
func dedupeKey(name string) string {
	return strings.ToLower(norm.NFC.String(name))
}
