package markup

import (
	"cmp"
	"log/slog"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/yuin/goldmark/ast"
	gmtext "github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/wikinoapp/wikino/go/internal/model"
)

var (
	// 1行目のMarkdown画像形式: ![alt](/attachments/id) or ![alt](/attachments/id "title")
	featuredMarkdownImgRegex = regexp.MustCompile(`!\[[^\]]*\]\(/attachments/([^\s/)]+)(\s[^)]*)?\)`)

	// 1行目のHTML img形式: <img src="/attachments/id">（大文字小文字不問）
	featuredHTMLImgRegex = regexp.MustCompile(`(?i)<img[^>]+src=["']/attachments/([^/"']+)["'][^>]*>`)
)

// ExtractAttachmentIDs returns the attachments a Markdown body references, without duplicates and
// in the order they first appear.
//
// What counts as a reference comes from ScanAttachmentRefMatches, so the same definition serves
// this and the export. A path the body shows as Markdown code, or writes where nothing of it
// reaches the reader, is left out here as well: nothing on the screen points at that attachment.
//
// [Ja] ExtractAttachmentIDs は Markdown 本文が参照している添付ファイルを、重複を除いて最初に
// 現れる順で返す。
//
// 何を参照とするかは ScanAttachmentRefMatches から得るため、こことエクスポートで同じ定義を使う。
// 本文が Markdown のコードとして見せているパスや、読み手に何も届かない場所に書かれたパスは、
// ここでも含めない。画面上でその添付ファイルを指すものが無いためである。
func ExtractAttachmentIDs(body string) []string {
	return attachmentIDsOf(ScanAttachmentRefMatches(body))
}

// attachmentIDsOf returns the attachments the matches name, without duplicates and in the order
// they first appear.
//
// [Ja] attachmentIDsOf は matches が指す添付ファイルを、重複を除いて最初に現れる順で返す。
func attachmentIDsOf(matches []AttachmentRefMatch) []string {
	seen := make(map[string]bool)
	var ids []string

	for _, match := range matches {
		id := string(match.AttachmentID)
		if seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}

	return ids
}

// ExtractFeaturedImageID はMarkdown本文の1行目から画像IDを抽出する。
// 1行目にMarkdown画像形式またはHTML img要素がある場合、その添付ファイルIDを返す。
// 該当なしの場合はnilを返す。
func ExtractFeaturedImageID(body string) *string {
	if body == "" {
		return nil
	}

	firstLine := strings.TrimSpace(strings.SplitN(body, "\n", 2)[0])
	if firstLine == "" {
		return nil
	}

	// 1. Markdown画像形式をチェック（優先）
	if match := featuredMarkdownImgRegex.FindStringSubmatch(firstLine); match != nil {
		return &match[1]
	}

	// 2. HTML img要素をチェック
	if match := featuredHTMLImgRegex.FindStringSubmatch(firstLine); match != nil {
		return &match[1]
	}

	return nil
}

// attachmentPathPrefix opens the path a body uses to point at an attachment of the space.
//
// [Ja] attachmentPathPrefix は、本文がスペースの添付ファイルを指すときに使うパスの先頭。
const attachmentPathPrefix = "/attachments/"

// escapeMarkers are the bytes a Markdown escape or a character reference starts with. A body
// holding neither writes every destination as the bytes the destination is read as.
//
// [Ja] escapeMarkers は、Markdown のエスケープまたは文字参照が始まるバイト。どちらも持たない本文
// は、どのリンク先も読まれるとおりのバイトで書いている。
const escapeMarkers = `\&`

