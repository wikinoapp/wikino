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

	// 1行目のHTML img形式: <img src="/attachments/id"> (大文字小文字不問)
	featuredHTMLImgRegex = regexp.MustCompile(`(?i)<img[^>]+src=["']/attachments/([^/"']+)["'][^>]*>`)
)

// ExtractAttachmentIDsはMarkdown本文が参照している添付ファイルを、重複を除いて最初に
// 現れる順で返す。
//
// 何を参照とするかはScanAttachmentRefMatchesから得るため、こことエクスポートで同じ定義を使う。
// 本文がMarkdownのコードとして見せているパスや、読み手に何も届かない場所に書かれたパスは、
// ここでも含めない。画面上でその添付ファイルを指すものが無いためである。
func ExtractAttachmentIDs(body string) []string {
	return attachmentIDsOf(ScanAttachmentRefMatches(body))
}

// attachmentIDsOfはmatchesが指す添付ファイルを、重複を除いて最初に現れる順で返す。
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

// ExtractFeaturedImageIDはMarkdown本文の1行目から画像IDを抽出する。
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

	// 1. Markdown画像形式をチェック (優先)
	if match := featuredMarkdownImgRegex.FindStringSubmatch(firstLine); match != nil {
		return &match[1]
	}

	// 2. HTML img要素をチェック
	if match := featuredHTMLImgRegex.FindStringSubmatch(firstLine); match != nil {
		return &match[1]
	}

	return nil
}

// attachmentPathPrefixは、本文がスペースの添付ファイルを指すときに使うパスの先頭。
const attachmentPathPrefix = "/attachments/"

// escapeMarkersは、Markdownのエスケープまたは文字参照が始まるバイト。どちらも持たない本文
// は、どのリンク先も読まれるとおりのバイトで書いている。
const escapeMarkers = `\&`

// attachmentHTMLRefRegexはHTMLのimg要素のsrc属性とa要素のhref属性を、ソース位置を
// 保って見つける。周囲の要素も照合することで、data-src= のように末尾がsrc=・href=
// になっているだけの属性を参照にしない。各キャプチャグループが1つのリンク先で、どの引用符と
// どの要素で書かれたかをグループで見分ける。引用符のない属性値も対象とする。
//
// Markdownのリンク・画像は代わりに解析済みのドキュメントから見つける (scanMarkdownAttachmentRefs)。
// ラベルはエスケープされた角括弧や対応の取れた角括弧を持てるため、正規表現では表せない。
var attachmentHTMLRefRegex = regexp.MustCompile(`(?i)(?:<img\b(?:[^>"']|"[^"]*"|'[^']*')*?[\t\n\f\r ]src[\t\n\f\r ]*=[\t\n\f\r ]*(?:"([^"]*)"|'([^']*)'|([^\t\n\f\r "'<=>\x60]+))|<a\b(?:[^>"']|"[^"]*"|'[^']*')*?[\t\n\f\r ]href[\t\n\f\r ]*=[\t\n\f\r ]*(?:"([^"]*)"|'([^']*)'|([^\t\n\f\r "'<=>\x60]+)))`)

// attachmentHTMLRefGroupsはattachmentHTMLRefRegexのうちリンク先を持つキャプチャグループ。
// 1つのマッチではこのうち1つだけが埋まる。
var attachmentHTMLRefGroups = []int{1, 2, 3, 4, 5, 6}

// AttachmentRefMatchはMarkdown本文の添付ファイルへの参照1件と、本文中でそのリンク先が
// 占めるバイト範囲。
type AttachmentRefMatch struct {
	// AttachmentIDはリンク先が指す添付ファイル
	AttachmentID model.AttachmentID

	// Startはリンク先の先頭のバイト位置
	Start int

	// Stopはリンク先の末尾の次のバイト位置
	Stop int

	// InHTMLAttributeは、リンク先がMarkdownのリンク・画像のリンク先ではなくHTMLの
	// 属性値であるかを表す。属性値である場合、リンク先を書き換える呼び出し元はHTML用の
	// エスケープを行う必要がある。
	InHTMLAttribute bool
}

// attachmentRefは参照1件と、リンク先が属する要素をレンダリングが開く位置の組。参照が
// 読み手に届くかどうかが決まるのは、リンク先そのものではなくその位置である。
type attachmentRef struct {
	match        AttachmentRefMatch
	elementStart int
}

