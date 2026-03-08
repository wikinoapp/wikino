package markup

import (
	"net/url"
	"regexp"
	"strings"
)

var (
	// HTML img要素のsrc属性から添付ファイルIDを抽出
	extractImgSrcRegex = regexp.MustCompile(`<img[^>]+src=["'](/attachments/([^/"']+))["'][^>]*>`)

	// HTML a要素のhref属性から添付ファイルIDを抽出
	extractAHrefRegex = regexp.MustCompile(`<a[^>]+href=["'](/attachments/([^/"']+))["'][^>]*>`)

	// Markdown画像形式から添付ファイルIDを抽出: ![alt](/attachments/id) or ![alt](/attachments/id "title")
	extractMarkdownImgRegex = regexp.MustCompile(`!\[[^\]]*\]\(/attachments/([^\s/)]+)(\s[^)]*)?\)`)

	// Markdownリンク形式（画像含む）から添付ファイルIDを抽出: [text](/attachments/id)
	// Go正規表現は後読み非対応のため、マッチ後に直前文字で画像形式を除外する
	extractMarkdownLinkRegex = regexp.MustCompile(`\[[^\]]+\]\(/attachments/([^\s/)]+)(\s[^)]*)?\)`)

	// 1行目のMarkdown画像形式: ![alt](/attachments/id) or ![alt](/attachments/id "title")
	featuredMarkdownImgRegex = regexp.MustCompile(`!\[[^\]]*\]\(/attachments/([^\s/)]+)(\s[^)]*)?\)`)

	// 1行目のHTML img形式: <img src="/attachments/id">（大文字小文字不問）
	featuredHTMLImgRegex = regexp.MustCompile(`(?i)<img[^>]+src=["']/attachments/([^/"']+)["'][^>]*>`)
)

// ExtractAttachmentIDs はbodyHTMLから添付ファイルIDを抽出する。
// HTML img/aタグ、Markdown画像/リンクの4パターンを検索し、
// 重複を除いたIDのスライスを返す。
func ExtractAttachmentIDs(bodyHTML string) []string {
	seen := make(map[string]bool)
	var ids []string

	addID := func(id string) {
		// bluemondayがURL内の特殊文字をパーセントエンコードするため、デコードする
		if decoded, err := url.PathUnescape(id); err == nil {
			id = decoded
		}
		// バックスラッシュを含むIDは不正な入力として除外する
		if strings.ContainsRune(id, '\\') {
			return
		}
		if id != "" && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}

	// 1. HTML img要素のsrc属性から抽出
	for _, match := range extractImgSrcRegex.FindAllStringSubmatch(bodyHTML, -1) {
		addID(match[2])
	}

	// 2. HTML a要素のhref属性から抽出
	for _, match := range extractAHrefRegex.FindAllStringSubmatch(bodyHTML, -1) {
		addID(match[2])
	}

	// 3. Markdown画像形式から抽出
	for _, match := range extractMarkdownImgRegex.FindAllStringSubmatch(bodyHTML, -1) {
		addID(match[1])
	}

	// 4. Markdownリンク形式から抽出（直前が'!'の場合は画像形式なのでスキップ）
	for _, loc := range extractMarkdownLinkRegex.FindAllStringSubmatchIndex(bodyHTML, -1) {
		matchStart := loc[0]
		if matchStart > 0 && bodyHTML[matchStart-1] == '!' {
			continue
		}
		id := bodyHTML[loc[2]:loc[3]]
		addID(id)
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