// attachmentHTMLRefRegex finds the src attribute of an HTML img element and the href attribute of
// an HTML a element, preserving their source positions. Matching the surrounding element
// keeps an attribute that merely ends in src= or href=, such as data-src=, from becoming a
// reference. Each capture group is one destination, and the group tells the quote and the element
// it was written with apart, including an unquoted attribute value.
//
// A Markdown link or image is found through the parsed document instead, in
// scanMarkdownAttachmentRefs: a label can hold escaped or balanced brackets, which no regular
// expression describes.
//
// [Ja] attachmentHTMLRefRegex は HTML の img 要素の src 属性と a 要素の href 属性を、ソース位置を
// 保って見つける。周囲の要素も照合することで、data-src= のように末尾が src=・href=
// になっているだけの属性を参照にしない。各キャプチャグループが 1 つのリンク先で、どの引用符と
// どの要素で書かれたかをグループで見分ける。引用符のない属性値も対象とする。
//
// Markdown のリンク・画像は代わりに解析済みのドキュメントから見つける (scanMarkdownAttachmentRefs)。
// ラベルはエスケープされた角括弧や対応の取れた角括弧を持てるため、正規表現では表せない。
var attachmentHTMLRefRegex = regexp.MustCompile(`(?i)(?:<img\b(?:[^>"']|"[^"]*"|'[^']*')*?[\t\n\f\r ]src[\t\n\f\r ]*=[\t\n\f\r ]*(?:"([^"]*)"|'([^']*)'|([^\t\n\f\r "'<=>\x60]+))|<a\b(?:[^>"']|"[^"]*"|'[^']*')*?[\t\n\f\r ]href[\t\n\f\r ]*=[\t\n\f\r ]*(?:"([^"]*)"|'([^']*)'|([^\t\n\f\r "'<=>\x60]+)))`)

// attachmentHTMLRefGroups are the capture groups of attachmentHTMLRefRegex that hold a
// destination. Exactly one of them is set on a match.
//
// [Ja] attachmentHTMLRefGroups は attachmentHTMLRefRegex のうちリンク先を持つキャプチャグループ。
// 1 つのマッチではこのうち 1 つだけが埋まる。
var attachmentHTMLRefGroups = []int{1, 2, 3, 4, 5, 6}

// AttachmentRefMatch is one reference to an attachment of a Markdown body, together with the bytes
// its destination occupies there.
//
// [Ja] AttachmentRefMatch は Markdown 本文の添付ファイルへの参照 1 件と、本文中でそのリンク先が
// 占めるバイト範囲。
type AttachmentRefMatch struct {
	// AttachmentID is the attachment the destination names.
	//
	// [Ja] AttachmentID はリンク先が指す添付ファイル
	AttachmentID model.AttachmentID

	// Start is the offset of the first byte of the destination.
	//
	// [Ja] Start はリンク先の先頭のバイト位置
	Start int

	// Stop is the offset just past its last byte.
	//
	// [Ja] Stop はリンク先の末尾の次のバイト位置
	Stop int

	// InHTMLAttribute reports whether the destination is the value of an HTML attribute rather
	// than the destination of a Markdown link or image. A caller that writes a new destination
	// has to escape it for HTML in the former case.
	//
	// [Ja] InHTMLAttribute は、リンク先が Markdown のリンク・画像のリンク先ではなく HTML の
	// 属性値であるかを表す。属性値である場合、リンク先を書き換える呼び出し元は HTML 用の
	// エスケープを行う必要がある。
	InHTMLAttribute bool
}

// attachmentRef is one reference together with the offset where rendering opens the element the
// destination belongs to. That offset, rather than the destination itself, is where the reference
// either reaches the reader or does not.
//
// [Ja] attachmentRef は参照 1 件と、リンク先が属する要素をレンダリングが開く位置の組。参照が
// 読み手に届くかどうかが決まるのは、リンク先そのものではなくその位置である。
type attachmentRef struct {
	match        AttachmentRefMatch
	elementStart int
}

