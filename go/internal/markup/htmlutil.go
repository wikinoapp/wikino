package markup

import (
	"log/slog"
	"strings"

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
