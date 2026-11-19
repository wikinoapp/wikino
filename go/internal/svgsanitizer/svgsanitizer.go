// Package svgsanitizerは、利用者がアップロードしたSVGから、スクリプトと外部のSVG・CSSへの参照を取り除く。
// image要素のHTTP(S)の画像参照は許可するため、表示したときに参照先へ通信することがある。
//
// SVGをXMLとして読み、許可リストにある要素と属性だけを書き直す。ブラウザはimage/svg+xmlを
// XMLとして解釈し、XMLとして正しくないSVGは表示しないため、読めない内容は直さずにエラーにする。
// 要素と属性の名前は大文字と小文字を区別して照合し、書き出しでもそのまま保つ (viewBoxなど)。
package svgsanitizer

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"golang.org/x/net/html/charset"
)

// ErrMalformedは、内容をSVGとして読めないことを表す。同じ内容で何度試しても結果は変わらない
var ErrMalformed = errors.New("SVGとして解釈できません")

const (
	svgNamespace   = "http://www.w3.org/2000/svg"
	xlinkNamespace = "http://www.w3.org/1999/xlink"
)

// entityDeclarationは、DOCTYPEの内部サブセットにある、値を直接書いた一般実体の宣言。
// Adobe Illustratorが名前空間のURIを実体で書くため、SVG・XLinkとAdobeの名前空間のURIを値に持つものだけを展開する。
// それ以外と外部の実体 (SYSTEM・PUBLIC) は登録せず、参照されたら未定義の実体としてエラーにする
var entityDeclaration = regexp.MustCompile(`<!ENTITY\s+([^\s%"'<>]+)\s+(?:"([^"]*)"|'([^']*)')\s*>`)

// adobeNamespaceは、Illustratorが「Illustratorの編集機能を保持」で宣言するAdobeの名前空間のURI
// (http://ns.adobe.com/AdobeIllustrator/10.0/ など) の形
var adobeNamespace = regexp.MustCompile(`^http://ns\.adobe\.com/[A-Za-z0-9._/-]+$`)

// maxAdobeNamespaceLengthは、実体として登録するAdobeの名前空間のURIの長さの上限。Illustratorが書く最長の
// http://ns.adobe.com/GenericCustomNamespace/1.0/ (47文字) が収まり、参照1つあたりの展開を短く抑える
const maxAdobeNamespaceLength = 64

// Sanitizeは、SVGのdataから許可リストに無い要素と属性を取り除いたSVGを、UTF-8で返す。
// script・style・foreignObjectとSVG以外の名前空間の要素は中身ごと落とし、それ以外の許可しない
// 要素はタグだけを落として子を残す。コメント・処理命令・DOCTYPEは落とす。
// 外部のSVG・CSSへの参照は落とすが、image要素のHTTP(S)の画像参照は許可する。
// 結果をもう一度渡しても同じバイト列が返る。SVGとして読めなければErrMalformedを包んだエラーを返す
func Sanitize(data []byte) ([]byte, error) {
	d := xml.NewDecoder(bytes.NewReader(data))
	d.CharsetReader = charset.NewReaderLabel
	d.Entity = map[string]string{}

	w := &writer{}
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrMalformed, err)
		}
		done, err := w.handle(d, tok)
		if err != nil {
			return nil, err
		}
		if done {
			return w.bytes(), nil
		}
	}
	if w.depth == 0 {
		return nil, fmt.Errorf("%w: svg要素がありません", ErrMalformed)
	}
	return nil, fmt.Errorf("%w: svg要素が閉じていません", ErrMalformed)
}

// writerは、読んだトークンのうち残すものを書き出す
type writer struct {
	// rootは、最上位のsvg要素の開始タグの属性 (名前空間の宣言を除く)
	root bytes.Buffer
	// bodyは、最上位のsvg要素の中身
	body bytes.Buffer
	// stackは、開いている要素ごとの、書き出した名前 (タグだけを落とした要素は空文字列)
	stack []string
	// depthは、最上位のsvg要素から数えた、開いている要素の深さ
	depth int
	// openは、bodyに書いた開始タグを、まだ「>」で閉じていないことを表す。
	// 中身が無ければ「/>」で閉じる
	open bool
	// usesXLinkは、xlink:href属性を書き出したことを表す
	usesXLink bool
}