// ScanAttachmentRefMatches returns the references to attachments of a Markdown body in the order
// they appear, each one carrying where its destination sits in the body so that a caller can
// replace it. A reference the body shows as Markdown code is left out, the same way
// ScanWikilinkMatches leaves out a wiki link there: the destination is a piece of text the author
// is showing, not a file to reach. So is one written inside an element sanitization drops together
// with its content, which reaches no reader, and one written after the start tag of a raw-text
// element, whose rendered link reaches the reader as the notation itself.
//
// A raw HTML code or pre element is not one of those. Markdown written inside one is parsed as it
// is anywhere else, so the destination there points at a live link or image on the screen and the
// archive has to carry it like any other.
//
// [Ja] ScanAttachmentRefMatches は Markdown 本文の添付ファイルへの参照を現れる順に返す。
// 呼び出し元が置き換えられるよう、各参照はリンク先が本文中で占める位置を持つ。本文が Markdown の
// コードとして見せている参照は、ScanWikilinkMatches が Wiki リンクを外すのと同じように含めない。
// そこにあるリンク先は書き手が見せている文字列であって、たどり着く先のファイルではないためである。
// サニタイズが中身ごと落とす要素の中に書かれた参照も、読み手に届かないため含めない。raw text 要素の
// 開始タグより後ろに書かれた参照も、レンダリングされたリンクが記法そのものとして読み手に届くため
// 含めない。
//
// raw HTML の code 要素・pre 要素はこれらに当たらない。その中の Markdown はほかの場所と同じように
// 解析されるため、そこにあるリンク先は画面上の生きたリンク・画像を指しており、アーカイブもほかと
// 同じように持つ必要がある。
func ScanAttachmentRefMatches(body string) []AttachmentRefMatch {
	if !holdsAttachmentPath(body) {
		return nil
	}

	source := []byte(body)
	document := md.Parser().Parse(gmtext.NewReader(source))
	bodyHTML, err := renderSanitized(source, document)
	if err != nil {
		slog.Warn("添付ファイル参照の走査でのMarkdown変換に失敗", "error", err)
	}

	return scanAttachmentRefMatches(source, document, bodyHTML, err == nil)
}

// holdsAttachmentPath reports whether a body can hold a reference to an attachment at all. A body
// that never writes the path holds none, so parsing it would find nothing. The path can also be
// written with Markdown escapes or character references, which the scan resolves, so a body
// holding either marker is parsed even when the plain path is absent.
//
// [Ja] holdsAttachmentPath は、本文が添付ファイルへの参照を持ちうるかを返す。パスを一度も
// 書いていない本文は参照を持たないため、解析しても何も見つからない。パスは Markdown の
// エスケープや文字参照でも書け、走査はそれを解決するので、どちらかの記号を持つ本文は
// プレーンなパスが無くても解析する。
func holdsAttachmentPath(body string) bool {
	return strings.Contains(body, attachmentPathPrefix) || strings.ContainsAny(body, escapeMarkers)
}

