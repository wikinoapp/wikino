package viewmodel

import (
	"context"
	"fmt"
	"strings"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// Pageはテンプレートで表示するページ情報です
type Page struct {
	Title        string
	Body         string
	Number       int32
	ShowingDraft bool
}

// NewPageForEditは編集画面用のPageを生成します。
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

// NewPageFromFormInputはバリデーションエラー時にフォームの入力値を保持したPageを生成します
func NewPageFromFormInput(title string, body string, number model.PageNumber) Page {
	return Page{
		Title:  title,
		Body:   body,
		Number: int32(number),
	}
}

// AutofocusTitleはタイトル入力欄にオートフォーカスすべきかを返します
func (p Page) AutofocusTitle() bool {
	return p.Title == ""
}

// PageForShowはページ表示画面に描画するページ情報。BodyHTMLはページ公開時に
// レンダリング・サニタイズ済みの本文のため、テンプレートでは生のHTMLとして展開する。
type PageForShow struct {
	title    string
	BodyHTML string
	Number   int32

	// ogImageAttachmentIDはog:imageタグが指してよいアイキャッチ画像の添付ファイル。ページが
	// アイキャッチ画像を持たない場合や、その画像をプレビューとして出してはいけない場合は空になる
	// (NewPageForShowを参照)。
	ogImageAttachmentID string
}

// NewPageForShowはmodel.Pageからページ表示画面用のViewModelを生成する。
// featuredImageAttachmentはページのアイキャッチ画像で、持たない場合はnil。
func NewPageForShow(pg *model.Page, featuredImageAttachment *model.Attachment) PageForShow {
	var title string
	if pg.Title != nil {
		title = *pg.Title
	}

	return PageForShow{
		title:               title,
		BodyHTML:            pg.BodyHTML,
		Number:              int32(pg.Number),
		ogImageAttachmentID: ogImageAttachmentID(featuredImageAttachment),
	}
}

// ogImageAttachmentIDはog:imageタグが指してよいアイキャッチ画像を選び、無い場合は空文字列を
// 返す。GIFのアイキャッチ画像は対象外とする。og:imageエンドポイントは静止画の1200x630 jpgを
// 配信するため、アニメーション画像を指すと画像の持ち味を失ったプレビューを宣伝することになる。
func ogImageAttachmentID(featuredImageAttachment *model.Attachment) string {
	if featuredImageAttachment == nil {
		return ""
	}
	if strings.HasSuffix(strings.ToLower(featuredImageAttachment.Filename), ".gif") {
		return ""
	}

	return string(featuredImageAttachment.ID)
}

// OGImageAttachmentIDはog:imageタグが指す添付ファイルのIDを返し、対象が無い場合は空文字列を
// 返す。呼び出し元はそのときサイト共通の既定OGP画像を保つ。
func (p PageForShow) OGImageAttachmentID() string {
	return p.ogImageAttachmentID
}

// DisplayTitleはページの表示用タイトルを返し、未設定の場合はローカライズされた「無題」へ
// フォールバックする。
func (p PageForShow) DisplayTitle(ctx context.Context) string {
	if p.title != "" {
		return p.title
	}
	return i18n.T(ctx, "page_show_untitled")
}

// metaDescriptionMaxLengthは生成するmeta descriptionの上限を120文字にする。検索エンジンは
// これより長いスニペットを切り詰める。本文の言語はページごとに分からないため、英語で許容される
// 150文字程度ではなく、推奨範囲のうち短い側 (日本語) に合わせている。
const metaDescriptionMaxLength = 120

// MetaDescriptionはmeta descriptionタグ用に本文のプレーンテキスト要約を組み立てる。本文に
// テキストが無い場合 (未公開のページや画像だけのページ) は空文字列を返し、呼び出し元が空の
// descriptionを出さずサイト共通の既定値を保てるようにする。
func (p PageForShow) MetaDescription() string {
	// 上限より1文字多く要求する。収まる本文と切り詰めが要る本文はこれで区別でき、長いページの
	// 本文全体をリクエストのたびに取り出さずに済む。
	text := markup.PlainText(p.BodyHTML, metaDescriptionMaxLength+1)
	if text == "" {
		return ""
	}

	runes := []rune(text)
	if len(runes) <= metaDescriptionMaxLength {
		return text
	}

	// 省略記号を付ける前に末尾の空白を落とす。切り取り位置がmarkup.PlainTextの出すブロック
	// 境界の半角スペースと重なることがあり、そのままだと "テキスト …" のように見えてしまう。
	return strings.TrimRight(string(runes[:metaDescriptionMaxLength-1]), " ") + "…"
}

// PageForMoveはページ移動画面用のページ情報です
type PageForMove struct {
	Title  string
	Number int32
}

// NewPageForMoveはmodel.Pageからページ移動画面用のViewModelを生成します
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

// CardLinkPageはリンク一覧・バックリンク一覧で使用するページカードの表示データです
type CardLinkPage struct {
	Title        string
	Number       int32
	Topic        *Topic
	Pinned       bool
	CardImageURL string
	CanEdit      bool
}

// NewCardLinkPageはmodel.Pageとトピック情報からカード用のビューモデルを生成します
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
