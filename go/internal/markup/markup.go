// Package markup はMarkdownテキストをサニタイズ済みHTMLに変換する機能を提供する。
package markup

import (
	"bytes"
	"log/slog"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// zwnj はゼロ幅非接合子（Zero-Width Non-Joiner）
const zwnj = "\u200C"

// 単独行のimgタグの直後にMarkdownインライン記法（イタリック等）が続くパターン。
// goldmarkがimgタグをHTMLブロックとして解析しないよう、ZWNJを挿入する。
var standaloneImgRegex = regexp.MustCompile(`(?m)^(<img[^>]*>)[ \t]*\n(\*[^*]+\*)[ \t]*$`)

// ZWNJを含む不要なHTML断片を除去するパターン
var zwnjBrRegex = regexp.MustCompile(`(?m)^<p>` + zwnj + `<br\s*/?>\n?`)

var md = goldmark.New(
	goldmark.WithExtensions(
		extension.Linkify,
		extension.NewTable(
			extension.WithTableCellAlignMethod(extension.TableCellAlignAttribute),
		),
		extension.Strikethrough,
		extension.TaskList,
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
	),
	goldmark.WithRendererOptions(
		// Rails版の parse: {html: true}, render: {unsafe: true} に相当。
		// Markdown内のHTMLタグをそのまま出力する。
		html.WithUnsafe(),
		html.WithHardWraps(),
	),
)

// policy はHTMLサニタイズポリシー。
// bluemondayのUGCPolicy（User Generated Content向け）をベースに、
// タスクリスト用のinput要素とimg要素のwidth/height属性を許可する。
var policy = newSanitizationPolicy()

func newSanitizationPolicy() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()

	// タスクリスト記法（- [ ] / - [x]）で生成されるチェックボックスを許可
	p.AllowAttrs("type", "checked", "disabled").OnElements("input")

	// img要素のwidth/height属性を許可（Rails版のsanitization_configに相当）
	p.AllowAttrs("src", "alt", "title", "width", "height").OnElements("img")

	// テーブルの配置指定（GFM）で生成されるalign属性を許可
	p.AllowAttrs("align").OnElements("td", "th")

	return p
}

// RenderMarkdown はMarkdownテキストをサニタイズ済みHTMLに変換する。
// 空文字列が渡された場合は空文字列を返す。
func RenderMarkdown(text string) string {
	if text == "" {
		return ""
	}

	// CRLFをLFに正規化（ブラウザのtextareaはCRLFで送信する場合がある）
	text = strings.ReplaceAll(text, "\r\n", "\n")

	// 単独行のimgタグの前にZWNJを挿入してインラインHTMLとして処理させる
	processed := standaloneImgRegex.ReplaceAllString(text, zwnj+"\n$1\n$2")

	// Markdown → HTML変換
	var buf bytes.Buffer
	if err := md.Convert([]byte(processed), &buf); err != nil {
		slog.Warn("Markdown変換に失敗", "error", err)
		return ""
	}

	// HTMLサニタイズ（XSS対策）
	sanitized := policy.Sanitize(buf.String())

	// ZWNJと余分なHTML断片を除去
	sanitized = zwnjBrRegex.ReplaceAllString(sanitized, "<p>")
	sanitized = strings.ReplaceAll(sanitized, zwnj, "")

	return sanitized
}