// scanAttachmentRefMatches reads the references of an already parsed body. bodyHTML is the
// sanitized HTML of the same body, which a caller that has rendered it hands over rather than
// repeating the baseline rendering; rendered says whether it is the real thing. Repeated IDs
// additionally use a marked rendering to distinguish their individual use sites.
//
// [Ja] scanAttachmentRefMatches は解析済みの本文から参照を読む。bodyHTML は同じ本文の
// サニタイズ済みHTMLで、呼び出し元が通常の描画を繰り返さずに済むよう渡す。
// renderedは実際の描画結果かどうかを表す。重複IDは参照元を区別するマーカー付き描画も行う。
func scanAttachmentRefMatches(
	source []byte,
	document ast.Node,
	bodyHTML string,
	rendered bool,
) []AttachmentRefMatch {
	excludedRanges, tagRanges := scanRawHTMLRanges(source, rawHTMLScanOptions{
		rawHTMLRanges: rawHTMLNodeRanges(document),
		protectElement: func(_ html.TokenType, name string, _ []byte) bool {
			return rawTextDiscardedElements[name] || rawTextEscapedElements[name]
		},
		discardElement: func(name string) bool { return contentDiscardedElements[name] },
		acceptToken: func(tokenType html.TokenType, name string) bool {
			return (tokenType == html.StartTagToken || tokenType == html.SelfClosingTagToken) &&
				(name == "img" || name == "a")
		},
	})

	matches := scanMarkdownAttachmentRefs(source, document)
	matches = append(matches, scanHTMLAttachmentRefs(source, tagRanges)...)

	// The steps below decide which of the candidates reach the reader, so a body writing none of
	// them is answered here. Reading the rendered tree of such a body would settle nothing, and
	// the path this scan shares with display runs it for every body of a page list.
	//
	// [Ja] 以降の手順は、候補のうちどれが読み手に届くかを決めるものなので、候補を 1 つも書いて
	// いない本文はここで答える。そうした本文の描画後ツリーを読んでも決まるものは無く、この走査
	// は表示側と共有する経路でページ一覧の本文ごとに行われる。
	if len(matches) == 0 {
		return nil
	}

	excludedRanges = append(excludedRanges, scanMarkdownCodeRanges(document)...)
	excludedRanges = normalizeByteRanges(excludedRanges)

	// The two scans each run in the order of the body, but a nested link puts the destination of
	// the outer one after the destination of the inner one. Sorting brings the whole set back into
	// the order a caller replacing the destinations reads the body in.
	//
	// [Ja] 2 つの走査はそれぞれ本文の順に進むが、入れ子のリンクでは外側のリンク先が内側のリンク先
	// より後ろに来る。並べ替えることで、リンク先を置き換える呼び出し元が本文を読む順に全体を戻す。
	slices.SortStableFunc(matches, func(a, b attachmentRef) int {
		return cmp.Compare(a.match.Start, b.match.Start)
	})

	live, liveKnown := liveAttachmentIDs(bodyHTML, rendered)

	var eligible []attachmentRef
	for _, ref := range matches {
		// An unterminated comment or tag makes the sanitizer drop everything written after it,
		// including raw HTML the parser recognized as its own node further on. The rendered tree
		// settles that, since it is the reader's page: an attachment no element of it points at is
		// referenced nowhere on the screen.
		//
		// [Ja] 閉じられていないコメントやタグがあると、サニタイザーはその後ろに書かれたものを
		// すべて落とす。パーサーが先で別のノードとして認識した raw HTML も同じである。これを
		// 決めるのは読み手のページである描画後のツリーで、その要素がどれも指していない添付
		// ファイルは、画面のどこからも参照されていない。
		if liveKnown && !live[ref.match.AttachmentID] {
			continue
		}
		// Whether the reference reaches the reader is decided where its element opens, not where
		// the destination is written. A label can hold the start tag of an element sanitization
		// drops together with its content, and the element the destination belongs to has already
		// opened before it.
		//
		// [Ja] 参照が読み手に届くかは、リンク先が書かれた位置ではなく、その要素が開く位置で決まる。
		// ラベルはサニタイズが中身ごと落とす要素の開始タグを持てるが、リンク先が属する要素は
		// その手前で既に開いている。
		if byteRangesHold(excludedRanges, ref.elementStart) {
			continue
		}

		eligible = append(eligible, ref)
	}

	visible, known := visibleAttachmentRefs(source, document, eligible)
	var kept []AttachmentRefMatch
	previousStop := 0
	for _, ref := range eligible {
		if ref.match.Start < previousStop || (known && !visible[ref.elementStart]) {
			continue
		}
		kept = append(kept, ref.match)
		previousStop = ref.match.Stop
	}

	return kept
}

