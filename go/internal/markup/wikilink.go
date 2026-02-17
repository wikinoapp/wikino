package markup

import (
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// wikilinkRegex はWikiリンク記法 [[...]] を検出する正規表現
var wikilinkRegex = regexp.MustCompile(`\[\[(.*?)\]\]`)

// skipElements はWikiリンク変換をスキップするHTML要素のセット
var skipElements = map[atom.Atom]bool{
	atom.A:      true,
	atom.Code:   true,
	atom.Pre:    true,
	atom.Script: true,
	atom.Style:  true,
}

// WikilinkKey はWikiリンクのトピック名とページタイトルのペア
type WikilinkKey struct {
	// Raw はWikiリンクの原文（例: "トピック名/ページタイトル"）
	Raw string
	// TopicName はトピック名
	TopicName string
	// PageTitle はページタイトル
	PageTitle string
}

// PageLocation はWikilinkKeyに対応する解決済みページ情報
type PageLocation struct {
	// Key は元のWikiリンクキー
	Key WikilinkKey
	// TopicName は解決されたトピック名
	TopicName string
	// PageID はページID
	PageID string
	// PageNumber はページ番号（URLで使用）
	PageNumber int
	// PageTitle はページタイトル（リンクテキストで使用）
	PageTitle string
}

// ScanWikilinks はMarkdown本文からWikiリンクをパースし、
// トピック名とページタイトルのペアのリストを返す。
// [[ページ名]] 形式の場合はcurrentTopicNameを使用する。
// [[トピック名/ページ名]] 形式の場合はそのまま使用する。
func ScanWikilinks(body string, currentTopicName string) []WikilinkKey {
	matches := wikilinkRegex.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return nil
	}

	var keys []WikilinkKey
	for _, match := range matches {
		raw := strings.TrimSpace(match[1])
		if raw == "" {
			continue
		}

		parts := strings.SplitN(raw, "/", 2)
		if len(parts) == 2 {
			// [[トピック名/ページ名]] 形式
			keys = append(keys, WikilinkKey{
				Raw:       raw,
				TopicName: parts[0],
				PageTitle: parts[1],
			})
		} else {
			// [[ページ名]] 形式 — 現在のトピック名を使用
			keys = append(keys, WikilinkKey{
				Raw:       raw,
				TopicName: currentTopicName,
				PageTitle: parts[0],
			})
		}
	}

	return keys
}

// ReplaceWikilinks はbodyHTML内のWikiリンクをHTML <a>タグに変換する。
// <a>, <code>, <pre>, <script>, <style>タグ内のWikiリンクは変換しない。
// ページが存在する場合は <a href="/s/{spaceIdentifier}/pages/{pageNumber}">ページタイトル</a> に変換し、
// 存在しない場合はプレーンテキストのまま残す。
func ReplaceWikilinks(bodyHTML string, currentTopicName string, spaceIdentifier string, pageLocations []PageLocation) string {
	if !strings.Contains(bodyHTML, "[[") {
		return bodyHTML
	}

	container, err := parseHTMLFragmentWithContainer(bodyHTML)
	if err != nil {
		slog.Warn("Wikiリンク変換時のHTMLパースに失敗", "error", err)
		return bodyHTML
	}

	modified := processWikilinkNodes(container, currentTopicName, spaceIdentifier, pageLocations, false)

	if !modified {
		return bodyHTML
	}

	return renderContainerChildren(container)
}

// processWikilinkNodes はDOMツリーを再帰的に走査し、
// テキストノード内のWikiリンクを<a>要素に変換する。変更があればtrueを返す。
func processWikilinkNodes(n *html.Node, currentTopicName string, spaceIdentifier string, pageLocations []PageLocation, inSkip bool) bool {
	if n.Type == html.ElementNode && skipElements[n.DataAtom] {
		inSkip = true
	}

	if n.Type == html.TextNode && !inSkip && strings.Contains(n.Data, "[[") {
		return replaceWikilinksInTextNode(n, currentTopicName, spaceIdentifier, pageLocations)
	}

	modified := false
	for c := n.FirstChild; c != nil; {
		next := c.NextSibling
		if processWikilinkNodes(c, currentTopicName, spaceIdentifier, pageLocations, inSkip) {
			modified = true
		}
		c = next
	}

	return modified
}

// replaceWikilinksInTextNode はテキストノード内のWikiリンクを検出して<a>要素に置換する。
// テキストを分割し、Wikiリンク部分を<a>ノードに、それ以外をテキストノードに変換する。
func replaceWikilinksInTextNode(textNode *html.Node, currentTopicName string, spaceIdentifier string, pageLocations []PageLocation) bool {
	text := textNode.Data
	matches := wikilinkRegex.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return false
	}

	// 置換対象があるかチェック
	hasReplacement := false
	for _, loc := range matches {
		raw := strings.TrimSpace(text[loc[2]:loc[3]])
		if raw == "" {
			continue
		}
		key := parseWikilinkRaw(raw, currentTopicName)
		if findPageLocation(key, pageLocations) != nil {
			hasReplacement = true
			break
		}
	}

	if !hasReplacement {
		return false
	}

	parent := textNode.Parent
	lastEnd := 0

	for _, loc := range matches {
		raw := strings.TrimSpace(text[loc[2]:loc[3]])
		if raw == "" {
			continue
		}

		key := parseWikilinkRaw(raw, currentTopicName)
		pl := findPageLocation(key, pageLocations)
		if pl == nil {
			continue
		}

		// マッチ前のテキストを挿入
		if loc[0] > lastEnd {
			before := &html.Node{
				Type: html.TextNode,
				Data: text[lastEnd:loc[0]],
			}
			parent.InsertBefore(before, textNode)
		}

		// <a>要素を挿入
		aNode := buildWikilinkNode(spaceIdentifier, pl)
		parent.InsertBefore(aNode, textNode)

		lastEnd = loc[1]
	}

	// 残りのテキスト
	if lastEnd < len(text) {
		after := &html.Node{
			Type: html.TextNode,
			Data: text[lastEnd:],
		}
		parent.InsertBefore(after, textNode)
	}

	// 元のテキストノードを削除
	parent.RemoveChild(textNode)

	return true
}

// parseWikilinkRaw はWikiリンクの原文からWikilinkKeyを構築する
func parseWikilinkRaw(raw string, currentTopicName string) WikilinkKey {
	parts := strings.SplitN(raw, "/", 2)
	if len(parts) == 2 {
		return WikilinkKey{
			Raw:       raw,
			TopicName: parts[0],
			PageTitle: parts[1],
		}
	}
	return WikilinkKey{
		Raw:       raw,
		TopicName: currentTopicName,
		PageTitle: parts[0],
	}
}

// findPageLocation はWikilinkKeyに対応するPageLocationを検索する
func findPageLocation(key WikilinkKey, locations []PageLocation) *PageLocation {
	for i := range locations {
		if locations[i].Key.Raw == key.Raw {
			return &locations[i]
		}
	}
	return nil
}

// buildWikilinkNode はWikiリンクの<a>要素ノードを構築する
func buildWikilinkNode(spaceIdentifier string, pl *PageLocation) *html.Node {
	href := fmt.Sprintf("/s/%s/pages/%d", url.PathEscape(spaceIdentifier), pl.PageNumber)

	aNode := &html.Node{
		Type:     html.ElementNode,
		DataAtom: atom.A,
		Data:     "a",
		Attr: []html.Attribute{
			{Key: "href", Val: href},
		},
	}

	linkText := &html.Node{
		Type: html.TextNode,
		Data: pl.PageTitle,
	}
	aNode.AppendChild(linkText)

	return aNode
}
