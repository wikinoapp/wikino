package output

import (
	"bytes"
	"io"
	"strings"
	"unicode"

	"golang.org/x/text/width"

	"github.com/wikinoapp/wikino/cli/internal/iostreams"
)

// 人間向けの表示で使うANSIのエスケープシーケンス。
const (
	ansiBold  = "\x1b[1m"
	ansiReset = "\x1b[0m"
)

// columnGapは表の列の間の空白の数。
const columnGap = 2

// Fieldは1件の詳細に並べる、ラベル付きの項目。
type Field struct {
	Label string
	Value string
}

// PrintTableは一覧を表にして出す。列の幅は表示幅 (全角文字を2とする) で揃える。
// セルの改行やタブ、端末を操作する制御文字は空白に置き換え、行の崩れと端末への注入を防ぐ。
func (p *Printer) PrintTable(headers []string, rows [][]string) error {
	table := make([][]string, 0, len(rows)+1)
	table = append(table, sanitizeCells(headers))
	for _, row := range rows {
		table = append(table, sanitizeCells(row))
	}

	widths := make([]int, len(headers))
	for _, row := range table {
		for i, cell := range row {
			if i < len(widths) {
				widths[i] = max(widths[i], displayWidth(cell))
			}
		}
	}

	var buf bytes.Buffer
	for r, row := range table {
		var line strings.Builder
		for i, cell := range row {
			if i >= len(widths) {
				break
			}
			if i > 0 {
				line.WriteString(strings.Repeat(" ", columnGap))
			}
			line.WriteString(cell)
			if i < len(row)-1 {
				line.WriteString(strings.Repeat(" ", widths[i]-displayWidth(cell)))
			}
		}

		text := line.String()
		if r == 0 {
			text = p.bold(text)
		}
		buf.WriteString(text + "\n")
	}

	_, err := buf.WriteTo(p.ios.Out)

	return err
}

// PrintDetailは1件の詳細を、ラベル付きの項目に続けて本文をそのまま出す。
// 本文が空なら項目だけを出す。
func (p *Printer) PrintDetail(fields []Field, body string) error {
	labelWidth := 0
	for _, f := range fields {
		labelWidth = max(labelWidth, displayWidth(SanitizeLine(f.Label)))
	}

	var buf bytes.Buffer
	for _, f := range fields {
		label := SanitizeLine(f.Label)
		padding := strings.Repeat(" ", labelWidth-displayWidth(label))
		buf.WriteString(p.bold(label+":") + padding + " " + SanitizeLine(f.Value) + "\n")
	}

	if body != "" {
		buf.WriteString("\n" + sanitizeBody(body))
		if !strings.HasSuffix(body, "\n") {
			buf.WriteString("\n")
		}
	}

	_, err := buf.WriteTo(p.ios.Out)

	return err
}

func (p *Printer) bold(s string) string {
	if !p.ios.ColorEnabled() {
		return s
	}

	return ansiBold + s + ansiReset
}

// WriteTextは、ページの本文やAPIの応答のような外部由来の文字列を、そのまま標準出力に書く。
// 標準出力がTTYのときだけ、人間向けの本文の表示と同じく改行とタブ以外の制御文字を取り除き、
// CRLFをLFにする。パイプとファイルへの出力は、書き出したファイルの書き戻しや
// ほかのコマンドへの受け渡しを壊さないよう、1バイトも変えない。
func WriteText(ios *iostreams.IOStreams, s string) error {
	_, err := io.WriteString(ios.Out, terminalText(ios, s))

	return err
}

// terminalTextは、標準出力がTTYのときだけsの制御文字を取り除く。
func terminalText(ios *iostreams.IOStreams, s string) string {
	if !ios.IsStdoutTTY() {
		return s
	}

	return sanitizeBody(s)
}

func sanitizeCells(cells []string) []string {
	sanitized := make([]string, len(cells))
	for i, cell := range cells {
		sanitized[i] = SanitizeLine(cell)
	}

	return sanitized
}

// SanitizeLineは、1行に収める値の制御文字 (改行・タブ・エスケープなど) と双方向テキストの
// 制御文字を空白に置き換える。タイトルなどの値は他の利用者が書いたもので、端末を操作する
// エスケープシーケンスや、表示上の文字の並びを入れ替えて確認の文言を偽装する文字を含みうる。
func SanitizeLine(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || isBidiControl(r) {
			return ' '
		}

		return r
	}, s)
}

// isBidiControlは、rが双方向テキストの埋め込み・上書き・分離の文字かを返す。
// これらは制御文字 (Cc) ではなくformat文字 (Cf) のため、unicode.IsControlでは判定できない。
// 本文は他の言語の正当な文章を含みうるため、1行に収める値だけで使う。
func isBidiControl(r rune) bool {
	return (r >= '\u202a' && r <= '\u202e') || (r >= '\u2066' && r <= '\u2069')
}

// sanitizeBodyは、本文から改行とタブ以外の制御文字を取り除く。
// 復帰 (CR) は行の先頭に戻って表示を上書きできるため、CRLFの改行もLFにそろえる。
func sanitizeBody(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")

	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' || !unicode.IsControl(r) {
			return r
		}

		return -1
	}, s)
}

// displayWidthは端末での表示幅を返す。東アジアの全角文字を2、結合文字を0、それ以外を1とする。
func displayWidth(s string) int {
	w := 0
	for _, r := range s {
		switch {
		case unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r):
		case isWide(r):
			w += 2
		default:
			w++
		}
	}

	return w
}

func isWide(r rune) bool {
	switch width.LookupRune(r).Kind() {
	case width.EastAsianWide, width.EastAsianFullwidth:
		return true
	default:
		return false
	}
}
