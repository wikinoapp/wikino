package templates

import (
	"fmt"

	"github.com/a-h/templ"
)

// SpacePath はスペースのパスを生成します
func SpacePath(identifier string) templ.SafeURL {
	return templ.SafeURL("/s/" + identifier)
}

// TopicPath はトピックのパスを生成します
func TopicPath(spaceIdentifier string, topicNumber int32) string {
	return fmt.Sprintf("/s/%s/topics/%d", spaceIdentifier, topicNumber)
}
