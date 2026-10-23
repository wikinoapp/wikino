package output

import (
	"strings"
	"unicode/utf8"
)

// excerptEllipsisは、抜粋が本文の途中で切れていることを表す記号。
const excerptEllipsis = "…"

// Excerptは、一覧に出す本文の冒頭の抜粋を返す。
// 改行を含む連続した空白を1つの空白に詰め、先頭からmaxRunes文字までを残す。
// 切り詰めたときは末尾に「…」を付け、本文がそこで終わっていないことを示す。
// 文字数はバイトではなくrune (Unicodeのコードポイント) で数え、マルチバイト文字の途中で切らない。
//
// 一覧で本文を既定で出さない代わりに、本文を取らずに深掘り先を選べるようにするために使う。
func Excerpt(body string, maxRunes int) string {
	collapsed := strings.Join(strings.Fields(body), " ")
	if utf8.RuneCountInString(collapsed) <= maxRunes {
		return collapsed
	}

	count := 0
	for i := range collapsed {
		if count == maxRunes {
			return strings.TrimRight(collapsed[:i], " ") + excerptEllipsis
		}
		count++
	}

	return collapsed
}