// ScanAttachmentRefMatchesはMarkdown本文の添付ファイルへの参照を現れる順に返す。
// 呼び出し元が置き換えられるよう、各参照はリンク先が本文中で占める位置を持つ。本文がMarkdownの
// コードとして見せている参照は、ScanWikilinkMatchesがWikiリンクを外すのと同じように含めない。
// そこにあるリンク先は書き手が見せている文字列であって、たどり着く先のファイルではないためである。
// サニタイズが中身ごと落とす要素の中に書かれた参照も、読み手に届かないため含めない。raw text要素の
// 開始タグより後ろに書かれた参照も、レンダリングされたリンクが記法そのものとして読み手に届くため
// 含めない。
//
// raw HTMLのcode要素・pre要素はこれらに当たらない。その中のMarkdownはほかの場所と同じように
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

// holdsAttachmentPathは、本文が添付ファイルへの参照を持ちうるかを返す。パスを一度も
// 書いていない本文は参照を持たないため、解析しても何も見つからない。パスはMarkdownの
// エスケープや文字参照でも書け、走査はそれを解決するので、どちらかの記号を持つ本文は
// プレーンなパスが無くても解析する。
func holdsAttachmentPath(body string) bool {
	return strings.Contains(body, attachmentPathPrefix) || strings.ContainsAny(body, escapeMarkers)
}

// scanAttachmentRefMatchesは解析済みの本文から参照を読む。bodyHTMLは同じ本文の
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

	// 以降の手順は、候補のうちどれが読み手に届くかを決めるものなので、候補を1つも書いて
	// いない本文はここで答える。そうした本文の描画後ツリーを読んでも決まるものは無く、この走査
	// は表示側と共有する経路でページ一覧の本文ごとに行われる。
	if len(matches) == 0 {
		return nil
	}

	excludedRanges = append(excludedRanges, scanMarkdownCodeRanges(document)...)
	excludedRanges = normalizeByteRanges(excludedRanges)

	// 2つの走査はそれぞれ本文の順に進むが、入れ子のリンクでは外側のリンク先が内側のリンク先
	// より後ろに来る。並べ替えることで、リンク先を置き換える呼び出し元が本文を読む順に全体を戻す。
	slices.SortStableFunc(matches, func(a, b attachmentRef) int {
		return cmp.Compare(a.match.Start, b.match.Start)
	})

	live, liveKnown := liveAttachmentIDs(bodyHTML, rendered)

	var eligible []attachmentRef
	for _, ref := range matches {
		// 閉じられていないコメントやタグがあると、サニタイザーはその後ろに書かれたものを
		// すべて落とす。パーサーが先で別のノードとして認識したraw HTMLも同じである。これを
		// 決めるのは読み手のページである描画後のツリーで、その要素がどれも指していない添付
		// ファイルは、画面のどこからも参照されていない。
		if liveKnown && !live[ref.match.AttachmentID] {
			continue
		}
		// 参照が読み手に届くかは、リンク先が書かれた位置ではなく、その要素が開く位置で決まる。
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

// liveAttachmentIDsは、描画後のページが指している添付ファイルを、読み手が見るツリーの
// img要素・a要素から読み取って返す。ソースだけでは分からないこと、すなわちリンク先が属する
// 要素が少なくとも1つサニタイザーを通過したかどうかに答える。同じIDを持つ個々の参照元は
// visibleAttachmentRefsで区別する。
//
// 2つ目の戻り値は、その集合を読み取れたかどうかを表す。読み取れなかった場合、呼び出し元は
// すべての参照を残してソース側の範囲だけで判断する。参照を失うと、添付ファイルが孤児として
// 扱われないようにしている行が消えてしまい、2つの方向のうち間違えたときの害が大きいためである。
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

// scanMarkdownAttachmentRefsは、描画されるリンク・画像とリンク先のソースを対応づける。
// 参照リンクにはMarkdownパーサーと同じく、正規化したラベルの最初の定義を使う。
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

	// 下のウォーカーは失敗しないため、ast.Walkが返すエラーはnilにしかならない。
	_ = ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		walkStatus := ast.WalkContinue
		if node.Kind() == ast.KindImage {
			// 描画されるのは外側の画像のリンク先だけで、ラベルの子はaltテキストになる。
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

		// 置換する生の範囲とは別に、Markdownのエスケープと文字参照をレンダラーと同じ順で
		// 解決する。パーセント復号はattachmentIDOfPathだけが行う。
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

// scanHTMLAttachmentRefsは、スペースの添付ファイルを指すraw HTMLのimg要素・a要素の
// リンク先を返す。tagRangesはそれらの要素の開始タグ、すなわち属性がある場所である。本文全体では
// なくそこへ正規表現を当てることで、エスケープされた例、コメント、script要素・style要素内の
// raw textを結果から除外する。
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

// attachmentIDOfPathは、リンク先が添付ファイルへのパスであれば、それが指す添付ファイルを
// 返す。呼び出し元がMarkdown・HTMLの構文を解決済みである。パーセントエスケープは1回だけ
// 復号し、不正なエスケープとパス区切りは拒否する。IDはパスの1要素だからである。
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