// liveAttachmentIDs returns the attachments the rendered page points at, read from the img and a
// elements of the tree the reader sees. It answers what the source alone cannot: whether the
// sanitizer kept any element pointing at an attachment. Individual use sites sharing an ID are
// distinguished by visibleAttachmentRefs.
//
// The second result reports whether the set could be read at all. Where it could not, the caller
// keeps every reference and lets the source-side ranges decide on their own: losing one would drop
// the row that keeps an attachment from being treated as orphaned, which is the worse of the two
// directions to be wrong in.
//
// [Ja] liveAttachmentIDs は、描画後のページが指している添付ファイルを、読み手が見るツリーの
// img 要素・a 要素から読み取って返す。ソースだけでは分からないこと、すなわちリンク先が属する
// 要素が少なくとも1つサニタイザーを通過したかどうかに答える。同じIDを持つ個々の参照元は
// visibleAttachmentRefsで区別する。
//
// 2 つ目の戻り値は、その集合を読み取れたかどうかを表す。読み取れなかった場合、呼び出し元は
// すべての参照を残してソース側の範囲だけで判断する。参照を失うと、添付ファイルが孤児として
// 扱われないようにしている行が消えてしまい、2 つの方向のうち間違えたときの害が大きいためである。
func liveAttachmentIDs(bodyHTML string, rendered bool) (map[model.AttachmentID]bool, bool) {
	if !rendered {
		return nil, false
	}

	container, err := parseHTMLFragmentWithContainer(bodyHTML)
	if err != nil {
		slog.Warn("描画済み添付ファイル参照の読み取りに失敗", "error", err)

		return nil, false
	}

	live := map[model.AttachmentID]bool{}
	var walk func(node *html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			attribute := ""
			switch node.DataAtom {
			case atom.Img:
				attribute = "src"
			case atom.A:
				attribute = "href"
			}
			if attribute != "" {
				if id, ok := attachmentIDOfPath(getAttr(node, attribute)); ok {
					live[id] = true
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(container)

	return live, true
}

// scanMarkdownAttachmentRefs pairs each rendered link or image with the source of its destination.
// Reference links use the first definition of their normalized label, just as the Markdown parser
// does. Visibility belongs to the use site; rewriting belongs to the definition. Unused definitions
// render no reference and are therefore left alone.
//
// [Ja] scanMarkdownAttachmentRefs は、描画されるリンク・画像とリンク先のソースを対応づける。
// 参照リンクには Markdown パーサーと同じく、正規化したラベルの最初の定義を使う。
// 表示可否は使用箇所で判定し、書き換えは定義で行う。未使用の定義は参照を描画しないため変更しない。
func scanMarkdownAttachmentRefs(source []byte, document ast.Node) []attachmentRef {
	var matches []attachmentRef
	linkRanges := scanMarkdownLinkRanges(source, document)
	definitions := make(map[string]byteRange)
	_ = ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if definition, ok := node.(*ast.LinkReferenceDefinition); entering && ok {
			label := util.ToLinkReference(definition.Label)
			if _, found := definitions[label]; !found {
				definitions[label], _ = linkReferenceDefinitionDestinationRange(source, node)
			}
		}
		return ast.WalkContinue, nil
	})

	// The walker below never fails, so the error ast.Walk returns can only be nil.
	//
	// [Ja] 下のウォーカーは失敗しないため、ast.Walk が返すエラーは nil にしかならない。
	_ = ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		walkStatus := ast.WalkContinue
		if node.Kind() == ast.KindImage {
			// Only the outer image destination is rendered; label children become alt text.
			//
			// [Ja] 描画されるのは外側の画像のリンク先だけで、ラベルの子は alt テキストになる。
			walkStatus = ast.WalkSkipChildren
		}

		var sourceRange byteRange
		var elementStart int
		var ok bool

		switch node.Kind() {
		case ast.KindLink, ast.KindImage:
			ranges := linkRanges[node]
			sourceRange, ok = ranges.destination, ranges.hasDestination
			elementStart = ranges.node.start
			link, _ := markdownLinkOf(node)
			if link.reference != nil {
				sourceRange, ok = definitions[util.ToLinkReference(link.reference.Value)]
				ok = ok && sourceRange.start < sourceRange.stop
			}
		default:
			return walkStatus, nil
		}

		if !ok {
			return walkStatus, nil
		}

		// Resolve Markdown escapes and character references in the renderer's order, separately
		// from the raw replacement range. Percent decoding belongs to attachmentIDOfPath alone.
		//
		// [Ja] 置換する生の範囲とは別に、Markdown のエスケープと文字参照をレンダラーと同じ順で
		// 解決する。パーセント復号は attachmentIDOfPath だけが行う。
		destination := util.UnescapePunctuations(source[sourceRange.start:sourceRange.stop])
		destination = util.ResolveNumericReferences(destination)
		destination = util.ResolveEntityNames(destination)
		attachmentID, ok := attachmentIDOfPath(string(destination))
		if !ok {
			return walkStatus, nil
		}

		matches = append(matches, attachmentRef{
			match: AttachmentRefMatch{
				AttachmentID: attachmentID,
				Start:        sourceRange.start,
				Stop:         sourceRange.stop,
			},
			elementStart: elementStart,
		})

		return walkStatus, nil
	})

	return matches
}

