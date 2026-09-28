package templates

// PageNameはグローバルナビのアクティブリンク状態など、Presentationコンポーネントで
// 現在のページを識別するための型。
type PageName string

const (
	PageNameHome                               PageName = "home"
	PageNameSearch                             PageName = "search"
	PageNameProfile                            PageName = "profile"
	PageNamePageShow                           PageName = "page_show"
	PageNamePageEdit                           PageName = "page_edit"
	PageNamePageMove                           PageName = "page_move"
	PageNameDraftPageIndex                     PageName = "draft_page_index"
	PageNameSpaceShow                          PageName = "space_show"
	PageNameSpaceSettings                      PageName = "space_settings"
	PageNameTopicShow                          PageName = "topic_show"
	PageNameTopicNew                           PageName = "topic_new"
	PageNameTopicSettingsGeneral               PageName = "topic_settings_general"
	PageNameSuggestionIndex                    PageName = "suggestion_index"
	PageNameSuggestionShow                     PageName = "suggestion_show"
	PageNameSuggestionNew                      PageName = "suggestion_new"
	PageNameSuggestionChanges                  PageName = "suggestion_changes"
	PageNameSuggestionEdit                     PageName = "suggestion_edit"
	PageNameSuggestionPageNew                  PageName = "suggestion_page_new"
	PageNameSuggestionPageEditShow             PageName = "suggestion_page_edit_show"
	PageNameSuggestionCommentEdit              PageName = "suggestion_comment_edit"
	PageNameExportNew                          PageName = "export_new"
	PageNameExportShow                         PageName = "export_show"
	PageNamePersonalAccessTokenIndex           PageName = "personal_access_token_index"
	PageNamePersonalAccessTokenNew             PageName = "personal_access_token_new"
	PageNamePersonalAccessTokenCreate          PageName = "personal_access_token_create"
	PageNameOAuthApplicationIndex              PageName = "oauth_application_index"
	PageNameOAuthApplicationNew                PageName = "oauth_application_new"
	PageNameOAuthApplicationCreate             PageName = "oauth_application_create"
	PageNameOAuthApplicationShow               PageName = "oauth_application_show"
	PageNameOAuthApplicationEdit               PageName = "oauth_application_edit"
	PageNameOAuthApplicationClientSecretCreate PageName = "oauth_application_client_secret_create"
	PageNameOAuthGrantIndex                    PageName = "oauth_grant_index"
)
