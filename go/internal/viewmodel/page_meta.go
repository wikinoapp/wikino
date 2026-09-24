// Package viewmodelはビューモデル変換機能を提供します
package viewmodel

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/config"
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
}

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
		TwitterCard:  TwitterCardSummary,
		OGLocale:     localeToOGLocale(locale),
		AssetVersion: cfg.GetAssetVersion(),
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
