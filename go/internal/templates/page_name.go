package templates

// PageName はサイドバーのアクティブ状態を判定するためのページ名を表す型です
type PageName string

const (
	PageNameHome     PageName = "home"
	PageNameWelcome  PageName = "welcome"
	PageNameSearch   PageName = "search"
	PageNameProfile  PageName = "profile"
	PageNamePageEdit PageName = "page_edit"
	PageNamePageMove PageName = "page_move"
)
