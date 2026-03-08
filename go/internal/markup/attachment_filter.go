package markup

import (
	"context"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/wikinoapp/wikino/go/internal/model"
)

var (
	// インライン表示可能な画像フォーマット
	inlineImageFormats = map[string]bool{
		"jpg": true, "jpeg": true, "png": true,
		"gif": true, "svg": true, "webp": true,
	}

	// インライン表示可能な動画フォーマット
	inlineVideoFormats = map[string]bool{
		"mp4": true, "webm": true, "ogg": true, "mov": true,
	}

	// 添付ファイルURLパターン（/attachments/{id}のみマッチ）
	// URLデコード後にバックスラッシュを含むIDは不正な入力として除外する
	attachmentPathRegex = regexp.MustCompile(`^/attachments/([^/\\]+)$`)
)

// AttachmentFinder は添付ファイルの検索インターフェース。
// repository.AttachmentRepository がこのインターフェースを満たす。
type AttachmentFinder interface {
	FindByIDAndSpace(ctx context.Context, id model.AttachmentID, spaceID model.SpaceID) (*model.Attachment, error)
}

// FilterAttachments はHTML内の添付ファイルURLを持つimg・a要素を変換する。
// /attachments/{id}パターンのURLを検出し、ファイル種別に応じた表示用HTMLに変換する。
// URLはプレースホルダー（src=""、href="#"）で出力し、data-attachment-id属性を付与する。
// 同じスペースの添付ファイルが存在しない場合は変換をスキップする。
func FilterAttachments(ctx context.Context, bodyHTML string, spaceID model.SpaceID, finder AttachmentFinder) (string, error) {
	if bodyHTML == "" {
		return "", nil
	}

	container, err := parseHTMLFragmentWithContainer(bodyHTML)
	if err != nil {
		return "", err
	}

	modified, err := processAttachmentNodes(ctx, container, spaceID, finder)
	if err != nil {
		return "", err
	}

	if !modified {
		return bodyHTML, nil
	}

	return renderContainerChildren(container), nil
}

// processAttachmentNodes はDOMツリーを再帰的に走査し、
// img要素とa要素の添付ファイルURLを変換する。変更があればtrueを返す。
func processAttachmentNodes(ctx context.Context, n *html.Node, spaceID model.SpaceID, finder AttachmentFinder) (bool, error) {
	modified := false

	if n.Type == html.ElementNode {
		switch n.DataAtom {
		case atom.Img:
			m, err := processImgNode(ctx, n, spaceID, finder)
			if err != nil {
				return false, err
			}
			if m {
				modified = true
			}
		case atom.A:
			m, err := processANode(ctx, n, spaceID, finder)
			if err != nil {
				return false, err
			}
			if m {
				modified = true
			}
		}
	}

	for c := n.FirstChild; c != nil; {
		next := c.NextSibling
		m, err := processAttachmentNodes(ctx, c, spaceID, finder)
		if err != nil {
			return false, err
		}
		if m {
			modified = true
		}
		c = next
	}

	return modified, nil
}

// processImgNode はimg要素の添付ファイルURLを変換する。
// インライン画像フォーマットの場合はリンク付き画像に、それ以外はダウンロードリンクに変換する。
func processImgNode(ctx context.Context, n *html.Node, spaceID model.SpaceID, finder AttachmentFinder) (bool, error) {
	src := getAttr(n, "src")
	id := extractAttachmentID(src)
	if id == "" {
		return false, nil
	}

	attachment, err := finder.FindByIDAndSpace(ctx, model.AttachmentID(id), spaceID)
	if err != nil {
		return false, err
	}
	if attachment == nil {
		return false, nil
	}

	ext := fileExtension(attachment.Filename)
	if inlineImageFormats[ext] {
		replaceWithInlineImage(n, id, attachment.Filename)
	} else {
		replaceWithDownloadLink(n, id, attachment.Filename)
	}

	return true, nil
}

// processANode はa要素の添付ファイルURLを変換する。
// インライン動画フォーマットの場合はvideo要素に、それ以外はdata属性付きリンクに変換する。
func processANode(ctx context.Context, n *html.Node, spaceID model.SpaceID, finder AttachmentFinder) (bool, error) {
	href := getAttr(n, "href")
	id := extractAttachmentID(href)
	if id == "" {
		return false, nil
	}

	attachment, err := finder.FindByIDAndSpace(ctx, model.AttachmentID(id), spaceID)
	if err != nil {
		return false, err
	}
	if attachment == nil {
		return false, nil
	}

	ext := fileExtension(attachment.Filename)
	if inlineVideoFormats[ext] {
		replaceWithInlineVideo(n, id)
	} else {
		replaceWithAnchorDataAttrs(n, id)
	}

	return true, nil
}

