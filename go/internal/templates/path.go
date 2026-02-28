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

// PagePath はページのパスを生成します
func PagePath(spaceIdentifier string, pageNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d", spaceIdentifier, pageNumber))
}

// GoPagePath はGo版のページのパスを生成します
func GoPagePath(spaceIdentifier string, pageNumber int32) Path {
	return Path(fmt.Sprintf("/go/s/%s/pages/%d", spaceIdentifier, pageNumber))
}

// GoPageLinkListPath はGo版のページリンク一覧のパスを生成します
func GoPageLinkListPath(spaceIdentifier string, pageNumber int32) Path {
	return Path(fmt.Sprintf("/go/s/%s/pages/%d/link_list", spaceIdentifier, pageNumber))
}
