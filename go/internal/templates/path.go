package templates

import "fmt"

// Path はURLのパスを表す型です
type Path string

// SpacePath はスペースのパスを生成します
func SpacePath(identifier string) Path {
	return Path("/s/" + identifier)
}

// HomePath はホームのパスを生成します
func HomePath() Path {
	return Path("/home")
}

// TopicPath はトピックのパスを生成します
func TopicPath(spaceIdentifier string, topicNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/topics/%d", spaceIdentifier, topicNumber))
}

// NewPagePath はページ新規作成のパスを生成します
func NewPagePath(spaceIdentifier string, topicNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/topics/%d/pages/new", spaceIdentifier, topicNumber))
}

// PagePath はページのパスを生成します
func PagePath(spaceIdentifier string, pageNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d", spaceIdentifier, pageNumber))
}

// PageDraftPagePath は下書きページのパスを生成します
func PageDraftPagePath(spaceIdentifier string, pageNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d/draft_page", spaceIdentifier, pageNumber))
}

// SearchPath は検索のパスを生成します
func SearchPath() Path {
	return Path("/search")
}

// SearchPathWithSpaceFilter はスペースフィルター付きの検索パスを生成します
func SearchPathWithSpaceFilter(spaceIdentifier string) Path {
	return Path("/search?q=space:" + spaceIdentifier)
}

// ProfilePath はプロフィールのパスを生成します
func ProfilePath(atname string) Path {
	return Path("/@" + atname)
}

// SignInPath はサインインのパスを生成します
func SignInPath() Path {
	return Path("/sign_in")
}

// PageBacklinkListPath はバックリンク一覧のパスを生成します
func PageBacklinkListPath(spaceIdentifier string, pageNumber int32, linkedPageNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d/links/%d/backlink_list", spaceIdentifier, pageNumber, linkedPageNumber))
}

// SidebarJoinedTopicsPath はサイドバーの参加中トピック一覧のパスを生成します
func SidebarJoinedTopicsPath() Path {
	return Path("/sidebar/joined_topics")
}
