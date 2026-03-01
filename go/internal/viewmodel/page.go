package viewmodel

import (
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

// newPageFromModel はmodel.PageからPageビューモデルを生成します
func newPageFromModel(pg *model.Page) Page {
	var title string
	if pg.Title != nil {
		title = *pg.Title
	}
	return Page{
		Title:  title,
		Number: int32(pg.Number),
	}
}
