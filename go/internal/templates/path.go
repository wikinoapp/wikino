package templates

import "fmt"

// Path はURLのパスを表す型です
type Path string

// SpacePath はスペースのパスを生成します
func SpacePath(identifier string) Path {
	return Path("/s/" + identifier)
}

// TopicPath はトピックのパスを生成します
func TopicPath(spaceIdentifier string, topicNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/topics/%d", spaceIdentifier, topicNumber))
}
