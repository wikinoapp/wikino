package markup

import (
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// attachmentAttrKeysは、添付ファイルのプレースホルダーに付けた、画面のJSが読む属性の集合。
var attachmentAttrKeys = map[string]bool{
	"data-attachment-id":   true,
	"data-attachment-type": true,
	"data-attachment-link": true,
}

// AbsolutizeURLsは、RenderHTMLBatchで描画した本文のHTMLのURLを絶対URLに書き換える。
// フィードのように、ページ画面の外 (JSが動かず、相対URLの基準も無い場所) で本文を読ませるためのものである。
//
//   - 添付ファイルのプレースホルダー (data-attachment-idを持つ要素) は、{appURL}/attachments/{ID} を指す。
//     描画に使う属性 (data-attachment-*) は外す
//   - アプリ内の相対リンク (`/` で始まり `//` で始まらないもの。Wikiリンクを含む) は、appURLを前に付ける
//   - ページ内のリンク (`#` で始まるもの) は、pageURLの後ろに付ける
//   - それ以外 (外部リンクなど) は変えない
//
// appURLはスキームとホストだけのURL (例: https://example.com) 、pageURLは本文のページの絶対URLを渡す。
// リーダーによってxml:baseの扱いが揃わないため、基準のURLに頼らずURL自体を絶対にする。
func AbsolutizeURLs(bodyHTML string, appURL string, pageURL string) (string, error) {
	if bodyHTML == "" {
		return "", nil
	}

	container, err := parseHTMLFragmentWithContainer(bodyHTML)
	if err != nil {
		return "", err
	}

	r := urlAbsolutizer{appURL: strings.TrimRight(appURL, "/"), pageURL: pageURL}
	r.walk(container)

	return renderContainerChildren(container), nil
}

// urlAbsolutizerは、本文のツリーのURLを絶対URLへ書き換える。
type urlAbsolutizer struct {
	appURL  string
	pageURL string
}

func (r urlAbsolutizer) walk(n *html.Node) {
	if n.Type == html.ElementNode {
		r.rewriteElement(n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		r.walk(c)
	}
}

// rewriteElementは、要素がURLを持つ属性 (a要素のhref、img要素・video要素のsrc) を書き換える。
func (r urlAbsolutizer) rewriteElement(n *html.Node) {
	var urlKey string
	switch n.Data {
	case "a":
		urlKey = "href"
	case "img", "video":
		urlKey = "src"
	default:
		return
	}

	attachmentID := getAttr(n, "data-attachment-id")

	attrs := n.Attr[:0]
	for _, a := range n.Attr {
		if attachmentAttrKeys[a.Key] {
			continue
		}
		if a.Key == urlKey {
			if attachmentID != "" {
				a.Val = r.attachmentURL(attachmentID)
			} else {
				a.Val = r.absolutize(a.Val)
			}
		}
		attrs = append(attrs, a)
	}
	n.Attr = attrs
}

func (r urlAbsolutizer) attachmentURL(attachmentID string) string {
	return r.appURL + "/attachments/" + url.PathEscape(attachmentID)
}

// absolutizeは、アプリ内の相対リンクとページ内のリンクを絶対URLにする。
func (r urlAbsolutizer) absolutize(rawURL string) string {
	switch {
	case strings.HasPrefix(rawURL, "#"):
		return r.pageURL + rawURL
	case strings.HasPrefix(rawURL, "/") && !strings.HasPrefix(rawURL, "//"):
		return r.appURL + rawURL
	default:
		return rawURL
	}
}
