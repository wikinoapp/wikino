// Package viewmodel はビューモデル変換機能を提供します
package viewmodel

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/i18n"
)

// PageMeta はページのメタ情報を保持します
type PageMeta struct {
	Title        string // ページタイトル（<title>タグ、og:title、twitter:title用）
	Description  string // ページ説明（description、og:description、twitter:description用）
	OGType       string // og:typeの値（"website", "article"など）
	OGURL        string // og:urlの値
	OGImage      string // og:imageの値
	OGLocale     string // og:localeの値（"ja_JP", "en_US"など）
	AssetVersion string // CSSやJSのバージョン（キャッシュバスティング用）
}

// localeToOGLocale はlocale（"ja", "en"など）をOGPのlocale形式（"ja_JP", "en_US"など）に変換します
func localeToOGLocale(locale string) string {
	switch locale {
	case i18n.LangEn:
		return "en_US"
	default:
		return "ja_JP"
	}
}

// DefaultPageMeta はデフォルトのメタ情報を返します
// DetectLanguage()で検出された言語に応じて、タイトルと説明が自動的に切り替わります
// Titleには自動的に " | Wikino" サフィックスが付加されます
func DefaultPageMeta(ctx context.Context, cfg *config.Config) PageMeta {
	ogImageURL := cfg.AppURL() + "/static/images/og-image.png"
	title := i18n.T(ctx, "default_title") + " | Wikino"
	locale := i18n.GetLocale(ctx)
	return PageMeta{
		Title:        title,
		Description:  i18n.T(ctx, "default_description"),
		OGType:       "website",
		OGURL:        "",
		OGImage:      ogImageURL,
		OGLocale:     localeToOGLocale(locale),
		AssetVersion: cfg.GetAssetVersion(),
	}
}

// SetTitle はタイトルを設定します（" | Wikino" サフィックス付き）
// 通常のページで使用します
func (p *PageMeta) SetTitle(ctx context.Context, titleKey string) {
	p.Title = i18n.T(ctx, titleKey) + " | Wikino"
}

// SetTitleWithoutSuffix はタイトルを設定します（サフィックスなし）
// トップページなど、サフィックスが不要なページで使用します
func (p *PageMeta) SetTitleWithoutSuffix(ctx context.Context, titleKey string) {
	p.Title = i18n.T(ctx, titleKey)
}
