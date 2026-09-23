package markup

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

// urlRegexは http:// または https:// で始まるURLにマッチする正規表現。
// 空白文字とアングルブラケット (< >) をURLの境界として扱う。
var urlRegex = regexp.MustCompile(`https?://[^\s<>]+`)

// urlTrailingPunctはURLの末尾から無条件に取り除く句読点。
// 文末にURLが書かれた場合に、句読点までURLに含めないようにする。
const urlTrailingPunct = ".,;:!?"

// LinkifyPlainTextはプレーンテキストをHTMLエスケープした上で、
// http:// / https:// で始まるURLを <a> タグに変換したHTML文字列を返す。
// XSS対策のため、URL以外の部分は必ずエスケープしてから連結する。
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

		// URL末尾の句読点と不対応の閉じ括弧を取り除く。
		// `Foo_(bar)` のようにURL内で括弧が対応している場合は保持する。
		url, end = trimURLTrailing(url, end)

		if url == "" {
			continue
		}

		// マッチ前の通常テキストをエスケープして書き出す
		result.WriteString(html.EscapeString(text[lastEnd:start]))

		// URLを <a> タグに変換する (hrefとリンクテキストの両方をエスケープ)
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

// trimURLTrailingはURL末尾の句読点と不対応の閉じ括弧を取り除き、
// 取り除いた後のURLと元のテキスト上での終端位置を返す。
//
// 末尾の判定はASCII 1 byte単位で行うが、対象の句読点・閉じ括弧は
// すべてASCIIであり、マルチバイト文字のURL末尾 (例: 日本語パス)
// では何も除外されないため安全である。
//
// 句読点 (. , ; : ! ?) は無条件に取り除く。
// 閉じ括弧 ) ] } は、URL内に対応する開き括弧がない場合のみ取り除く
// (例: Wikipediaの `Foo_(bar)` のように括弧が対応しているURLは保持する)。
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

// isUnbalancedClosingBracketはurlの末尾文字lastが閉じ括弧であり、
// かつURL内に対応する開き括弧がないかどうかを返す。
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
