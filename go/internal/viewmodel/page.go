package viewmodel

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// Page はテンプレートで表示するページ情報です
type Page struct {
	Title  string
	Body   string
	Number int32
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
			Title:  title,
			Body:   draftPage.Body,
			Number: pg.Number,
		}
	}

	var title string
	if pg.Title != nil {
		title = *pg.Title
	}
	return Page{
		Title:  title,
		Body:   pg.Body,
		Number: pg.Number,
	}
}

// NewPageFromFormInput はバリデーションエラー時にフォームの入力値を保持したPageを生成します
func NewPageFromFormInput(title string, body string, number int32) Page {
	return Page{
		Title:  title,
		Body:   body,
		Number: number,
	}
}

// AutofocusTitle はタイトル入力欄にオートフォーカスすべきかを返します
func (p Page) AutofocusTitle() bool {
	return p.Title == ""
}
