package templates

// PageName identifies the current page for presentation components such as the global
// navigation's active-link state.
//
// [Ja] PageName はグローバルナビのアクティブリンク状態など、Presentation コンポーネントで
// 現在のページを識別するための型。
type PageName string

const (
	PageNameHome                   PageName = "home"
	PageNameWelcome                PageName = "welcome"
	PageNameSearch                 PageName = "search"
	PageNameProfile                PageName = "profile"
	PageNamePageEdit               PageName = "page_edit"
	PageNamePageMove               PageName = "page_move"
	PageNameDraftPageIndex         PageName = "draft_page_index"
	PageNameSpaceShow              PageName = "space_show"
	PageNameTopicShow              PageName = "topic_show"
	PageNameSuggestionIndex        PageName = "suggestion_index"
	PageNameSuggestionShow         PageName = "suggestion_show"
	PageNameSuggestionNew          PageName = "suggestion_new"
	PageNameSuggestionChanges      PageName = "suggestion_changes"
	PageNameSuggestionEdit         PageName = "suggestion_edit"
	PageNameSuggestionPageNew      PageName = "suggestion_page_new"
	PageNameSuggestionPageEditShow PageName = "suggestion_page_edit_show"
	PageNameSuggestionCommentEdit  PageName = "suggestion_comment_edit"
)