// handleは、トークンtokを処理する。最上位のsvg要素が閉じたらtrueを返す
func (w *writer) handle(d *xml.Decoder, tok xml.Token) (bool, error) {
	switch t := tok.(type) {
	case xml.StartElement:
		return false, w.start(d, t)
	case xml.EndElement:
		return w.end(), nil
	case xml.CharData:
		if w.depth > 0 {
			w.closeOpenTag()
			writeEscaped(&w.body, string(t), false)
		}
	case xml.Directive:
		if w.depth == 0 {
			declareEntities(d, t)
		}
	}
	// コメントと処理命令は書き出さない。xml-stylesheetの処理命令は外部のスタイルシートを読むため
	return false, nil
}

// startは、開始タグtを処理する
func (w *writer) start(d *xml.Decoder, t xml.StartElement) error {
	if w.depth == 0 {
		if !isSVGName(t.Name) || t.Name.Local != "svg" {
			return fmt.Errorf("%w: 最上位の要素がsvgではありません", ErrMalformed)
		}
		w.depth = 1
		w.stack = append(w.stack, t.Name.Local)
		w.writeAttributes(&w.root, t)
		return nil
	}

	_, allowed := allowedElements[t.Name.Local]
	_, dropped := droppedElements[t.Name.Local]
	if !isSVGName(t.Name) || dropped {
		if err := d.Skip(); err != nil {
			return fmt.Errorf("%w: %w", ErrMalformed, err)
		}
		return nil
	}

	w.depth++
	if !allowed {
		w.stack = append(w.stack, "")
		return nil
	}
	w.stack = append(w.stack, t.Name.Local)
	w.closeOpenTag()
	w.body.WriteByte('<')
	w.body.WriteString(t.Name.Local)
	w.writeAttributes(&w.body, t)
	w.open = true
	return nil
}

// endは、終了タグを処理する。最上位のsvg要素が閉じたらtrueを返す
func (w *writer) end() bool {
	if w.depth == 0 {
		return false
	}
	w.depth--
	name := w.stack[len(w.stack)-1]
	w.stack = w.stack[:len(w.stack)-1]
	if w.depth == 0 {
		return true
	}
	switch {
	case name == "":
	case w.open:
		w.body.WriteString("/>")
		w.open = false
	default:
		w.body.WriteString("</" + name + ">")
	}
	return false
}

// closeOpenTagは、閉じていない開始タグを「>」で閉じる
func (w *writer) closeOpenTag() {
	if w.open {
		w.body.WriteByte('>')
		w.open = false
	}
}

// bytesは、書き出したSVGを返す。名前空間は最上位のsvg要素でだけ宣言する
func (w *writer) bytes() []byte {
	var out bytes.Buffer
	out.WriteString(`<svg xmlns="` + svgNamespace + `"`)
	if w.usesXLink {
		out.WriteString(` xmlns:xlink="` + xlinkNamespace + `"`)
	}
	out.Write(w.root.Bytes())
	if w.body.Len() == 0 {
		out.WriteString("/>")
		return out.Bytes()
	}
	out.WriteByte('>')
	out.Write(w.body.Bytes())
	out.WriteString("</svg>")
	return out.Bytes()
}

// isSVGNameは、要素の名前がSVGの名前空間にあるかを返す。名前空間を宣言していないSVGも、
// SVGとして扱う
func isSVGName(name xml.Name) bool {
	return name.Space == svgNamespace || name.Space == ""
}

// declareEntitiesは、SVG・XLinkとAdobeの名前空間のURIを値に持つ一般実体だけを登録する。
// 長い値の単純な繰り返しでもトークンが巨大化するため、任意の値や長い値はデコーダーに渡さない
func declareEntities(d *xml.Decoder, directive xml.Directive) {
	if !bytes.HasPrefix(directive, []byte("DOCTYPE")) {
		return
	}
	for _, m := range entityDeclaration.FindAllSubmatch(directive, -1) {
		value := m[2]
		if value == nil {
			value = m[3]
		}
		if namespace := string(value); isEntityNamespace(namespace) {
			d.Entity[string(m[1])] = namespace
		}
	}
}

// isEntityNamespaceは、実体の値vが、実体として展開してよい名前空間のURIかを返す
func isEntityNamespace(v string) bool {
	if v == svgNamespace || v == xlinkNamespace {
		return true
	}
	return len(v) <= maxAdobeNamespaceLength && adobeNamespace.MatchString(v)
}

