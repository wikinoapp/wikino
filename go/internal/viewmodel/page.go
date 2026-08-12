package viewmodel

import (
	"context"
	"fmt"
	"strings"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// Page はテンプレートで表示するページ情報です
type Page struct {
	Title        string
	Body         string
	Number       int32
	ShowingDraft bool
}

// NewPageForEdit は編集画面用のPageを生成します。
// 下書きが存在する場合は下書きの内容を、存在しない場合は公開版の内容を使用します。
func NewPageForEdit(pg *model.Page, draftPage *model.DraftPage) Page {
	if draftPage != nil {
		var title string
		if draftPage.Title != nil {
			title = *draftPage.Title
		}
		return Page{
			Title:        title,
			Body:         draftPage.Body,
			Number:       int32(pg.Number),
			ShowingDraft: true,
		}
	}

	var title string
	if pg.Title != nil {
		title = *pg.Title
	}
	return Page{
		Title:  title,
		Body:   pg.Body,
		Number: int32(pg.Number),
	}
}

// NewPageFromFormInput はバリデーションエラー時にフォームの入力値を保持したPageを生成します
func NewPageFromFormInput(title string, body string, number model.PageNumber) Page {
	return Page{
		Title:  title,
		Body:   body,
		Number: int32(number),
	}
}

// AutofocusTitle はタイトル入力欄にオートフォーカスすべきかを返します
func (p Page) AutofocusTitle() bool {
	return p.Title == ""
}

// PageForShow is the page data rendered on the page detail screen. BodyHTML is the rendered and
// sanitized body produced when the page was published, so the template expands it as raw HTML.
//
// [Ja] PageForShow はページ表示画面に描画するページ情報。BodyHTML はページ公開時に
// レンダリング・サニタイズ済みの本文のため、テンプレートでは生の HTML として展開する。
type PageForShow struct {
	title    string
	BodyHTML string
	Number   int32
}

// NewPageForShow creates the ViewModel used by the page detail screen.
//
// [Ja] NewPageForShow は model.Page からページ表示画面用の ViewModel を生成する。
func NewPageForShow(pg *model.Page) PageForShow {
	var title string
	if pg.Title != nil {
		title = *pg.Title
	}

	return PageForShow{
		title:    title,
		BodyHTML: pg.BodyHTML,
		Number:   int32(pg.Number),
	}
}

// DisplayTitle returns the page's display title, falling back to the localized untitled label.
//
// [Ja] DisplayTitle はページの表示用タイトルを返し、未設定の場合はローカライズされた「無題」へ
// フォールバックする。
func (p PageForShow) DisplayTitle(ctx context.Context) string {
	if p.title != "" {
		return p.title
	}
	return i18n.T(ctx, "page_show_untitled")
}

// metaDescriptionMaxLength caps the generated meta description at 120 characters. Search engines
// truncate longer snippets, and the body language is unknown per page, so this stays within the
// shorter (Japanese) end of the recommended range rather than the ~150 characters English allows.
//
// [Ja] metaDescriptionMaxLength は生成する meta description の上限を 120 文字にする。検索エンジンは
// これより長いスニペットを切り詰める。本文の言語はページごとに分からないため、英語で許容される
// 150 文字程度ではなく、推奨範囲のうち短い側 (日本語) に合わせている。
const metaDescriptionMaxLength = 120

// MetaDescription builds a plain-text summary of the body for the meta description tag. It returns
// an empty string when the body has no text (an unpublished page, or one holding only images), so
// that the caller keeps the site-wide default rather than declaring an empty description.
//
// [Ja] MetaDescription は meta description タグ用に本文のプレーンテキスト要約を組み立てる。本文に
// テキストが無い場合 (未公開のページや画像だけのページ) は空文字列を返し、呼び出し元が空の
// description を出さずサイト共通の既定値を保てるようにする。
func (p PageForShow) MetaDescription() string {
	// Ask for one rune past the limit: that is enough to tell a body that fits from one that has to
	// be truncated, without extracting the whole body of a long page on every request.
	//
	// [Ja] 上限より 1 文字多く要求する。収まる本文と切り詰めが要る本文はこれで区別でき、長いページの
	// 本文全体をリクエストのたびに取り出さずに済む。
	text := markup.PlainText(p.BodyHTML, metaDescriptionMaxLength+1)
	if text == "" {
		return ""
	}

	runes := []rune(text)
	if len(runes) <= metaDescriptionMaxLength {
		return text
	}

	// Drop a trailing space before appending the ellipsis. The cut can land right on the single
	// space markup.PlainText emits between two blocks, which would otherwise read as "text …".
	//
	// [Ja] 省略記号を付ける前に末尾の空白を落とす。切り取り位置が markup.PlainText の出すブロック
	// 境界の半角スペースと重なることがあり、そのままだと "テキスト …" のように見えてしまう。
	return strings.TrimRight(string(runes[:metaDescriptionMaxLength-1]), " ") + "…"
}

// PageForMove はページ移動画面用のページ情報です
type PageForMove struct {
	Title  string
	Number int32
}

// NewPageForMove はmodel.Pageからページ移動画面用のViewModelを生成します
func NewPageForMove(pg *model.Page) PageForMove {
	var title string
	if pg.Title != nil {
		title = *pg.Title
	}
	return PageForMove{
		Title:  title,
		Number: int32(pg.Number),
	}
}

// CardLinkPage はリンク一覧・バックリンク一覧で使用するページカードの表示データです
type CardLinkPage struct {
	Title        string
	Number       int32
	Topic        *Topic
	Pinned       bool
	CardImageURL string
	Primary      bool
	CanEdit      bool
}

// NewCardLinkPage はmodel.Pageとトピック情報からカード用のビューモデルを生成します
func NewCardLinkPage(pg *model.Page, topicMap map[model.TopicID]*model.Topic) CardLinkPage {
	var title string
	if pg.Title != nil {
		title = *pg.Title
	}

	var topicVM *Topic
	if topic, ok := topicMap[pg.TopicID]; ok {
		t := NewTopic(topic)
		topicVM = &t
	}

	var cardImageURL string
	if pg.FeaturedImageAttachmentID != nil {
		cardImageURL = fmt.Sprintf("/attachments/%s", *pg.FeaturedImageAttachmentID)
	}

	return CardLinkPage{
		Title:        title,
		Number:       int32(pg.Number),
		Topic:        topicVM,
		Pinned:       pg.PinnedAt != nil,
		CardImageURL: cardImageURL,
	}
}