// replaceWithInlineImage はimg要素をリンク付きインライン画像に置換する。
// 親要素にa要素を挿入し、img要素をその子にする。
func replaceWithInlineImage(imgNode *html.Node, attachmentID, filename string) {
	width := getAttr(imgNode, "width")
	alt := getAttr(imgNode, "alt")
	if alt == "" {
		alt = filename
	}

	// img要素の属性を更新
	imgNode.Attr = []html.Attribute{
		{Key: "src", Val: ""},
		{Key: "data-attachment-id", Val: attachmentID},
		{Key: "data-attachment-type", Val: "image"},
		{Key: "alt", Val: alt},
		{Key: "class", Val: "wikino-attachment-image"},
	}
	if width != "" {
		imgNode.Attr = append(imgNode.Attr, html.Attribute{Key: "width", Val: width})
	}

	// a要素を作成してimg要素をラップ
	aNode := &html.Node{
		Type:     html.ElementNode,
		DataAtom: atom.A,
		Data:     "a",
		Attr: []html.Attribute{
			{Key: "href", Val: "#"},
			{Key: "data-attachment-id", Val: attachmentID},
			{Key: "data-attachment-link", Val: "true"},
			{Key: "target", Val: "_blank"},
			{Key: "rel", Val: "noopener noreferrer"},
			{Key: "class", Val: "wikino-attachment-image-link"},
		},
	}

	parent := imgNode.Parent
	parent.InsertBefore(aNode, imgNode)
	parent.RemoveChild(imgNode)
	aNode.AppendChild(imgNode)
}

// downloadLinkSVG はダウンロードリンクのSVGアイコン
const downloadLinkSVG = `<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">` +
	`<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"` +
	` d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />` +
	`</svg>`

// replaceWithDownloadLink はimg要素をダウンロードリンクに置換する。
func replaceWithDownloadLink(imgNode *html.Node, attachmentID, filename string) {
	aNode := &html.Node{
		Type:     html.ElementNode,
		DataAtom: atom.A,
		Data:     "a",
		Attr: []html.Attribute{
			{Key: "href", Val: "#"},
			{Key: "data-attachment-id", Val: attachmentID},
			{Key: "data-attachment-link", Val: "true"},
			{Key: "target", Val: "_blank"},
			{Key: "rel", Val: "noopener noreferrer"},
			{Key: "class", Val: "inline-flex items-center gap-1 text-blue-600 hover:text-blue-800 underline"},
		},
	}

	// SVGはhtml.Renderの外部コンテンツモード問題を回避するためテキストノードとして挿入
	svgText := &html.Node{
		Type: html.RawNode,
		Data: downloadLinkSVG,
	}
	aNode.AppendChild(svgText)

	filenameText := &html.Node{
		Type: html.TextNode,
		Data: filename,
	}
	aNode.AppendChild(filenameText)

	parent := imgNode.Parent
	parent.InsertBefore(aNode, imgNode)
	parent.RemoveChild(imgNode)
}

// replaceWithInlineVideo はa要素をインライン動画に置換する。
func replaceWithInlineVideo(aNode *html.Node, attachmentID string) {
	videoNode := &html.Node{
		Type:     html.ElementNode,
		DataAtom: atom.Video,
		Data:     "video",
		Attr: []html.Attribute{
			{Key: "src", Val: ""},
			{Key: "data-attachment-id", Val: attachmentID},
			{Key: "data-attachment-type", Val: "video"},
			{Key: "class", Val: "wikino-attachment-video"},
			{Key: "controls", Val: ""},
		},
	}

	fallbackA := &html.Node{
		Type:     html.ElementNode,
		DataAtom: atom.A,
		Data:     "a",
		Attr: []html.Attribute{
			{Key: "href", Val: "#"},
			{Key: "data-attachment-id", Val: attachmentID},
			{Key: "data-attachment-link", Val: "true"},
			{Key: "target", Val: "_blank"},
		},
	}
	fallbackText := &html.Node{
		Type: html.TextNode,
		Data: attachmentID,
	}
	fallbackA.AppendChild(fallbackText)
	videoNode.AppendChild(fallbackA)

	parent := aNode.Parent
	parent.InsertBefore(videoNode, aNode)
	parent.RemoveChild(aNode)
}

// replaceWithAnchorDataAttrs はa要素の内容を保持しつつdata属性付きリンクに変換する。
func replaceWithAnchorDataAttrs(aNode *html.Node, attachmentID string) {
	aNode.Attr = []html.Attribute{
		{Key: "href", Val: "#"},
		{Key: "data-attachment-id", Val: attachmentID},
		{Key: "data-attachment-link", Val: "true"},
		{Key: "target", Val: "_blank"},
		{Key: "rel", Val: "noopener noreferrer"},
	}
}

// extractAttachmentID はURLから添付ファイルIDを抽出する。
// /attachments/{id}パターンにマッチしない場合は空文字列を返す。
// bluemondayがURL内の特殊文字（例: \）をパーセントエンコード（%5C）するため、
// URLデコードしてから抽出する。
func extractAttachmentID(rawURL string) string {
	decoded, err := url.PathUnescape(rawURL)
	if err == nil {
		rawURL = decoded
	}
	match := attachmentPathRegex.FindStringSubmatch(rawURL)
	if match == nil {
		return ""
	}
	return match[1]
}

