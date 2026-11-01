// Package legaldocは、利用規約・API利用規約・プライバシーポリシーの言語版と、それぞれを固定する
// URLと本文を置くWikinoのページの対応を表として持つ。
//
// 本文はスペース `wikino` のページに置き、改定はページの編集と公開で行う。固定URLは本文のページへ
// 転送するためのもので、OpenAPI記述やOAuthのメタデータのように、機械向けの記述にも載せる。
// 画面のリンクは転送を挟まないよう、固定URLではなく本文のページのパスを直接指す。
//
// 本パッケージはアプリケーション内の他のパッケージに依存しないleafに保つ (depguardで強制)。
package legaldoc

// Keyは文書の種類を表す
type Key string

const (
	KeyTerms         Key = "terms"
	KeyAPITerms      Key = "api_terms"
	KeyPrivacyPolicy Key = "privacy_policy"
)

// 文書の言語。internal/i18nの言語コードと同じ値にする
const (
	LangJa = "ja"
	LangEn = "en"
)

// Documentは文書の1つの言語版を表す
type Document struct {
	Key  Key
	Lang string
	// Pathは文書を固定するURLのパス。本文のページ (PagePath) へ301で転送する
	Path string
	// PagePathは本文を置くWikinoのページのパス。固定URLと同じオリジンにある
	PagePath string
}

// documentsは文書の言語版の一覧表。英語版は日本語版の翻訳である
var documents = []Document{
	{Key: KeyTerms, Lang: LangJa, Path: "/terms", PagePath: "/s/wikino/pages/41"},
	{Key: KeyAPITerms, Lang: LangJa, Path: "/terms/api", PagePath: "/s/wikino/pages/323"},
	{Key: KeyPrivacyPolicy, Lang: LangJa, Path: "/privacy", PagePath: "/s/wikino/pages/42"},
	{Key: KeyTerms, Lang: LangEn, Path: "/en/terms", PagePath: "/s/wikino/pages/325"},
	{Key: KeyAPITerms, Lang: LangEn, Path: "/en/terms/api", PagePath: "/s/wikino/pages/326"},
	{Key: KeyPrivacyPolicy, Lang: LangEn, Path: "/en/privacy", PagePath: "/s/wikino/pages/328"},
}

// Lookupは固定URLのパスから文書の言語版を引く。一覧表に無いパスではfalseを返す
func Lookup(path string) (Document, bool) {
	for _, doc := range documents {
		if doc.Path == path {
			return doc, true
		}
	}
	return Document{}, false
}

// Pathsは一覧表の全文書の固定URLのパスを返す。ルーティングとリバースプロキシの振り分けの登録に使う
func Paths() []string {
	paths := make([]string, len(documents))
	for i, doc := range documents {
		paths[i] = doc.Path
	}
	return paths
}

// Pathは文書の正本 (日本語版) の固定URLのパスを返す。OpenAPI記述やOAuthのメタデータのように、
// 読み手の表示言語が分からない機械向けの記述に使う。一覧表に無い文書はプログラムの誤りのためpanicする
func Path(key Key) string {
	doc, ok := localizedVersion(key, LangJa)
	if !ok {
		panic("legaldoc: 一覧表に無い文書です: " + string(key))
	}
	return doc.Path
}

// localizedVersionは、表示言語localeで読む文書の言語版を返す。日本語なら日本語版、
// それ以外は英語版を返す。一覧表に無い文書ではfalseを返す
func localizedVersion(key Key, locale string) (Document, bool) {
	lang := LangEn
	if locale == LangJa {
		lang = LangJa
	}
	for _, doc := range documents {
		if doc.Key == key && doc.Lang == lang {
			return doc, true
		}
	}
	return Document{}, false
}

// PagePathは、表示言語localeで読む文書の本文のページのパスを返す。画面のリンクに使う。
// 一覧表に無い文書はプログラムの誤りのため、空文字列ではなくpanicで気づけるようにする
func PagePath(key Key, locale string) string {
	doc, ok := localizedVersion(key, locale)
	if !ok {
		panic("legaldoc: 一覧表に無い文書です: " + string(key))
	}
	return doc.PagePath
}
