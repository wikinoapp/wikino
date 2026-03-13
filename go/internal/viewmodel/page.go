package viewmodel

import (
	"fmt"

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