// writeAttributesは、要素tの属性のうち許可するものを、要素の名前の後ろに続けて書く
func (w *writer) writeAttributes(buf *bytes.Buffer, t xml.StartElement) {
	element := t.Name.Local
	seen := make(map[string]struct{}, len(t.Attr))
	for _, attr := range t.Attr {
		name, ok := attributeName(attr.Name)
		if !ok || !isAllowedAttribute(element, name) || !isSafeValue(element, name, attr.Value) {
			continue
		}
		// 別の接頭辞で同じXLinkの名前空間を指すと、書き出しで同じ名前になるため、最初の1つだけを残す
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		if name == "xlink:href" {
			w.usesXLink = true
		}
		buf.WriteByte(' ')
		buf.WriteString(name)
		buf.WriteString(`="`)
		writeEscaped(buf, attr.Value, true)
		buf.WriteByte('"')
	}
}

// attributeNameは、属性の名前を、許可リストと書き出しで使う名前にする。名前空間の宣言と、
// 名前空間の無い属性とXLinkのhref以外は、falseを返す
func attributeName(name xml.Name) (string, bool) {
	switch name.Space {
	case "":
		if name.Local == "xmlns" {
			return "", false
		}
		return name.Local, true
	case xlinkNamespace:
		if name.Local == "href" {
			return "xlink:href", true
		}
	}
	return "", false
}

// isAllowedAttributeは、要素elementに属性nameを許すかを返す
func isAllowedAttribute(element, name string) bool {
	if _, ok := commonAttributes[name]; ok {
		return true
	}
	_, ok := elementAttributes[element][name]
	return ok
}

// isSafeValueは、属性の値が、スクリプトの実行や外部への参照にならないかを返す。
// 参照の属性は、imageならhttpとhttpsのURLだけを、それ以外は文書の中の要素だけを許す
func isSafeValue(element, name, value string) bool {
	if name == "href" || name == "xlink:href" {
		if element == "image" {
			return hasHTTPScheme(value)
		}
		return strings.HasPrefix(strings.TrimSpace(value), "#")
	}
	if name == "id" || name == "class" {
		return true
	}
	// CSSのエスケープで関数の名前を隠せないよう、バックスラッシュを含む値は落とす。
	// @importなどのat-ruleも落とす
	if strings.ContainsAny(value, `\@`) {
		return false
	}
	return allowedFunctionCalls(value)
}

// hasHTTPSchemeは、URLがhttpかhttpsで始まるかを返す
func hasHTTPScheme(value string) bool {
	v := strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://")
}

// functionCallは、値の中の関数の呼び出し (名前と、開き括弧の直後の位置)
var functionCall = regexp.MustCompile(`([A-Za-z-]*)\s*\(`)

// allowedFunctionCallsは、値の中の関数の呼び出しが、すべて許可した関数かを返す。
// urlは、文書の中の要素 (「#」で始まる参照) を指すものだけを許す
func allowedFunctionCalls(value string) bool {
	for _, m := range functionCall.FindAllStringSubmatchIndex(value, -1) {
		fn := strings.ToLower(value[m[2]:m[3]])
		if _, ok := allowedFunctions[fn]; !ok {
			return false
		}
		if fn == "url" {
			target := strings.TrimLeft(value[m[1]:], " \t\n\r\f\"'")
			if !strings.HasPrefix(target, "#") {
				return false
			}
		}
	}
	return true
}

// writeEscapedは、sをXMLの文字データとしてbufに書く。属性の値では、引用符と、属性の値の
// 正規化で失われる空白も文字参照にする
func writeEscaped(buf *bytes.Buffer, s string, attribute bool) {
	for _, r := range s {
		switch {
		case r == '&':
			buf.WriteString("&amp;")
		case r == '<':
			buf.WriteString("&lt;")
		case r == '>':
			buf.WriteString("&gt;")
		case attribute && r == '"':
			buf.WriteString("&quot;")
		case attribute && r == '\t':
			buf.WriteString("&#x9;")
		case attribute && r == '\n':
			buf.WriteString("&#xA;")
		case r == '\r':
			buf.WriteString("&#xD;")
		default:
			buf.WriteRune(r)
		}
	}
}