// scanHTMLAttachmentRefs returns the destinations of raw HTML img and a elements that point at an
// attachment of the space. tagRanges holds the start tags of those elements, which is where the
// attributes are: applying the regular expression there rather than to the whole body keeps escaped
// examples, comments, and raw text inside script and style elements out of the result.
// The tokenizer resolves character references using attribute-value rules, while the regular
// expression locates only the first destination attribute in the original token.
//
// [Ja] scanHTMLAttachmentRefs は、スペースの添付ファイルを指す raw HTML の img 要素・a 要素の
// リンク先を返す。tagRanges はそれらの要素の開始タグ、すなわち属性がある場所である。本文全体では
// なくそこへ正規表現を当てることで、エスケープされた例、コメント、script 要素・style 要素内の
// raw text を結果から除外する。
// トークナイザーは属性値の規則で文字参照を解決し、正規表現は元トークンの最初のリンク先属性の
// 位置だけを求める。
func scanHTMLAttachmentRefs(source []byte, tagRanges []byteRange) []attachmentRef {
	var matches []attachmentRef

	for _, tagRange := range tagRanges {
		tag := string(source[tagRange.start:tagRange.stop])
		tokenizer := html.NewTokenizer(strings.NewReader(tag))
		tokenizer.Next()
		token := tokenizer.Token()
		attributeName := "src"
		if token.Data == "a" {
			attributeName = "href"
		}
		var destination string
		for _, attribute := range token.Attr {
			if attribute.Key == attributeName {
				destination = attribute.Val
				break
			}
		}
		attachmentID, ok := attachmentIDOfPath(destination)
		if !ok {
			continue
		}

		loc := attachmentHTMLRefRegex.FindStringSubmatchIndex(tag)
		if loc == nil {
			continue
		}
		for _, group := range attachmentHTMLRefGroups {
			relativeStart, relativeStop := loc[group*2], loc[group*2+1]
			if relativeStart < 0 {
				continue
			}

			matches = append(matches, attachmentRef{
				match: AttachmentRefMatch{
					AttachmentID:    attachmentID,
					Start:           tagRange.start + relativeStart,
					Stop:            tagRange.start + relativeStop,
					InHTMLAttribute: true,
				},
				elementStart: tagRange.start,
			})
			break
		}
	}

	return matches
}

// attachmentIDOfPath returns the attachment a destination names, when the destination is the path
// to one. The caller has already resolved Markdown/HTML syntax. Decode percent escapes exactly
// once, rejecting malformed escapes and path separators: the ID is a single path segment.
//
// [Ja] attachmentIDOfPath は、リンク先が添付ファイルへのパスであれば、それが指す添付ファイルを
// 返す。呼び出し元が Markdown・HTML の構文を解決済みである。パーセントエスケープは 1 回だけ
// 復号し、不正なエスケープとパス区切りは拒否する。ID はパスの 1 要素だからである。
func attachmentIDOfPath(destination string) (model.AttachmentID, bool) {
	id, found := strings.CutPrefix(destination, attachmentPathPrefix)
	if !found || id == "" {
		return "", false
	}
	id, err := url.PathUnescape(id)
	if err != nil || strings.ContainsAny(id, `/\`) {
		return "", false
	}

	return model.AttachmentID(id), true
}
