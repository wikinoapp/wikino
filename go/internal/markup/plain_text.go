package markup

import (
	"log/slog"
	"strings"
	"unicode"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// plainTextSkipElements are elements whose text is markup for the browser rather than prose, so
// their content never belongs in the extracted text.
//
// [Ja] plainTextSkipElements は、テキストが散文ではなくブラウザ向けのマークアップである要素。
// これらの内容は抽出したテキストに含めない。
var plainTextSkipElements = map[atom.Atom]bool{
	atom.Script: true,
	atom.Style:  true,
}

// plainTextBlockElements are elements that end a line of prose. A separator is emitted around them
// so that adjacent blocks do not run together ("<p>a</p><p>b</p>" must not become "ab"). Inline
// elements are deliberately absent: inserting a space around them would break text in languages
// that do not use word separators. The set also includes block-like raw HTML elements preserved by
// newSanitizationPolicy, even when Goldmark does not normally emit them.
//
// [Ja] plainTextBlockElements は散文の行を区切る要素。前後に区切り文字を出し、隣接するブロックが
// 繋がらないようにする ("<p>a</p><p>b</p>" が "ab" になってはいけない)。インライン要素を含めない
// のは意図的で、前後に空白を入れると分かち書きしない日本語の文が壊れるため
// ("これは<strong>太字</strong>です" は "これは太字です" のままにする)。Goldmark が通常生成しない
// raw HTML でも、newSanitizationPolicy が保持するブロック相当の要素はこの集合に含める。
var plainTextBlockElements = map[atom.Atom]bool{
	atom.Article:    true,
	atom.Aside:      true,
	atom.Blockquote: true,
	atom.Br:         true,
	atom.Caption:    true,
	atom.Dd:         true,
	atom.Details:    true,
	atom.Div:        true,
	atom.Dl:         true,
	atom.Dt:         true,
	atom.Figcaption: true,
	atom.Figure:     true,
	atom.H1:         true,
	atom.H2:         true,
	atom.H3:         true,
	atom.H4:         true,
	atom.H5:         true,
	atom.H6:         true,
	atom.Hgroup:     true,
	atom.Hr:         true,
	atom.Li:         true,
	atom.Ol:         true,
	atom.P:          true,
	atom.Pre:        true,
	atom.Section:    true,
	atom.Summary:    true,
	atom.Table:      true,
	atom.Tbody:      true,
	atom.Td:         true,
	atom.Tfoot:      true,
	atom.Th:         true,
	atom.Thead:      true,
	atom.Tr:         true,
	atom.Ul:         true,
}

// PlainText extracts the readable text from rendered body HTML, with every run of whitespace
// collapsed into a single space. Callers use it where HTML cannot appear, such as the meta
// description. Unparsable input yields an empty string rather than raw markup, so that tags never
// leak into a plain-text context.
//
// maxRunes caps the result, which lets a caller that only needs a short prefix stop the walk early
// instead of extracting a whole body it will throw away; a value of zero or less means no cap. The
// result is exactly the first maxRunes runes of the uncapped result, so a caller that asks for one
// rune more than it needs can tell from the length whether the text was cut short.
//
// Without a cap both ends are trimmed. With one, the prefix contract wins: a cut landing on the
// single space emitted between two blocks keeps that space at the end, so a caller that renders the
// result verbatim trims it itself.
//
// [Ja] PlainText はレンダリング済みの本文 HTML から読めるテキストを取り出し、連続する空白を半角
// スペース 1 個にまとめて返す。meta description など HTML を置けない箇所で使う。パースに失敗した
// 場合は生のマークアップではなく空文字列を返し、タグがプレーンテキストの文脈へ漏れないようにする。
//
// maxRunes は結果の上限で、短い前半だけが必要な呼び出し元が、捨てることになる本文全体を取り出さず
// 途中で走査を打ち切れるようにする。0 以下なら上限無し。返り値は上限無しの結果の先頭 maxRunes
// 文字そのものになるため、必要な文字数より 1 文字多く要求すれば、長さから切り詰めの有無が分かる。
//
// 上限が無い場合は前後の空白も取り除く。上限がある場合は先頭部分と一致させる契約を優先するため、
// 切り取り位置が 2 つのブロックの間に出す半角スペースと重なるとその空白が末尾に残る。結果をそのまま
// 表示する呼び出し元は自身で末尾を落とす。
func PlainText(bodyHTML string, maxRunes int) string {
	if bodyHTML == "" {
		return ""
	}

	container, err := parseHTMLFragmentWithContainer(bodyHTML)
	if err != nil {
		slog.Warn("プレーンテキスト抽出時のHTMLパースに失敗", "error", err)
		return ""
	}

	w := &plainTextWriter{limit: maxRunes}
	appendPlainText(w, container)

	return w.b.String()
}

// plainTextWriter collects text with every run of whitespace folded into a single space, stopping
// once limit runes have been written. Whitespace is buffered as pendingSpace rather than written
// immediately, so a run at either end of the text never reaches the output.
//
// [Ja] plainTextWriter は連続する空白を半角スペース 1 個にまとめながらテキストを集め、limit 文字に
// 達したら止まる。空白はすぐ書かず pendingSpace として保持するため、テキストの前後にある空白が
// 出力へ入ることはない。
type plainTextWriter struct {
	b            strings.Builder
	limit        int
	count        int
	pendingSpace bool
	done         bool
}

// writeRune appends one rune and marks the writer done once the limit is reached.
//
// [Ja] writeRune は 1 文字書き足し、上限に達したら done を立てる。
func (w *plainTextWriter) writeRune(r rune) {
	w.b.WriteRune(r)
	w.count++

	if w.limit > 0 && w.count >= w.limit {
		w.done = true
	}
}

// markBoundary records a block boundary. It becomes a single space only if more text follows.
//
// [Ja] markBoundary はブロックの境界を記録する。後ろにテキストが続く場合にだけ半角スペース 1 個になる。
func (w *plainTextWriter) markBoundary() {
	w.pendingSpace = true
}

// writeText appends a text node, collapsing its whitespace and honouring the limit.
//
// [Ja] writeText はテキストノードを書き足す。空白をまとめ、上限を守る。
func (w *plainTextWriter) writeText(s string) {
	for _, r := range s {
		if unicode.IsSpace(r) {
			w.pendingSpace = true
			continue
		}

		if w.pendingSpace {
			w.pendingSpace = false
			if w.b.Len() > 0 {
				w.writeRune(' ')
				if w.done {
					return
				}
			}
		}

		w.writeRune(r)
		if w.done {
			return
		}
	}
}

// appendPlainText walks the tree depth-first and writes each text node, marking a boundary around
// block elements so that adjacent blocks are separated by a single space.
//
// [Ja] appendPlainText はツリーを深さ優先で走査して各テキストノードを書き出し、ブロック要素の前後に
// 境界を記録して隣接するブロックが半角スペース 1 個で区切られるようにする。
func appendPlainText(w *plainTextWriter, n *html.Node) {
	if w.done {
		return
	}

	if n.Type == html.ElementNode && plainTextSkipElements[n.DataAtom] {
		return
	}

	if n.Type == html.TextNode {
		w.writeText(n.Data)
		return
	}

	isBlock := n.Type == html.ElementNode && plainTextBlockElements[n.DataAtom]
	if isBlock {
		w.markBoundary()
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		appendPlainText(w, c)
		if w.done {
			return
		}
	}

	if isBlock {
		w.markBoundary()
	}
}
