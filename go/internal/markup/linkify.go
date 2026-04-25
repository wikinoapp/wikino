package markup

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

// urlRegex は http:// または https:// で始まる URL にマッチする正規表現。
// 空白文字とアングルブラケット（< >）を URL の境界として扱う。
var urlRegex = regexp.MustCompile(`https?://[^\s<>]+`)

// urlTrailingPunct は URL の末尾から無条件に取り除く句読点。
// 文末に URL が書かれた場合に、句読点まで URL に含めないようにする。
const urlTrailingPunct = ".,;:!?"

// LinkifyPlainText はプレーンテキストを HTML エスケープした上で、
// http:// / https:// で始まる URL を <a> タグに変換した HTML 文字列を返す。
// XSS 対策のため、URL 以外の部分は必ずエスケープしてから連結する。
func LinkifyPlainText(text string) string {
	if text == "" {
		return ""
	}

	var result strings.Builder
	matches := urlRegex.FindAllStringIndex(text, -1)
	lastEnd := 0

	for _, match := range matches {
		start, end := match[0], match[1]
		url := text[start:end]

		// URL 末尾の句読点と不対応の閉じ括弧を取り除く。
		// `Foo_(bar)` のように URL 内で括弧が対応している場合は保持する。
		url, end = trimURLTrailing(url, end)

		if url == "" {
			continue
		}

		// マッチ前の通常テキストをエスケープして書き出す
		result.WriteString(html.EscapeString(text[lastEnd:start]))

		// URL を <a> タグに変換する（href とリンクテキストの両方をエスケープ）
		escapedURL := html.EscapeString(url)
		fmt.Fprintf(&result,
			`<a href="%s" rel="noopener noreferrer" target="_blank">%s</a>`,
			escapedURL, escapedURL,
		)

		lastEnd = end
	}

	// 最後のマッチ以降のテキストをエスケープして書き出す
	result.WriteString(html.EscapeString(text[lastEnd:]))

	return result.String()
}

// trimURLTrailing は URL 末尾の句読点と不対応の閉じ括弧を取り除き、
// 取り除いた後の URL と元のテキスト上での終端位置を返す。
//
// 末尾の判定は ASCII 1 byte 単位で行うが、対象の句読点・閉じ括弧は
// すべて ASCII であり、マルチバイト文字の URL 末尾（例: 日本語パス）
// では何も除外されないため安全である。
//
// 句読点（. , ; : ! ?）は無条件に取り除く。
// 閉じ括弧 ) ] } は、URL 内に対応する開き括弧がない場合のみ取り除く
// （例: Wikipedia の `Foo_(bar)` のように括弧が対応している URL は保持する）。
func trimURLTrailing(url string, end int) (string, int) {
	for len(url) > 0 {
		last := url[len(url)-1]
		if strings.ContainsRune(urlTrailingPunct, rune(last)) {
			url = url[:len(url)-1]
			end--
			continue
		}
		if isUnbalancedClosingBracket(url, last) {
			url = url[:len(url)-1]
			end--
			continue
		}
		break
	}
	return url, end
}

// isUnbalancedClosingBracket は url の末尾文字 last が閉じ括弧であり、
// かつ URL 内に対応する開き括弧がないかどうかを返す。
func isUnbalancedClosingBracket(url string, last byte) bool {
	switch last {
	case ')':
		return strings.Count(url, "(") < strings.Count(url, ")")
	case ']':
		return strings.Count(url, "[") < strings.Count(url, "]")
	case '}':
		return strings.Count(url, "{") < strings.Count(url, "}")
	}
	return false
}