// WrapStandaloneImageLinks は<p>要素で囲まれていない独立した
// a.wikino-attachment-image-link要素を<p>要素で囲む。
// 画像リンクの直後にインライン要素（<br>, <em>等）が続く場合はラッピングしない。
func WrapStandaloneImageLinks(bodyHTML string) string {
	if bodyHTML == "" {
		return ""
	}

	container, err := parseHTMLFragmentWithContainer(bodyHTML)
	if err != nil {
		return bodyHTML
	}

	modified := wrapImageLinksInNode(container)
	if !modified {
		return bodyHTML
	}

	return renderContainerChildren(container)
}

// wrapImageLinksInNode はノードの子要素を走査し、
// スタンドアロンの画像リンクを<p>要素で囲む。
// 画像リンクの直後に<br>+<em>（キャプション）が続く場合はまとめて<p>で囲む。
func wrapImageLinksInNode(n *html.Node) bool {
	modified := false

	for c := n.FirstChild; c != nil; {
		next := c.NextSibling

		if isAttachmentImageLink(c) && !isParentParagraph(c) {
			if captionSiblings := getImageCaptionSiblings(c); captionSiblings != nil {
				lastSibling := captionSiblings[len(captionSiblings)-1]
				next = lastSibling.NextSibling
				wrapInParagraphWithSiblings(c, captionSiblings)
				modified = true
			} else if !hasInlineNextSibling(c) {
				wrapInParagraph(c)
				modified = true
			}
		} else if c.Type == html.ElementNode && c.DataAtom != atom.P {
			if wrapImageLinksInNode(c) {
				modified = true
			}
		}

		c = next
	}

	return modified
}

// isAttachmentImageLink はノードがa.wikino-attachment-image-link要素かを判定する
func isAttachmentImageLink(n *html.Node) bool {
	if n.Type != html.ElementNode || n.DataAtom != atom.A {
		return false
	}
	return strings.Contains(getAttr(n, "class"), "wikino-attachment-image-link")
}

// isParentParagraph はノードの親が<p>要素かを判定する
func isParentParagraph(n *html.Node) bool {
	return n.Parent != nil && n.Parent.Type == html.ElementNode && n.Parent.DataAtom == atom.P
}

// hasInlineNextSibling は次の非空白兄弟ノードがインライン要素かを判定する
func hasInlineNextSibling(n *html.Node) bool {
	for sibling := n.NextSibling; sibling != nil; sibling = sibling.NextSibling {
		if sibling.Type == html.TextNode && strings.TrimSpace(sibling.Data) == "" {
			continue
		}
		if sibling.Type == html.ElementNode {
			switch sibling.DataAtom {
			case atom.Br, atom.Em, atom.Strong, atom.Span, atom.Code:
				return true
			}
		}
		break
	}
	return false
}

// getImageCaptionSiblings は画像リンクの直後にキャプションパターン（<br> + <em>/<strong>）が
// 続くかチェックし、キャプションを構成するノード群を返す。パターンが見つからない場合はnilを返す。
func getImageCaptionSiblings(n *html.Node) []*html.Node {
	var nodes []*html.Node
	sibling := n.NextSibling

	for sibling != nil && sibling.Type == html.TextNode && strings.TrimSpace(sibling.Data) == "" {
		nodes = append(nodes, sibling)
		sibling = sibling.NextSibling
	}

	if sibling == nil || sibling.Type != html.ElementNode || sibling.DataAtom != atom.Br {
		return nil
	}
	nodes = append(nodes, sibling)
	sibling = sibling.NextSibling

	for sibling != nil && sibling.Type == html.TextNode && strings.TrimSpace(sibling.Data) == "" {
		nodes = append(nodes, sibling)
		sibling = sibling.NextSibling
	}

	if sibling == nil || sibling.Type != html.ElementNode {
		return nil
	}
	if sibling.DataAtom != atom.Em && sibling.DataAtom != atom.Strong {
		return nil
	}
	nodes = append(nodes, sibling)

	return nodes
}

// wrapInParagraphWithSiblings はノードと後続のキャプションノード群をまとめて<p>要素で囲む
func wrapInParagraphWithSiblings(n *html.Node, siblings []*html.Node) {
	p := &html.Node{
		Type:     html.ElementNode,
		DataAtom: atom.P,
		Data:     "p",
	}
	n.Parent.InsertBefore(p, n)
	n.Parent.RemoveChild(n)
	p.AppendChild(n)
	for _, s := range siblings {
		s.Parent.RemoveChild(s)
		p.AppendChild(s)
	}
}

// wrapInParagraph はノードを<p>要素で囲む
func wrapInParagraph(n *html.Node) {
	p := &html.Node{
		Type:     html.ElementNode,
		DataAtom: atom.P,
		Data:     "p",
	}
	n.Parent.InsertBefore(p, n)
	n.Parent.RemoveChild(n)
	p.AppendChild(n)
}

// fileExtension はファイル名から拡張子を小文字で取得する（ドット無し）
func fileExtension(filename string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		return ""
	}
	return strings.ToLower(ext[1:])
}
