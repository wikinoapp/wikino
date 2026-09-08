package markup

import (
	"bytes"
	"log/slog"
	"strings"

	"github.com/yuin/goldmark/ast"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// parseHTMLFragmentWithContainer はHTMLフラグメントをパースし、
// 全ノードを一時的なコンテナノードの子として保持する。
// コンテナノードを返すので、子ノードには必ず親が存在する。
func parseHTMLFragmentWithContainer(s string) (*html.Node, error) {
	context := &html.Node{
		Type:     html.ElementNode,
		DataAtom: atom.Body,
		Data:     "body",
	}
	nodes, err := html.ParseFragment(strings.NewReader(s), context)
	if err != nil {
		return nil, err
	}

	container := &html.Node{
		Type: html.ElementNode,
		Data: "wikino-container",
	}
	for _, n := range nodes {
		container.AppendChild(n)
	}
	return container, nil
}

// renderContainerChildren はコンテナノードの子ノードをHTML文字列に変換する。
func renderContainerChildren(container *html.Node) string {
	var b strings.Builder
	for c := container.FirstChild; c != nil; c = c.NextSibling {
		if err := html.Render(&b, c); err != nil {
			slog.Warn("HTMLノードのレンダリングに失敗", "error", err)
			continue
		}
	}
	return b.String()
}

// getAttr はノードから指定キーの属性値を取得する。見つからない場合は空文字列を返す。
func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// renderSanitized renders document and sanitizes the result, which is the HTML display hands to
// the reader. source is what document was parsed from, or a copy of it a caller has extended; the
// renderer reads every node's segments out of it.
//
// [Ja] renderSanitized は document をレンダリングして結果をサニタイズする。これは表示側が
// 読み手へ渡す HTML である。source は document の解析元、または呼び出し元がそれを拡張したもので、
// レンダラーは各ノードのセグメントをここから読む。
func renderSanitized(source []byte, document ast.Node) (string, error) {
	var rendered bytes.Buffer
	if err := md.Renderer().Render(&rendered, source, document); err != nil {
		return "", err
	}

	return policy.Sanitize(rendered.String()), nil
}

// renderedTree renders document and returns the HTML tree display builds from it, using the same
// renderer, sanitizer and fragment parser. A caller reading a body the way the screen shows it
// therefore sees the elements, the text and the drops the reader sees.
//
// [Ja] renderedTree は document をレンダリングし、表示側が組み立てるのと同じ HTML ツリーを返す。
// レンダラー・サニタイザー・フラグメントパーサーは表示側と同じものを使う。画面の見え方どおりに
// 本文を読む側は、これにより読み手と同じ要素・テキスト・欠落を見ることになる。
func renderedTree(source []byte, document ast.Node) (*html.Node, error) {
	bodyHTML, err := renderSanitized(source, document)
	if err != nil {
		return nil, err
	}

	return parseHTMLFragmentWithContainer(bodyHTML)
}
