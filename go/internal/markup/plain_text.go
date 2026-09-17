package markup

import (
	"log/slog"
	"strings"
	"unicode"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// plainTextSkipElementsは、テキストが散文ではなくブラウザ向けのマークアップである要素。
// これらの内容は抽出したテキストに含めない。
var plainTextSkipElements = map[atom.Atom]bool{
	atom.Script: true,
	atom.Style:  true,
}

// plainTextBlockElementsは散文の行を区切る要素。前後に区切り文字を出し、隣接するブロックが
// 繋がらないようにする ("<p>a</p><p>b</p>" が "ab" になってはいけない)。インライン要素を含めない
// のは意図的で、前後に空白を入れると分かち書きしない日本語の文が壊れるため
// ("これは<strong>太字</strong>です" は "これは太字です" のままにする)。Goldmarkが通常生成しない
// raw HTMLでも、newSanitizationPolicyが保持するブロック相当の要素はこの集合に含める。
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

// PlainTextはレンダリング済みの本文HTMLから読めるテキストを取り出し、連続する空白を半角
// スペース1個にまとめて返す。meta descriptionなどHTMLを置けない箇所で使う。パースに失敗した
// 場合は生のマークアップではなく空文字列を返し、タグがプレーンテキストの文脈へ漏れないようにする。
//
// maxRunesは結果の上限で、短い前半だけが必要な呼び出し元が、捨てることになる本文全体を取り出さず
// 途中で走査を打ち切れるようにする。0以下なら上限無し。返り値は上限無しの結果の先頭maxRunes
// 文字そのものになるため、必要な文字数より1文字多く要求すれば、長さから切り詰めの有無が分かる。
//
// 上限が無い場合は前後の空白も取り除く。上限がある場合は先頭部分と一致させる契約を優先するため、
// 切り取り位置が2つのブロックの間に出す半角スペースと重なるとその空白が末尾に残る。結果をそのまま
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

// plainTextWriterは連続する空白を半角スペース1個にまとめながらテキストを集め、limit文字に
// 達したら止まる。空白はすぐ書かずpendingSpaceとして保持するため、テキストの前後にある空白が
// 出力へ入ることはない。
type plainTextWriter struct {
	b            strings.Builder
	limit        int
	count        int
	pendingSpace bool
	done         bool
}

// writeRuneは1文字書き足し、上限に達したらdoneを立てる。
func (w *plainTextWriter) writeRune(r rune) {
	w.b.WriteRune(r)
	w.count++

	if w.limit > 0 && w.count >= w.limit {
		w.done = true
	}
}

// markBoundaryはブロックの境界を記録する。後ろにテキストが続く場合にだけ半角スペース1個になる。
func (w *plainTextWriter) markBoundary() {
	w.pendingSpace = true
}

// writeTextはテキストノードを書き足す。空白をまとめ、上限を守る。
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

// appendPlainTextはツリーを深さ優先で走査して各テキストノードを書き出し、ブロック要素の前後に
// 境界を記録して隣接するブロックが半角スペース1個で区切られるようにする。
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
