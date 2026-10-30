package model

import (
	"strings"
	"unicode/utf8"
)

// BodyExcerptMaxRunesは本文の抜粋に残す最大の文字数 (rune)
const BodyExcerptMaxRunes = 80

// bodyExcerptEllipsisは、抜粋が本文の途中で切れていることを表す記号
const bodyExcerptEllipsis = "…"

// BodyExcerptは、一覧に出す本文の冒頭の抜粋を返す。
// 改行を含む連続した空白を1つの空白に詰め、先頭からBodyExcerptMaxRunes文字までを残す。
// 切り詰めたときは末尾に「…」を付け、本文がそこで終わっていないことを示す。
// 文字数はバイトではなくruneで数え、マルチバイト文字の途中で切らない。
//
// 一覧で本文を返さなくても、本文を読まずにどのページを詳しく読むかを選べるようにするために使う
func BodyExcerpt(body string) string {
	collapsed := strings.Join(strings.Fields(body), " ")
	if utf8.RuneCountInString(collapsed) <= BodyExcerptMaxRunes {
		return collapsed
	}

	count := 0
	for i := range collapsed {
		if count == BodyExcerptMaxRunes {
			return strings.TrimRight(collapsed[:i], " ") + bodyExcerptEllipsis
		}
		count++
	}

	return collapsed
}
