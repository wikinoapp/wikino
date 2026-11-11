// Package viewmodelはビューモデル変換機能を提供します
package viewmodel

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/darkmode"
	"github.com/wikinoapp/wikino/go/internal/i18n"
)

// PageMetaはページのメタ情報を保持します
type PageMeta struct {
	Title                  string          // ページタイトル (<title>タグ、og:title、twitter:title用)
	Description            string          // ページ説明 (description、og:description、twitter:description用)
	OGType                 string          // og:typeの値 ("website", "article"など)
	OGURL                  string          // og:urlの値
	OGImage                string          // og:imageの値
	OGImageWidth           int             // og:image:widthの値。0のときは出力しない (画像ごとに寸法が変わるカバー画像など)
	OGImageHeight          int             // og:image:heightの値。0のときは出力しない
	TwitterCard            string          // twitter:cardの値 ("summary"、"summary_large_image")
	OGLocale               string          // og:localeの値 ("ja_JP", "en_US"など)
	AssetVersion           string          // CSSやJSのバージョン (キャッシュバスティング用)
	CurrentSpaceIdentifier SpaceIdentifier // 現在のスペース識別子。グローバルホットキーの検索URL (`/search?q=space:{identifier}`) をdefault.templで組み立てるために使用する。スペース外画面では空文字
	// PreconnectOriginsはそのページが確実にリクエストする第三者オリジンの一覧。それぞれを
	// <link rel="preconnect"> として出力し、リクエスト自体が見つかる前にブラウザが接続を
	// 済ませられるようにする。宣言するのは実際にリクエストするページだけ。preconnectは
	// それ自体がそのオリジンへの接続になり、利用者のIPアドレスを第三者に送るためである。
	PreconnectOrigins []string
	// Feedsは、画面に関係するフィードの一覧。それぞれを <link rel="alternate" type="application/atom+xml"> として
	// 並べた順に出力し、画面のURLを貼るだけでフィードリーダーが見つけられるようにする (autodiscovery)。
	// リーダーは先頭のフィードを選ぶことが多いため、画面に最も近いフィードを先に並べる
	Feeds []FeedLink
	// FeatureScriptは、画面がmain.jsのあとに読み込む機能のエントリ。ゼロ値のときは読み込まない。
	// 機能のエントリは同じモジュールを含みうるため、1つの画面で読むのは1つまでにする
	FeatureScript FeatureScript
	// DarkModeEnabledは、閲覧者にダークモードのフィーチャーフラグが有効かどうか。
	// trueのときだけ、OSの配色がダークならダークで表示するための宣言とスクリプトを <head> に出す
	DarkModeEnabled bool
}

// FeatureScriptは、main.jsとは別にバンドルした機能のエントリ (web/<名前>.js)。
// 名前を外から作れないようにして、存在しないファイルを指せないようにしている
type FeatureScript struct {
	name string
}

// 機能のエントリ。web/ のエントリとpackage.jsonのbuild:jsに対応する
var (
	// FeatureScriptEditorは、ページの編集画面のエディタとZenモード・編集履歴の差分の色付け
	FeatureScriptEditor = FeatureScript{name: "editor"}
	// FeatureScriptMarkdownDiffは、差分の色付け
	FeatureScriptMarkdownDiff = FeatureScript{name: "markdown-diff"}
)

// Pathはエントリの配信パスを返す。ゼロ値のときは空文字列を返す
func (s FeatureScript) Path() string {
	if s.name == "" {
		return ""
	}
	return "/static/js/" + s.name + ".js"
}

// FeedLinkは画面の <head> で告知するフィード
type FeedLink struct {
	Title string // フィードの <title> と同じ文字列
	URL   string // フィードの絶対URL
}

// turnstileOriginはcomponents.Turnstileがウィジェットのスクリプトを読み込むオリジン。
const turnstileOrigin = "https://challenges.cloudflare.com"

// twitter:cardの値。
const (
	TwitterCardSummary           = "summary"
	TwitterCardSummaryLargeImage = "summary_large_image"
)

// localeToOGLocaleはlocale ("ja", "en"など) をOGPのlocale形式 ("ja_JP", "en_US"など) に変換します
func localeToOGLocale(locale string) string {
	switch locale {
	case i18n.LangEn:
		return "en_US"
	default:
		return "ja_JP"
	}
}

// DefaultPageMetaはデフォルトのメタ情報を返します
// DetectLanguage()で検出された言語に応じて、タイトルと説明が自動的に切り替わります
// Titleには自動的に " | Wikino" サフィックスが付加されます
func DefaultPageMeta(ctx context.Context, cfg *config.Config) PageMeta {
	ogImageURL := cfg.AppURL() + "/static/images/og-image.png"
	title := i18n.T(ctx, "default_title") + " | Wikino"
	locale := i18n.GetLocale(ctx)
	return PageMeta{
		Title:       title,
		Description: i18n.T(ctx, "default_description"),
		OGType:      "website",
		OGURL:       "",
		OGImage:     ogImageURL,
		// 既定のOGP画像はサイト共通のロゴのため、Xでは小さなサムネイルで出す。ページ固有の
		// 画像を出す画面だけが大きなカード (summary_large_image) に上書きする。
		TwitterCard:     TwitterCardSummary,
		OGLocale:        localeToOGLocale(locale),
		AssetVersion:    cfg.GetAssetVersion(),
		DarkModeEnabled: darkmode.IsEnabled(ctx),
	}
}

// SetTitleはタイトルを設定します (" | Wikino" サフィックス付き)
// 通常のページで使用します
func (p *PageMeta) SetTitle(ctx context.Context, titleKey string) {
	p.Title = i18n.T(ctx, titleKey) + " | Wikino"
}

// SetTitleWithoutSuffixはタイトルを設定します (サフィックスなし)
// スペース配下のページなど、" | Wikino" サフィックスが不要なページで使用します
func (p *PageMeta) SetTitleWithoutSuffix(ctx context.Context, titleKey string, data ...map[string]any) {
	p.Title = i18n.T(ctx, titleKey, data...)
}

// AddTurnstilePreconnectはTurnstileのウィジェットを描画するページで、そのオリジンを
// preconnectの対象として宣言します。components.TurnstileはsiteKeyがあるときだけ
// ウィジェットとスクリプトを描画するため、siteKeyが空ならヒントも出しません。
func (p *PageMeta) AddTurnstilePreconnect(siteKey string) {
	if siteKey == "" {
		return
	}
	p.PreconnectOrigins = append(p.PreconnectOrigins, turnstileOrigin)
}
