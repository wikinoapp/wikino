package ogcard

import (
	"strings"
	"unicode"

	"github.com/rivo/uniseg"
	"golang.org/x/image/math/fixed"
)

// ellipsisは行に収まらない文字列の末尾に付ける省略記号
const ellipsis = "…"

// measureFuncは文字列を描画したときの幅を返す
type measureFunc func(s string) fixed.Int26_6

// sanitizeは描画する文字列から、フォントで描けない文字と制御文字を取り除く
//
// フォントに無い文字を描くと豆腐 (.notdefの四角) になり、壊れた画像に見えるため取り除く。
// 絵文字の異体字セレクタやZWJもフォントに無いため、絵文字と一緒に取り除かれる。
// 改行などの制御文字は空白に置き換え、連続する空白は1つにまとめる。
func sanitize(s string, hasGlyph func(r rune) bool) string {
	var b strings.Builder
	lastIsSpace := true
	for _, r := range s {
		if unicode.IsControl(r) || r == ' ' {
			if !lastIsSpace {
				b.WriteRune(' ')
				lastIsSpace = true
			}
			continue
		}
		if !hasGlyph(r) {
			continue
		}
		b.WriteRune(r)
		lastIsSpace = false
	}
	return strings.TrimRight(b.String(), " ")
}

// headerTextはヘッダーに描く「スペース名 / トピック名」を組み立てる
//
// 絵文字だけの名前などでsanitize後に空になった値は区切りで連結しない。
// 区切りだけが浮いたり、先頭の空白で字下げされたりするのを避けるため。
func headerText(card Card, hasGlyph func(r rune) bool) string {
	var parts []string
	for _, s := range []string{card.SpaceName, card.TopicName} {
		if s := sanitize(s, hasGlyph); s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, headerSeparator)
}

// wrapは文字列をmaxWidthに収まる行に折り返し、最大maxLines行を返す
//
// 折り返し位置はUAX #14 (Unicodeの行分割規則) に従う。和文は文字間で、欧文は単語の境界で
// 折り返し、句読点や閉じ括弧は行頭に来ない。1語だけで行に収まらない場合 (長いURLなど) は、
// 書記素クラスタの境界で折り返す。
// maxLines行に収まらない場合は、最終行の末尾を省略記号に置き換える。
func wrap(s string, maxWidth fixed.Int26_6, maxLines int, measure measureFunc) []string {
	var lines []string
	var line string
	truncated := false

	state := -1
	rest := s
	for rest != "" {
		var segment string
		segment, rest, _, state = uniseg.FirstLineSegmentInString(rest, state)

		// 行末の空白は幅に数えない (折り返した行の末尾に残らないため)
		if measure(strings.TrimRight(line+segment, " ")) <= maxWidth {
			line += segment
			continue
		}

		if line != "" {
			lines = append(lines, strings.TrimRight(line, " "))
			line = ""
			if len(lines) == maxLines {
				truncated = true
				break
			}
		}

		if measure(strings.TrimRight(segment, " ")) <= maxWidth {
			line = segment
			continue
		}

		// 1語で行に収まらないため、書記素クラスタの境界で折り返す
		graphemes := uniseg.NewGraphemes(segment)
		for graphemes.Next() {
			cluster := graphemes.Str()
			if line == "" || measure(strings.TrimRight(line+cluster, " ")) <= maxWidth {
				line += cluster
				continue
			}
			lines = append(lines, strings.TrimRight(line, " "))
			line = cluster
			if len(lines) == maxLines {
				truncated = true
				break
			}
		}
		if truncated {
			break
		}
	}

	if !truncated && line != "" {
		lines = append(lines, strings.TrimRight(line, " "))
	}

	if truncated {
		last := len(lines) - 1
		lines[last] = ellipsize(lines[last], maxWidth, measure)
	}

	return lines
}

// ellipsizeは続きがあることを示すため、行の末尾に省略記号を付ける
//
// 省略記号を付けると行に収まらない場合は、収まるまで末尾の書記素クラスタを削る。
func ellipsize(line string, maxWidth fixed.Int26_6, measure measureFunc) string {
	var clusters []string
	graphemes := uniseg.NewGraphemes(line)
	for graphemes.Next() {
		clusters = append(clusters, graphemes.Str())
	}

	for n := len(clusters); n >= 0; n-- {
		candidate := strings.TrimRight(strings.Join(clusters[:n], ""), " ") + ellipsis
		if measure(candidate) <= maxWidth {
			return candidate
		}
	}
	return ellipsis
}
