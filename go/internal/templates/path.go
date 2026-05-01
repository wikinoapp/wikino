package templates

import (
	"fmt"
	"net/url"

	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Path はURLのパスを表す型です
type Path string

// SpacePath はスペースのパスを生成します
func SpacePath(identifier viewmodel.SpaceIdentifier) Path {
	return Path("/s/" + string(identifier))
}

// HomePath はホームのパスを生成します
func HomePath() Path {
	return Path("/home")
}

// TopicPath はトピックのパスを生成します
func TopicPath(spaceIdentifier viewmodel.SpaceIdentifier, topicNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/topics/%d", spaceIdentifier, topicNumber))
}

// TopicSettingsPath はトピック設定のパスを生成します
func TopicSettingsPath(spaceIdentifier viewmodel.SpaceIdentifier, topicNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/topics/%d/settings", spaceIdentifier, topicNumber))
}

// NewPagePath はページ新規作成のパスを生成します
func NewPagePath(spaceIdentifier viewmodel.SpaceIdentifier, topicNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/topics/%d/pages/new", spaceIdentifier, topicNumber))
}

// PagePath はページのパスを生成します
func PagePath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber viewmodel.PageNumber) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d", spaceIdentifier, pageNumber))
}

// PageDraftPagePath は下書きページのパスを生成します
func PageDraftPagePath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d/draft_page", spaceIdentifier, pageNumber))
}

// PageDraftPageRevisionPath は下書きリビジョン手動保存のパスを生成します
func PageDraftPageRevisionPath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d/draft_page_revision", spaceIdentifier, pageNumber))
}

// SearchPath は検索のパスを生成します
func SearchPath() Path {
	return Path("/search")
}

// SearchPathWithSpaceFilter はスペースフィルター付きの検索パスを生成します
func SearchPathWithSpaceFilter(spaceIdentifier viewmodel.SpaceIdentifier) Path {
	return Path("/search?q=space:" + string(spaceIdentifier))
}

// SearchPathFor は現在のスペースに応じた検索パスを生成します
// スペース内ならスペースフィルター付き、スペース外なら素の `/search` を返します
func SearchPathFor(spaceIdentifier viewmodel.SpaceIdentifier) Path {
	if spaceIdentifier != "" {
		return SearchPathWithSpaceFilter(spaceIdentifier)
	}
	return SearchPath()
}

// ProfilePath はプロフィールのパスを生成します
func ProfilePath(atname string) Path {
	return Path("/@" + atname)
}

// SignInPath はサインインのパスを生成します
func SignInPath() Path {
	return Path("/sign_in")
}

// PageLinkListPath はリンク一覧のパスを生成します
func PageLinkListPath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d/link_list", spaceIdentifier, pageNumber))
}

// PageBacklinkListPath はバックリンク一覧のパスを生成します
func PageBacklinkListPath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber int32, linkedPageNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d/links/%d/backlink_list", spaceIdentifier, pageNumber, linkedPageNumber))
}

// PageBacklinksPath はページレベルのバックリンク一覧のパスを生成します
func PageBacklinksPath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d/backlinks", spaceIdentifier, pageNumber))
}

// PageEditPath はページ編集のパスを生成します
func PageEditPath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d/edit", spaceIdentifier, pageNumber))
}

// PageMovePath はページ移動のパスを生成します
func PageMovePath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d/move", spaceIdentifier, pageNumber))
}

// SuggestionListPath は編集提案一覧のパスを生成します
func SuggestionListPath(spaceIdentifier viewmodel.SpaceIdentifier, topicNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/topics/%d/suggestions", spaceIdentifier, topicNumber))
}

// SuggestionShowPath は編集提案詳細のパスを生成します
func SuggestionShowPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d", spaceIdentifier, suggestionNumber))
}

// SuggestionNewPath は編集提案作成のパスを生成します
func SuggestionNewPath(spaceIdentifier viewmodel.SpaceIdentifier, topicNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/topics/%d/suggestions/new", spaceIdentifier, topicNumber))
}

// SuggestionChangesPath は編集提案の変更差分のパスを生成します
func SuggestionChangesPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/changes", spaceIdentifier, suggestionNumber))
}

// SuggestionApplyPath は編集提案反映のパスを生成します
func SuggestionApplyPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/apply", spaceIdentifier, suggestionNumber))
}

// SuggestionClosePath は編集提案クローズのパスを生成します
func SuggestionClosePath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/close", spaceIdentifier, suggestionNumber))
}

// SuggestionCommentsPath は編集提案コメント作成のパスを生成します
func SuggestionCommentsPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/comments", spaceIdentifier, suggestionNumber))
}

// SuggestionPageEditsPath は編集提案ページの編集開始のパスを生成します
func SuggestionPageEditsPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/page_edits", spaceIdentifier, suggestionNumber))
}

// SuggestionPageEditShowPath は編集提案ページ編集の確認画面のパスを生成します
func SuggestionPageEditShowPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32, suggestionPageID string) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/page_edits/%s", spaceIdentifier, suggestionNumber, url.PathEscape(suggestionPageID)))
}

// SuggestionEditPath は編集提案編集のパスを生成します
func SuggestionEditPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/edit", spaceIdentifier, suggestionNumber))
}

// SuggestionCommentPath は編集提案コメントのパスを生成します
func SuggestionCommentPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32, commentNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/comments/%d", spaceIdentifier, suggestionNumber, commentNumber))
}

// SuggestionCommentEditPath は編集提案コメント編集のパスを生成します
func SuggestionCommentEditPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32, commentNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/comments/%d/edit", spaceIdentifier, suggestionNumber, commentNumber))
}

// SuggestionPagePath は編集提案ページのパスを生成します
func SuggestionPagePath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32, suggestionPageID string) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/suggestion_pages/%s", spaceIdentifier, suggestionNumber, url.PathEscape(suggestionPageID)))
}

// SuggestionPagesPath は編集提案ページ一覧のパスを生成します
func SuggestionPagesPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/suggestion_pages", spaceIdentifier, suggestionNumber))
}

// SuggestionPageNewPath は編集提案ページ追加のパスを生成します
func SuggestionPageNewPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/suggestion_pages/new", spaceIdentifier, suggestionNumber))
}

// DraftsPath は下書き一覧のパスを生成します
func DraftsPath() Path {
	return Path("/drafts")
}

// SidebarJoinedTopicsPath はサイドバーの参加中トピック一覧のパスを生成します
func SidebarJoinedTopicsPath() Path {
	return Path("/sidebar/joined_topics")
}

// SidebarDraftPagesPath はサイドバーの下書きページ一覧のパスを生成します
func SidebarDraftPagesPath() Path {
	return Path("/sidebar/draft_pages")
}
